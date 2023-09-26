package fs

import (
	"io"
)

type FileSystem interface {
	Save(key, filePath, contentType string) (string, error)
	SaveFile(key string, buffer io.Reader, contentType string, contentLength int64) (string, error)
	DownloadVideoUrl(videoUrl string) (string, error)
}
