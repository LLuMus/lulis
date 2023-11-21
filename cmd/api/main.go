package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/llumus/lulis/internal/fs/s3"
	"github.com/llumus/lulis/internal/gpt/openai"
	"github.com/llumus/lulis/internal/mixer/replicate"
	"github.com/llumus/lulis/internal/queue/memory"
	"github.com/llumus/lulis/internal/stream/ffmpeg"
	"github.com/llumus/lulis/internal/tts/elevenlabs"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

var bannedWords = []string{
	"ass",
	"fuck",
	"suck",
	"bitch",
	"crap",
	"puta",
	"sex",
	"learning disability",
	"learning disabilities",
	"learning disorder",
	"learning disorders",
	"learning difficulties",
	"learning difficulty",
	"problemas de aprendizagem",
	"problema de aprendizagem",
	"transtorno de aprendizagem",
	"replace",
	"replaced",
	"replacing",
	"replaces",
	"substitua",
	"substituir",
	"substituindo",
	"\"",
	"“",
	"”",
	"’",
	"‘",
	"''",
	"``",
	"\\",
	"!*",
}

// healthCheckHandler is a simple HTTP handler function which writes a response used for cloud deploys
func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "OK")
}

// slotCount is a knob to control the number of files in the tmp folder and cache size for the workload
const slotCount = 128

// autoPlayInterval is a knob to control the interval between automatic video plays
const autoPlayInterval = 5 * time.Minute

// autoPlayRecurrentInterval is a knob to control the interval between automatic video plays
const autoPlayRecurrentInterval = 2 * time.Minute

// queuesThroughput is a knob to control the throughput of the queues, careful it consumes CPU 🔥
const queuesThroughput = 3 * time.Second

var (
	// mutex for thread-safe access to playedVideos
	mutex sync.Mutex

	// playedVideos to keep track of played videos
	playedVideos []string = make([]string, 0, slotCount)

	// messageTimer timer to send a random cached video
	messageTimer = time.NewTimer(autoPlayInterval)
)

func main() {
	var port = os.Getenv("PORT")
	var basePath = os.Getenv("BASE_PATH")
	var twitchChannelName = os.Getenv("TWITCH_CHANNEL_NAME")
	var twitchStreamKey = os.Getenv("TWITCH_STREAM_KEY")
	var twitchClientId = os.Getenv("TWITCH_CLIENT_ID")
	var openAiKey = os.Getenv("OPEN_AI_KEY")
	var elevenLabsKey = os.Getenv("ELEVEN_LABS_KEY")
	var elevenLabsVoiceId = os.Getenv("ELEVEN_LABS_VOICE_ID")
	var replicateKey = os.Getenv("REPLICATE_KEY")
	var awsBucket = os.Getenv("AWS_BUCKET_NAME")
	var awsBaseUrl = os.Getenv("AWS_BUCKET_BASE_URL")
	var faceVideoUrl = os.Getenv("FACE_VIDEO_URL")

	gpt := openai.NewOpenAI(openAiKey)
	fs := s3.NewFileSystem(awsBucket, basePath, slotCount)
	tts := elevenlabs.NewElevenLabs(elevenLabsKey, basePath, elevenLabsVoiceId, http.DefaultClient, fs)
	stream := ffmpeg.NewStream(twitchStreamKey, filepath.Join(basePath, "tmp", "playlist.txt"))
	mixer := replicate.NewMixer(replicateKey, awsBaseUrl, faceVideoUrl, http.DefaultClient, fs)

	// Create a server instance
	server := &http.Server{Addr: ":" + port}
	http.HandleFunc("/", healthCheckHandler)

	go func() {
		fmt.Println("Server is running on port " + port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("ListenAndServe error: %v\n", err)
		}
	}()

	go func() {
		log.Println("Starting stream...")
		err := stream.StartStream()
		if err != nil {
			log.Println("Error starting stream:", err)
		}
		for err != nil {
			log.Println("Error starting stream:", err)
			err = stream.StartStream()
			time.Sleep(queuesThroughput)
		}
	}()

	videoQueue := memory.NewQueue()
	go func() {
		for {
			video, ok := videoQueue.Dequeue()
			if ok {
				log.Infof("Video from queue: %s", video)

				err := stream.PlayLatest(video)
				if err != nil {
					log.Errorf("Error switching video: %v", err)
					continue
				}

				addPlayedVideo(video)
			}
			time.Sleep(queuesThroughput)
		}
	}()

	client := twitch.NewClient(twitchChannelName, twitchClientId)
	msgQueue := memory.NewQueue()
	go func() {
		ctx := context.Background()
		for {
			message, ok := msgQueue.Dequeue()
			if ok {
				log.Debugf("Message from queue: %s", message)

				var bannedWordDetected string
				for _, word := range bannedWords {
					if strings.Contains(message, word) {
						bannedWordDetected = word
						break
					}
				}

				if bannedWordDetected != "" {
					log.Warnf("Banned word detected: %s", bannedWordDetected)
					continue
				}

				answer, err := gpt.GenerateResponse(ctx, message)
				if err != nil {
					log.Println("Error generating response:", err)
					continue
				}

				log.Infof("Generated response for: %s", answer)
				log.Infof("Generating audio for: %s", answer)

				fsKey, err := tts.GenerateAudio(ctx, answer)
				if err != nil {
					log.Println("Error generating audio:", err)
					continue
				}

				client.Say(twitchChannelName, "Almost ready...")

				log.Infof("Generated audio: %s", fsKey)
				log.Infof("Generating lip sync for: %s", answer)

				videoLocalPath, err := mixer.GenerateLipSyncVideo(ctx, fsKey)
				if err != nil {
					log.Println("Error generating video:", err)
					continue
				}

				client.Say(twitchChannelName, "Anytime now...")

				log.Infof("Generated video: %s", videoLocalPath)
				log.Infof("Sending video to queue: %s", videoLocalPath)

				videoQueue.Enqueue(videoLocalPath)
				messageTimer.Reset(autoPlayInterval)
			}

			time.Sleep(queuesThroughput)
		}
	}()

	go func() {
		for {
			select {
			case <-messageTimer.C:
				// Timer expired, send a random cached video
				if len(playedVideos) > 0 {
					randomIndex := rand.Intn(len(playedVideos))
					randomVideo := playedVideos[randomIndex]
					videoQueue.Enqueue(randomVideo)
				}
				messageTimer.Reset(autoPlayRecurrentInterval)
			}
		}
	}()

	client.Join(twitchChannelName)
	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		log.Infof("Message received: %s", message.Message)
		if strings.HasPrefix(message.Message, "Lula, ") {
			log.Infof("Message to the queue: %s", message.Message)
			msgQueue.Enqueue(message.Message + " - " + message.User.Name)
			client.Say(message.Channel, "We are processing your request "+message.User.Name+", please wait a minute or two.")
		} else {
			log.Infof("Message not for me: %s", message.Message)
			client.Say(message.Channel, "To talk to Lula, a message have to start with 'Lula, '")
		}
	})

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}

// addPlayedVideo to add a video to the playedVideos slice
func addPlayedVideo(videoPath string) {
	mutex.Lock()
	defer mutex.Unlock()

	// Check if we need to remove the oldest video
	if len(playedVideos) >= 128 {
		playedVideos = playedVideos[1:]
	}

	playedVideos = append(playedVideos, videoPath)
}
