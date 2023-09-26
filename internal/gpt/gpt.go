package gpt

import "context"

type GPT interface {
	GenerateResponse(ctx context.Context, question string) (string, error)
}
