package s3

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type FileSystem struct {
	awsBucket    string
	basePath     string
	maxFileSlots int
}

func NewFileSystem(awsBucket string, basePath string, maxFileSlots int) *FileSystem {
	return &FileSystem{
		awsBucket:    awsBucket,
		basePath:     basePath,
		maxFileSlots: maxFileSlots,
	}
}

func (s *FileSystem) Save(key, filePath, contentType string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil
	}

	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return "", err
	}

	return s.SaveFile(key, f, contentType, fileInfo.Size())
}

func (s *FileSystem) SaveFile(key string, originalReader io.Reader, contentType string, contentLength int64) (string, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return "", err
	}

	var b = make([]byte, contentLength)

	_, err = originalReader.Read(b)
	if err != nil {
		return "", err
	}

	if contentLength <= 0 {
		return "", fmt.Errorf("empty bytes file buffer %s", key)
	}

	if contentType == "" {
		contentType = http.DetectContentType(b)
	}

	var (
		uploader    = s3.New(awsSession)
		uploadInput = &s3.PutObjectInput{
			Bucket:        aws.String(s.awsBucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(b),
			ContentType:   aws.String(contentType),
			ContentLength: aws.Int64(contentLength),
		}
	)

	_, err = uploader.PutObject(uploadInput)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", key), nil
}

func (s *FileSystem) DownloadVideoUrl(videoUrl string) (string, error) {
	result, err := http.Get(videoUrl)
	if err != nil {
		return "", err
	}
	defer result.Body.Close()

	// Generate a random number from 0 to maxFileSlots-1
	random := strconv.Itoa(rand.Intn(s.maxFileSlots))
	finalPath := filepath.Join(s.basePath, "tmp", "latest"+random+".mp4")

	f, err := os.Create(finalPath)
	if err != nil {
		return "", err
	}

	defer f.Close()

	// write file
	_, err = io.Copy(f, result.Body)
	if err != nil {
		return "", err
	}

	return finalPath, nil
}
