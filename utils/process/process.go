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
Package process provides process processing-related functions.
*/
package process

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// DaemonizeProcess create daemon process
//
// Returns:
//   - error: success(nil), failure(error)
func DaemonizeProcess() error {
	// If the ppid of the current process is 1,
	// it is already a daemon process
	if os.Getppid() != 1 {
		// Get absolute path
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %s", err)
		}

		// Create child process
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		cmd.Stdin = nil
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Start daemon process (child process)
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to start daemon process: %s", err)
		}

		// Terminate parent process
		os.Exit(0)
	}

	return nil
}

// IsProcessRun verify that the process is operational
//
// Parameters:
//   - pid: process id
//
// Returns:
//   - bool: running(true), stop(false)
func IsProcessRun(pid int) bool {
	// Find process by pid
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send Signal 0 to verify that the actual process is running
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// SendSignal sends signal to process
//
// Parameters:
//   - pid: process id
//   - sig: signal
//
// Returns:
//   - error: success(nil), failure(error)
func SendSignal(pid int, sig syscall.Signal) error {
	// Find process by pid
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %s", err)
	}

	// Send signal to process
	err = proc.Signal(sig)
	if err != nil {
		return fmt.Errorf("failed to send signal: %s", err)
	}

	return nil
}
