package gpt

import "context"

type GPT interface {
	GenerateResponse(ctx context.Context, question string) (string, error)
	GenerateQuestion(ctx context.Context) (string, error)
}
