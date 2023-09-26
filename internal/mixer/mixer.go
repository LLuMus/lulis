package mixer

import "context"

type Mixer interface {
	GenerateLipSyncVideo(ctx context.Context, fsKey string) (string, error)
}
