// Copyright 2024 JongHoon Shim and The log_manager Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

/*
Package goroutine stably manages the life cycle of a number of goroutines.
*/
package goroutine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GoroutineManager goroutine management structure
type GoroutineManager struct {
	mu           sync.Mutex
	parentWG     sync.WaitGroup
	parentCtx    context.Context
	parentCancel context.CancelFunc
	tasks        map[string]*taskWrapper
}

// taskWrapper goroutine task structure
type taskWrapper struct {
	childWG     sync.WaitGroup
	childCtx    context.Context
	childCancel context.CancelFunc
	task        func(ctx context.Context)
}

// NewGoroutineManager create goroutine manager.
//
// Returns:
//   - *GoroutineManager: goroutine manager structure
func NewGoroutineManager() *GoroutineManager {
	// Generating parent context for termination of the entire goroutine
	ctx, cancel := context.WithCancel(context.Background())
	// Returns the initialized GoroutineManager
	return &GoroutineManager{
		parentCtx:    ctx,
		parentCancel: cancel,
		tasks:        make(map[string]*taskWrapper),
	}
}

// AddTask register the goroutine task.
//
// Parameters:
//   - name: task name (key)
//   - task: function (value)
func (gm *GoroutineManager) AddTask(name string, task func(ctx context.Context)) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Gernerating child context for termination of the individual goroutine
	ctx, cancel := context.WithCancel(gm.parentCtx)
	// Set goroutine task
	gm.tasks[name] = &taskWrapper{
		childCtx:    ctx,
		childCancel: cancel,
		task:        task,
	}
}

// RemoveTask terminate and remove a task.
//
// Parameters:
//   - name: task name
//   - timeout: wait group timeout
//
// Returns:
//   - error: success(nil), timeout occurred(error)
func (gm *GoroutineManager) RemoveTask(name string, timeout time.Duration) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if t, exists := gm.tasks[name]; exists {
		t.childCancel()
		if WaitGroupWithTimeout(&t.childWG, timeout) != WaitSuccess {
			return fmt.Errorf("goroutine was not terminated within the specified timeout"+
				"(goroutine: %s, timeout: %.2fsec)", name, timeout.Seconds())
		}
		delete(gm.tasks, name)
	}

	return nil
}

// StartAll run all goroutines registered in the job.
func (gm *GoroutineManager) StartAll() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	for _, t := range gm.tasks {
		gm.parentWG.Add(1)
		t.childWG.Add(1)
		// Hand over the pointer to the go function,
		// but the corresponding pointer address value is maintained
		go func(tw *taskWrapper) {
			defer func() {
				tw.childWG.Done()
				gm.parentWG.Done()
			}()

			// Run a job
			tw.task(tw.childCtx)
		}(t)
	}
}

// StopAll shut down all goroutines that are being worked on.
//
// Parameters:
//   - timeout: wait group timeout
//
// Returns:
//   - error: success(nil), timeout occurred(error)
func (gm *GoroutineManager) StopAll(timeout time.Duration) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.parentCancel()
	if WaitGroupWithTimeout(&gm.parentWG, timeout) != WaitSuccess {
		return fmt.Errorf("goroutines were not terminated within the specified timeout"+
			"(timeout: %.2fsec)", timeout.Seconds())
	}
	return nil
}

// Start run a goroutine.
//
// Parameters:
//   - name: task name
//
// Returns:
//   - error: success(nil), failure(error)
func (gm *GoroutineManager) Start(name string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Verify that the job exists
	t, exists := gm.tasks[name]
	if !exists {
		return fmt.Errorf("task does not exist (%s)", name)
	}

	gm.parentWG.Add(1)
	t.childWG.Add(1)
	go func() {
		defer func() {
			t.childWG.Done()
			gm.parentWG.Done()
		}()

		// Run a job
		t.task(t.childCtx)
	}()

	return nil
}

// Stop terminate a goroutine.
//
// Parameters:
//   - name: task name
//   - timeout: wait group timeout
//
// Returns:
//   - error: success(nil), timeout occurred(error)
func (gm *GoroutineManager) Stop(name string, timeout time.Duration) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if t, exists := gm.tasks[name]; exists {
		t.childCancel()
		if WaitGroupWithTimeout(&t.childWG, timeout) != WaitSuccess {
			return fmt.Errorf("goroutine was not terminated within the specified timeout"+
				"(goroutine: %s, timeout: %.2fsec)", name, timeout.Seconds())
		}
	}
	return nil
}
