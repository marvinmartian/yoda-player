package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bogem/id3v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tcolgate/mp3"
)

var (
	mu                sync.Mutex
	lastPlayedID      string
	lastStartTime     time.Time
	timeout           = 3500 * time.Millisecond
	timeoutTimer      *time.Timer
	jsonData          map[string]map[string]interface{}
	isPlaying         bool
	allow_play_resume bool = false
	canStartTrack     bool = true
	canPlayTimer      *time.Timer
	canPlayTimeout    = 5 * time.Second
)

var (
	// Define a Prometheus CounterVec to track the number of plays for each track ID and name.
	playsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "track_plays_total",
			Help: "Total number of plays for each track ID.",
		},
		[]string{"track_id", "track_name"},
	)

	// Define a Prometheus Counter for a podcast plays.
	podcastCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "podcast_plays",
			Help: "A counter for the number of times a particular podcast is played.",
		},
		[]string{"episode_title"},
	)

	// Define a Prometheus Counter for tracking the number of errors.
	playErrorsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "play_errors_total",
			Help: "Total number of play errors.",
		},
	)

	trackDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "track_play_duration",
			Help: "Track the total amount of time played in seconds",
		},
		[]string{"track_id"},
	)
)

type PostData struct {
	ID string `json:"id"`
}

type mp3Data struct {
	EpisodeTitle string
	Author       string
	PodcastTitle string
}

type Track struct {
	Duration  float64 // Duration of the track in seconds (you can change the data type based on your requirements).
	TrackName string  // Name of the track.
}

// Message represents the structure of the JSON message
type SocketMessage struct {
	Text string `json:"text"`
}

var tracks = make(map[string]Track)

func init() {
	// Register the Prometheus metrics.
	prometheus.MustRegister(
		playsCounter,
		podcastCounter,
		playErrorsCounter,
		trackDuration,
	)
}

// Function to set the start time
func setStartTime() time.Time {
	startTime := time.Now()
	fmt.Println("Start time set:", startTime)
	return startTime
}

// Function to get the duration since the start time
func durationSinceStart(startTime time.Time) time.Duration {
	return time.Since(startTime)
}

func getFramecount(track string) (float64, int) {
	t := 0.0
	frameCount := 0

	r, err := os.Open(track)
	if err != nil {
		fmt.Println(err)
		return 0, 0
	}
	defer r.Close()

	d := mp3.NewDecoder(r)
	var f mp3.Frame
	skipped := 0

	for {

		if err := d.Decode(&f, &skipped); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			return 0, 0
		}
		// fmt.Println(f.Header().BitRate())
		t = t + f.Duration().Seconds()
		frameCount++
	}

	return t, frameCount
}

func readID3(filepath string) mp3Data {
	fmt.Println(filepath)
	tag, err := id3v2.Open(filepath, id3v2.Options{Parse: true})
	if err != nil {
		log.Fatal("Error while opening mp3 file: ", err)
	}
	defer tag.Close()

	// Create an instance of mp3Data and populate its fields from the ID3 tag
	data := mp3Data{
		EpisodeTitle: tag.Title(),
		Author:       tag.Artist(),
		PodcastTitle: tag.Album(),
	}

	// fmt.Println(tag.Artist())
	// fmt.Println(data)
	return data
}

func playMP3(filePath string, offset int, currentID string) {
	stopMP3()
	if isPlaying && currentID == lastPlayedID {
		fmt.Println("This track is already playing.")
		return
	} else if isPlaying {
		fmt.Println("Music is already playing, but starting new track.")
		stopMP3()
	}

	isPlaying = true

	cmd := exec.Command("mpg123", "-q", "-b", "1024", fmt.Sprintf("-k %d", offset), filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		defer func() {
			isPlaying = false
		}()
		err := cmd.Run()
		if err != nil {
			fmt.Println("Error playing MP3:", err)
			playErrorsCounter.Inc()
			// os.Exit(1)
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
		isPlaying = false
	}
}

func updateTrackPlayInfo(track string, duration float64) {
	// fmt.Printf("Update Track Play Info\n")

	// Add/Update the duration of the existing track
	tracks[track] = Track{Duration: duration, TrackName: track}
	// fmt.Printf("Track %s updated with new duration: %f seconds\n", track, duration)
}

func getTrackPlayInfo(track string) Track {
	// fmt.Printf("Get Track Play Info\n")
	// Attempt to get the track from the map
	trackInfo, found := tracks[track]

	// Check if the track was found
	if !found {
		fmt.Printf("Track not found: %s\n", track)
	}
	return trackInfo
}

func getTrackFromID(id string) (Track, bool) {
	if data, ok := jsonData[id]; ok {
		filePath, _ := data["file"].(string)
		offset := data["offset"].(float64)
		return Track{Duration: offset, TrackName: filePath}, true
	} else {
		return Track{}, false
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
	// fmt.Println("Received a POST request to /play with data:", currentID)

	mu.Lock()
	defer mu.Unlock()

	// Check if something is playing now
	if isPlaying {
		if currentID != lastPlayedID {
			// If something is already playing, and it's not the same as the incoming ID
			fmt.Printf("Stopping the previous track (ID: %s) and starting the new track (ID: %s)\n", lastPlayedID, currentID)
			stopMP3()
			lastPlayedID = currentID
		} else {
			// If the same ID is requested again
			// fmt.Printf("Received the same ID again (ID: %s). Current track remains unchanged.\n", currentID)
			durationSince := durationSinceStart(lastStartTime)
			// fmt.Println("durationSinceStart:", durationSince.Seconds())
			updateTrackPlayInfo(currentID, durationSince.Seconds())
			trackDuration.WithLabelValues(currentID).Add(durationSince.Seconds())
			// trackInfo, ok := getTrackFromID(currentID)
			// if ok {
			// 	length, frameCount := getFramecount(trackInfo.TrackName)
			// 	fmt.Printf("Track: %s - Duration: %f ", trackInfo.TrackName, trackInfo.Duration)
			// 	fmt.Printf("Frames: %d - Length: %f ", frameCount, length)
			// }
			// Reset the timer
			if timeoutTimer != nil {
				timeoutTimer.Stop()
			}
			// Start or reset the timer
			if canPlayTimer != nil {
				// fmt.Println("reset canPlayTimer")
				canPlayTimer.Reset(canPlayTimeout)
			}

		}
	} else {

		// Start or reset the timer
		if canPlayTimer != nil {
			// fmt.Println("reset canPlayTimer")
			canPlayTimer.Reset(canPlayTimeout)
		} else {
			canPlayTimer = time.AfterFunc(canPlayTimeout, func() {
				mu.Lock()
				defer mu.Unlock()

				// Allow playing again
				fmt.Println("canPlayTimer timeout reached. Allowing play again")
				canStartTrack = true
				stopMP3()
			})
		}

		if canStartTrack {

			// If nothing is playing, start playing the song
			fmt.Printf("Starting to play the track (ID: %s)\n", currentID)
			lastPlayedID = currentID

			// Start or reset the timer
			if timeoutTimer != nil {
				timeoutTimer.Stop()
			}
			timeoutTimer = time.AfterFunc(timeout, func() {
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
					if isPlaying {
						fmt.Println("")
					}
					trackPath := filePath
					id3_info := readID3(trackPath)
					lastStartTime = setStartTime()
					go func() {
						// Call the function with the track path
						duration, count := getFramecount(trackPath)

						// Print the results
						fmt.Printf("Duration=%.2f seconds, Frame count=%d\n", duration, count)

						playsCounter.WithLabelValues(currentID, id3_info.EpisodeTitle).Inc()
						podcastCounter.WithLabelValues(id3_info.PodcastTitle).Inc()

						padded_offset := 0
						if allow_play_resume {
							playInfo := getTrackPlayInfo(currentID)
							// fmt.Printf("getTrackPlayInfo.Duration: %f \n", playInfo.Duration)
							frames_per_second := count / int(duration)
							// fmt.Println(frames_per_second)
							padded_offset = frames_per_second * int(playInfo.Duration)
							// fmt.Println(padded_offset)

						}

						playMP3(trackPath, int(offset)+padded_offset, currentID)
						canStartTrack = false
					}()
					// duration, frames := getFramecount(trackPath)
					// fmt.Printf("frames: %d - duration %f seconds", frames, duration)
					// Increment the playsCounter metric for the current track ID.

				} else {
					fmt.Println("Offset field not found in JSON for ID:", currentID)
				}
			} else {
				fmt.Println("ID not found in JSON:", currentID)
			}
		}
	}

	fmt.Fprintln(w, "Data received and printed to console") // Respond to the client
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request
		// fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
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

	// Define a new route for Prometheus metrics
	router.Handle("/metrics", promhttp.Handler())

	// Create a handler chain with the request logger
	chain := http.Handler(logRequest(router))

	// Start the web server on port 3001
	fmt.Println("Listening on port 3001...")
	err = http.ListenAndServe(":3001", chain)
	if err != nil {
		fmt.Println("Error:", err)
	}

}
