package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/marvinmartian/yoda-player/internal/player"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	mu             sync.Mutex
	lastPlayedID   string
	lastStartTime  time.Time
	timeout        = 3500 * time.Millisecond
	timeoutTimer   *time.Timer
	jsonData       map[string]map[string]interface{}
	isPlaying      bool
	canStartTrack  bool = true
	canPlayTimer   *time.Timer
	canPlayTimeout = 5 * time.Second
)

var (
	// Define a Prometheus CounterVec to track the number of plays for each track ID and name.
	playsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "track_plays_total",
			Help: "Total number of plays for each track ID.",
		},
		[]string{"track_id", "track_name", "track_artist"},
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

type Track struct {
	Duration  float64 // Duration of the track in seconds (you can change the data type based on your requirements).
	TrackName string  // Name of the track.
}

// Message represents the structure of the JSON message
type SocketMessage struct {
	Text string `json:"text"`
}

var tracks = make(map[string]Track)

var mp3JsonPath string

func init() {
	// Register the Prometheus metrics.
	prometheus.MustRegister(
		playsCounter,
		podcastCounter,
		playErrorsCounter,
		trackDuration,
	)

	flag.StringVar(&mp3JsonPath, "mp3File", "../mp3.json", "Path to the MP3 JSON file")
	flag.Parse()
}

// Function to set the start time
func setStartTime() time.Time {
	startTime := time.Now()
	fmt.Println("Start time set:", startTime)
	return startTime
}

// Function to get the duration since the start time
// func durationSinceStart(startTime time.Time, playedDuration float64) time.Duration {
// 	// Add the played duration to the original startTime
// 	newStartTime := startTime.Add(time.Duration(playedDuration * float64(time.Second)))

// 	// Calculate the duration since the updated startTime
// 	return time.Since(newStartTime)
// }

func durationSinceStart(startTime time.Time) time.Duration {
	return time.Since(startTime)
}

func playMP3(player *player.Player, filePath string, offset int, currentID string) {
	stopMP3(player)
	if isPlaying && currentID == lastPlayedID {
		fmt.Println("This track is already playing.")
		return
	} else if isPlaying {
		fmt.Println("Music is already playing, but starting new track.")
		stopMP3(player)
	}

	isPlaying = true

	player.Clear()
	player.AddToPlaylist(filePath)

	player.Play()
	if offset > 0 {
		seekErr := player.Seek(offset)
		if seekErr != nil {
			fmt.Println(seekErr)
		}
	}
}

func stopMP3(player *player.Player) {
	if isPlaying {
		// If music is playing, stop it
		player.Stop()
		player.Clear()
		isPlaying = false
		lastPlayedID = "0"
	}
}

func updateTrackPlayInfo(track string, duration float64) {
	// fmt.Printf("Update Track Play Info\n")

	// Add/Update the duration of the existing track
	tracks[track] = Track{Duration: duration, TrackName: track}
	// fmt.Printf("Track %s updated with new duration: %f seconds\n", track, duration)
}

func playHandler(player *player.Player) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			playStats, _ := player.Status()
			if currentID != lastPlayedID {
				// If something is already playing, and it's not the same as the incoming ID
				fmt.Printf("Stopping the previous track (ID: %s) and starting the new track (ID: %s)\n", lastPlayedID, currentID)
				stopMP3(player)
				lastPlayedID = currentID
			} else {
				// If the same ID is requested again
				// fmt.Printf("Received the same ID again (ID: %s). Current track remains unchanged.\n", currentID)

				elapsedStr, ok := playStats["elapsed"]
				elapsed := 0.0
				if ok {
					elapsed, err = strconv.ParseFloat(elapsedStr, 64)
					if err != nil {
						fmt.Println("Error converting elapsed to float64:", err)
						return
					}
					fmt.Printf("Elapsed: %f\n", elapsed)
				} else {
					fmt.Println("Elapsed not found")
					fmt.Println(playStats)
				}
				durationSince := durationSinceStart(lastStartTime)
				// fmt.Println("durationSinceStart:", durationSince.Seconds())
				updateTrackPlayInfo(currentID, durationSince.Seconds())
				trackDuration.WithLabelValues(currentID).Add(durationSince.Seconds())
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
					stopMP3(player)
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
					stopMP3(player)
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
						// id3_info := mp3.ReadID3(trackPath, player)
						lastStartTime = setStartTime()
						go func() {
							// Call the function with the track path

							// duration := 2342.32
							// count := 232
							// duration, count := getFramecount(trackPath)

							// Print the results
							// fmt.Printf("Duration=%.2f seconds, Frame count=%d\n", duration, count)

							padded_offset := 0

							// if allow_play_resume {
							// 	playInfo := getTrackPlayInfo(currentID)
							// 	// fmt.Printf("getTrackPlayInfo.Duration: %f \n", playInfo.Duration)
							// 	frames_per_second := count / int(duration)
							// 	// fmt.Println(frames_per_second)
							// 	padded_offset = frames_per_second * int(playInfo.Duration)
							// 	// fmt.Println(padded_offset)

							// }

							playMP3(player, trackPath, int(offset)+padded_offset, currentID)

							currentSong, _ := player.CurrentSong()
							fmt.Println(currentSong.Name)
							fmt.Println(currentSong.Album)
							fmt.Println(currentSong.Artist)

							// duration := playStats["duration"]
							// fmt.Println("-- Duration:", playStats["duration"])

							playsCounter.WithLabelValues(currentID, currentSong.Album, currentSong.Artist).Inc()
							podcastCounter.WithLabelValues(currentSong.Name).Inc()
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
	jsonDataFile, err := os.ReadFile(mp3JsonPath)
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

	mpdAddress := "localhost:6600"
	// mpdPort := 6600
	mpdPassword := ""
	mpdConfig := player.MPDConfig{}
	mpdConfig.MpdAddress = &mpdAddress
	// mpdConfig.Port = &mpdPort
	mpdConfig.MpdPassword = &mpdPassword

	mpdPlayer, err := player.NewPlayer(&mpdConfig)
	if err != nil {
		fmt.Println("MPD Player Error:", err)
	}

	fmt.Println(mpdPlayer.Status())

	// Define the route and handler for /play
	router.HandleFunc("/play", playHandler(mpdPlayer))

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
