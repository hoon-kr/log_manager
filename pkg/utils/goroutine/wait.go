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

package goroutine

import (
	"context"
	"sync"
	"time"
)

type WaitError int

const (
	WaitSuccess WaitError = iota
	WaitTimeout
	WaitInvalidParam
)

// WaitCancelWithTimeout wait context cancel with timeout.
//
// Parameters:
//   - ctx: context
//   - timeout: context wait timeout
//
// Returns:
//   - WaitError: received cancel signal(WaitSuccess), failure(WaitError)
func WaitCancelWithTimeout(ctx context.Context, timeout time.Duration) WaitError {
	// If the timeout is less than 0, it waits for cancellation indefinitely
	if timeout < 0 {
		<-ctx.Done()
		return WaitSuccess
	}

	select {
	case <-ctx.Done():
		// Received cancel
		return WaitSuccess
	case <-time.After(timeout):
		// Timeout occurred
		return WaitTimeout
	}
}

// WaitGroupWithTimeout wait goroutine termination with timeout.
//
// Parameters:
//   - wg: wait group
//   - timeout: wait group timeout
//
// Returns:
//   - WaitError: exit goroutine normally(WaitSuccess), failure(WaitError)
func WaitGroupWithTimeout(wg *sync.WaitGroup, timeout time.Duration) WaitError {
	if wg == nil {
		return WaitInvalidParam
	}

	// End goroutine job waiting indefinitely if timeout is less than 0
	if timeout < 0 {
		wg.Wait()
		return WaitSuccess
	}

	// Create a channel to signal the end of an operation
	done := make(chan struct{})

	// Wait for the end of the goroutine operation
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		// goroutine normal shutdown
		return WaitSuccess
	case <-time.After(timeout):
		// Timeout occurred
		return WaitTimeout
	}
}
