package ffmpeg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type FFProbeOutput struct {
	Format Format `json:"format"`
}

type Format struct {
	Duration string `json:"duration"`
}

type Stream struct {
	playlistPath     string
	tempPlaylistPath string
	currentCmd       *exec.Cmd
	stdout           io.ReadCloser
	reader           *bufio.Reader
	twitchStreamKey  string
	isAwaitingFinish bool
}

var log = logrus.New()

func NewStream(twitchStreamKey string, playlistPath string) *Stream {
	if err := copyAssetsToTmp(playlistPath); err != nil {
		log.Fatalf("Error copying assets to tmp: %s", err)
	}
	return &Stream{
		playlistPath:     playlistPath,
		tempPlaylistPath: strings.Replace(playlistPath, "playlist.txt", "temp_playlist.txt", 1),
		twitchStreamKey:  twitchStreamKey,
	}
}

func (s *Stream) StartStream() error {
	s.currentCmd = exec.Command("ffmpeg",
		"-re",
		"-loglevel", "verbose",
		"-stream_loop", "-1",
		"-f", "concat",
		"-safe", "0",
		"-i", s.playlistPath,
		"-pix_fmt", "yuv420p",
		"-x264-params", "keyint=48:min-keyint=48:scenecut=-1",
		"-b:v", "4500k",
		"-b:a", "128k",
		"-ar", "44100",
		"-acodec", "aac",
		"-vcodec", "libx264",
		"-preset", "faster",
		"-f", "flv",
		"rtmp://live.twitch.tv/app/"+s.twitchStreamKey)

	var err error
	s.stdout, err = s.currentCmd.StderrPipe()
	if err != nil {
		return err
	}

	s.reader = bufio.NewReader(s.stdout)

	err = s.currentCmd.Start()
	if err != nil {
		return err
	}

	go s.printStdOut(s.stdout)
	return s.currentCmd.Wait()
}

func (s *Stream) StopStream() error {
	return s.currentCmd.Process.Kill()
}

func (s *Stream) PlayLatest(path string) error {
	if err := s.replaceSecondLine(s.tempPlaylistPath, "file '"+filepath.Base(path)+"'"); err != nil {
		return err
	}

	log.Infof("Replaced playlist should play now, will wait for finish %s", path)

	duration, err := s.getVideoDuration(path)
	if err != nil {
		log.Errorf("Error getting video duration: %s", err)
		duration = 10
	}

	roundDuration := time.Duration((duration + 2) * float64(time.Second))
	log.Infof("Waiting for %s", roundDuration)

	time.Sleep(roundDuration)

	log.Infof("Done waiting for %s putting loop back", roundDuration)
	if err := s.replaceSecondLine(s.tempPlaylistPath, "file 'loop.mp4'"); err != nil {
		return err
	}

	return nil
}

func (s *Stream) printStdOut(stdout io.ReadCloser) {
	r := bufio.NewReader(stdout)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err != io.EOF {
				log.Errorf("Error reading stderr: %s", err)
			}
			break
		}

		if strings.Contains(string(line), " Reinit context") {
			s.isAwaitingFinish = false
			log.Infof("got reinit context: %s", line)
		} else {
			log.Infof("line: %s", line)
		}
	}
}

func (s *Stream) getVideoDuration(filepath string) (float64, error) {
	// Run ffprobe to get video details in JSON format
	cmd := exec.Command("ffprobe", "-v", "error", "-show_format", "-of", "json", filepath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	// Parse the JSON output
	var probeOutput FFProbeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return 0, err
	}

	// Convert duration string to float64 and return
	duration, err := strconv.ParseFloat(probeOutput.Format.Duration, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func (s *Stream) replaceSecondLine(filename, newLine string) error {
	// Read the file into memory
	input, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")

	// Ensure there's at least two lines
	if len(lines) < 2 {
		return fmt.Errorf("file %s does not have at least two lines", filename)
	}

	// Replace the second line
	lines[1] = newLine

	// Write the modified lines back to the file
	output := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(output), 0644)
	return err
}

func copyAssetsToTmp(playlistPath string) error {
	var assetsDir = strings.Replace(playlistPath, "playlist.txt", "../assets", 1)
	var tmpDir = strings.Replace(playlistPath, "/playlist.txt", "", 1)

	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		files, err := ioutil.ReadDir(assetsDir)
		if err != nil {
			return err
		}

		for _, file := range files {
			sourceFile := filepath.Join(assetsDir, file.Name())
			destFile := filepath.Join(tmpDir, file.Name())

			// Perform the file copy
			err := copyFile(sourceFile, destFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
