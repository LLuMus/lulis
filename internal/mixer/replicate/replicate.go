package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/llumus/lulis/internal/fs"
	"github.com/sirupsen/logrus"
)

const (
	apiUrl       = "https://api.replicate.com/v1/predictions"
	version      = "8d65e3f4f4298520e079198b493c25adfc43c058ffec924f2aefc8010ed25eef"
	pollInterval = 3 * time.Second
	timeout      = 60 * time.Second
)

var log = logrus.New()

type Payload struct {
	Version    string                 `json:"version"`
	Input      map[string]interface{} `json:"input"`
	IsTraining bool                   `json:"is_training"`
}

type Response struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error"`
	Output string `json:"output,omitempty"`
}

type Mixer struct {
	apiKey        string
	baseUrl       string
	finalVideoUrl string
	client        *http.Client
	fs            fs.FileSystem
}

func NewMixer(apiKey string, baseUrl string, finalVideoUrl string, client *http.Client, fs fs.FileSystem) *Mixer {
	return &Mixer{
		apiKey:        apiKey,
		baseUrl:       baseUrl,
		finalVideoUrl: finalVideoUrl,
		client:        client,
		fs:            fs,
	}
}

func (m *Mixer) GenerateLipSyncVideo(_ context.Context, fsKey string) (string, error) {
	body, err := json.Marshal(&Payload{
		Version: version,
		Input: map[string]interface{}{
			"fps":           25,
			"face":          m.finalVideoUrl,
			"pads":          "0 10 0 0",
			"audio":         m.baseUrl + fsKey,
			"smooth":        true,
			"resize_factor": 1,
		},
		IsTraining: false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return "", fmt.Errorf("error: %s", string(body))
	}

	var result Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if result.ID != "" {
		generatedVideoUrl, err := m.waitJobCompleteOrFail(result.ID)
		if err != nil {
			return "", err
		}

		return m.fs.DownloadVideoUrl(generatedVideoUrl)
	}

	return "", nil
}

func (m *Mixer) waitJobCompleteOrFail(jobID string) (string, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := checkStatus(m.apiKey, jobID)
			if err != nil {
				return resp, err
			}
			if resp != "" {
				return resp, nil
			}
		case <-timer.C:
			return "", fmt.Errorf("timed out waiting for job to complete")
		}
	}
}

func checkStatus(apiToken string, jobID string) (string, error) {
	url := apiUrl + "/" + jobID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result Response
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if result.Status == "succeeded" && result.Output != "" {
		return result.Output, nil
	}

	log.Infof("Job %s status: %s output %s", jobID, result.Status, result.Output)

	return "", nil
}
