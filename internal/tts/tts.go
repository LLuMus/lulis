package tts

import "context"

type TTS interface {
	GenerateAudio(ctx context.Context, text string) (string, error)
}
