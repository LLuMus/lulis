package stream

type Stream interface {
	StartStream() error
	StopStream() error
	PlayLatest(path string) error
}
