package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	previousID string
	timeout    = 8 * time.Second
	timer      *time.Timer
	jsonData   map[string]map[string]interface{}
	isPlaying  bool
)

type PostData struct {
	ID string `json:"id"`
}

func playMP3(filePath string, offset int) {
	if isPlaying {
		fmt.Println("Music is already playing.")
		return
	}

	isPlaying = true

	cmd := exec.Command("mpg123", fmt.Sprintf("-k %d", offset), filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		defer func() {
			isPlaying = false
		}()
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error playing MP3:", err)
		}
	}()
}

func stopMP3() {
	if isPlaying {
		// If music is playing, stop it
		err := exec.Command("pkill", "mpg123").Run()
		if err != nil {
			fmt.Println("Error stopping MP3:", err)
		}
	}
}

func playHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the request body into a PostData struct
	var postData PostData
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&postData)
	if err != nil {
		http.Error(w, "Failed to decode request body", http.StatusInternalServerError)
		return
	}

	currentID := postData.ID
	fmt.Println("Received a POST request to /play with data:", currentID)

	mu.Lock()
	defer mu.Unlock()

	if currentID == previousID {
		fmt.Printf("Received the same ID again. Current ID: %s, Previous ID: %s\n", currentID, previousID)
		// Reset the timer
		if timer != nil {
			timer.Stop()
		}
	} else {
		previousID = currentID
	}

	// Start or reset the timer
	if timer != nil {
		timer.Stop()
	}
	timer = time.AfterFunc(timeout, func() {
		mu.Lock()
		defer mu.Unlock()

		// Stop playing if the timeout is reached
		fmt.Println("Timeout reached. Stopping play...")
		stopMP3()
	})

	// Check if the currentID exists in the JSON data
	if data, ok := jsonData[currentID]; ok {
		filePath, _ := data["file"].(string)
		offset, ok := data["offset"].(float64)
		if ok {
			playMP3("../"+filePath, int(offset))
		} else {
			fmt.Println("Offset field not found in JSON for ID:", currentID)
		}
	} else {
		fmt.Println("ID not found in JSON:", currentID)
	}

	fmt.Fprintln(w, "Data received and printed to console") // Respond to the client
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request
		fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Load JSON data from the "mp3.json" file
	jsonDataFile, err := os.ReadFile("../mp3.json")
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}

	if err := json.Unmarshal(jsonDataFile, &jsonData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Create a router
	router := http.NewServeMux()

	// Define the route and handler for /play
	router.HandleFunc("/play", playHandler)

	// Create a handler chain with the request logger
	chain := http.Handler(logRequest(router))

	// Start the web server on port 3001
	fmt.Println("Listening on port 3001...")
	err = http.ListenAndServe(":3001", chain)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
