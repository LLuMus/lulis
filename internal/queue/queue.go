package queue

type Queue interface {
	Enqueue(message string)
	Dequeue() (string, bool)
}
