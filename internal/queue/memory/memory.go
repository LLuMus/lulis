package memory

import (
	"fmt"
	"sync"
)

type Queue struct {
	data []string
	mu   sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		data: make([]string, 0),
	}
}

func (q *Queue) Enqueue(message string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.data) < 10 {
		q.data = append(q.data, message)
	} else {
		fmt.Println("Queue is full, discarding message:", message)
	}
}

func (q *Queue) Dequeue() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.data) == 0 {
		return "", false
	}

	message := q.data[0]
	q.data = q.data[1:]
	return message, true
}
