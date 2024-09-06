package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Task struct {
	Body     string `json:"body"`
	Priority string `json:"priority"` // "low", "medium", "high"
}

// A single queue with priority levels
type Queue struct {
	High   []Task
	Medium []Task
	Low    []Task
	mu     sync.Mutex
}

// Add a task to the queue based on its priority
func (q *Queue) Push(task Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Default priority is "low"
	switch task.Priority {
	case "high":
		q.High = append(q.High, task)
	case "medium":
		q.Medium = append(q.Medium, task)
	default:
		q.Low = append(q.Low, task)
	}
}

// Pop a task based on priority (high > medium > low)
func (q *Queue) Pop() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.High) > 0 {
		task := q.High[0]
		q.High = q.High[1:]
		return &task
	}
	if len(q.Medium) > 0 {
		task := q.Medium[0]
		q.Medium = q.Medium[1:]
		return &task
	}
	if len(q.Low) > 0 {
		task := q.Low[0]
		q.Low = q.Low[1:]
		return &task
	}
	return nil // No tasks available
}

// Central queue manager that manages multiple named queues
type QueueManager struct {
	Queues map[string]*Queue
	mu     sync.RWMutex
}

func NewQueueManager() *QueueManager {
	return &QueueManager{
		Queues: make(map[string]*Queue),
	}
}

// Get or create a queue by name
func (qm *QueueManager) GetQueue(queueName string) *Queue {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if _, exists := qm.Queues[queueName]; !exists {
		qm.Queues[queueName] = &Queue{}
	}
	return qm.Queues[queueName]
}

var manager = NewQueueManager()

// HTTP handler to push a task to the queue
func pushTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Extract the queue name from query params
	queueName := r.URL.Query().Get("queue")
	if queueName == "" {
		http.Error(w, "Queue query param required", http.StatusBadRequest)
		return
	}

	// Default to low priority if not set
	if task.Priority == "" {
		task.Priority = "low"
	}

	// Add task to the named queue
	queue := manager.GetQueue(queueName)
	queue.Push(task)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task added to queue '%s' with priority: %s\n", queueName, task.Priority)
}

// HTTP handler to pop a task from the queue
func popTaskHandler(w http.ResponseWriter, r *http.Request) {
	queueName := r.URL.Query().Get("queue")
	if queueName == "" {
		http.Error(w, "Queue query param required", http.StatusBadRequest)
		return
	}

	// Pop task from the named queue
	queue := manager.GetQueue(queueName)
	task := queue.Pop()
	if task == nil {
		http.Error(w, "No tasks available in queue", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Main function to start the HTTP server
func main() {
	http.HandleFunc("/push", pushTaskHandler) // Handle pushing tasks
	http.HandleFunc("/pop", popTaskHandler)   // Handle popping tasks

	log.Println("Starting queue service on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
