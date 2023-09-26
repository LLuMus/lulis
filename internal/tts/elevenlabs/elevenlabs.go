package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/llumus/lulis/internal/fs"
)

const baseUrl = "https://api.elevenlabs.io/v1/text-to-speech/"

type VoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style"`
}

type Payload struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceSettings VoiceSettings `json:"voice_settings"`
}

type ElevenLabs struct {
	basePath string
	voiceId  string
	client   *http.Client
	apiKey   string
	fs       fs.FileSystem
}

func NewElevenLabs(apiKey string, basePath string, voiceId string, client *http.Client, fs fs.FileSystem) *ElevenLabs {
	return &ElevenLabs{
		basePath: basePath,
		voiceId:  voiceId,
		client:   client,
		apiKey:   apiKey,
		fs:       fs,
	}
}

func (e *ElevenLabs) GenerateAudio(_ context.Context, text string) (string, error) {
	var newFileName = uuid.NewString() + ".mp3"
	payload := Payload{
		Text:    text,
		ModelID: "eleven_multilingual_v2",
		VoiceSettings: VoiceSettings{
			Stability:       0.7,
			SimilarityBoost: 0.4,
			Style:           0.2,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseUrl+e.voiceId, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("accept", "audio/mpeg")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return e.fs.SaveFile(newFileName, bytes.NewReader(body), "audio/mpeg", int64(len(body)))
}
