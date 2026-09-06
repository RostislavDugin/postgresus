package task_cancellation

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Registry struct {
	mutex       sync.Mutex
	cancelTasks map[uuid.UUID]context.CancelFunc
}

func NewRegistry() *Registry {
	return &Registry{
		cancelTasks: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (r *Registry) RegisterTask(taskID uuid.UUID, cancelTask context.CancelFunc) {
	r.mutex.Lock()
	r.cancelTasks[taskID] = cancelTask
	r.mutex.Unlock()
}

func (r *Registry) UnregisterTask(taskID uuid.UUID) {
	r.mutex.Lock()
	delete(r.cancelTasks, taskID)
	r.mutex.Unlock()
}

func (r *Registry) CancelRegisteredTask(taskID uuid.UUID) {
	r.mutex.Lock()
	cancelTask, isRegistered := r.cancelTasks[taskID]
	if isRegistered {
		delete(r.cancelTasks, taskID)
	}
	r.mutex.Unlock()

	if !isRegistered {
		return
	}

	cancelTask()
}
