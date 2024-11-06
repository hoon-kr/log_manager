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
Package server controls log_manager module
*/
package server

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hoon-kr/log_manager/config"
	"github.com/hoon-kr/log_manager/internal/logger"
	"github.com/hoon-kr/log_manager/pkg/utils/file"
	"github.com/hoon-kr/log_manager/pkg/utils/process"
	"github.com/spf13/cobra"
)

// StartServer runs the Log Management daemon.
//
// Parameters:
//   - cmd: command parameter info
//
// Returns:
//   - int: normal shutdown(0), abnormal shutdown(>=1)
//   - error: normal shutdown(nil), abnormal shutdown(error)
func StartServer(cmd *cobra.Command) (int, error) {
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "[WARNING] invalid parameter: [*cobra.Command] is nil\n")
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Change working path to the current process path
	err := file.ChangeWorkPathToModulePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Verify that there is a process in operation
	var pid int
	if isRunning(&pid) {
		fmt.Fprintf(os.Stdout, "[INFO] there is already a process in operation (pid:%d)\n", pid)
		return config.ExitCodeSuccess, nil
	}

	// Daemonize process
	err = process.DaemonizeProcess()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Save current process pid
	config.RunConf.Pid = os.Getpid()

	// Write PID to file
	err = file.WriteDataToTextFile(config.PidFilePath, config.RunConf.Pid, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Check debug mode
	// In debug mode, stdout, stderr is output to the console
	if cmd.Use == "debug" {
		config.RunConf.DebugMode = true
	} else {
		os.Stdout = nil
		os.Stderr = nil
	}

	// Setup signal
	sigChan := setupSignal()

	// Module initialization
	initialization()
	// Finalization at the end of the module
	defer finalization()

	logger.Log.LogInfo("Start %s (pid:%d, mode:%s)", config.ModuleName, config.RunConf.Pid,
		func() string {
			if config.RunConf.DebugMode {
				return "debug"
			}
			return "normal"
		}())

	// Wait for the signal to terminate (SIGINT, SIGTERM)
	sig := <-sigChan
	logger.Log.LogInfo("Received %s signal (%d)", sig.String(), sig)

	return config.ExitCodeSuccess, nil
}

// StopServer stop the Log Management daemon.
//
// Parameters:
//   - cmd: command parameter info
//
// Returns:
//   - int: normal shutdown(0), abnormal shutdown(>=1)
//   - error: normal shutdown(nil), abnormal shutdown(error)
func StopServer(cmd *cobra.Command) (int, error) {
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "[WARNING] invalid parameter: [*cobra.Command] is nil\n")
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Change working path to the current process path
	err := file.ChangeWorkPathToModulePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	// Check process running
	var pid int
	if !isRunning(&pid) {
		return config.ExitCodeSuccess, nil
	}

	// Send stop(SIGTERM) signal
	if err := process.SendSignal(pid, syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "[WARNING] %s\n", err)
		return config.ExitCodeFailure, fmt.Errorf("%s(%d)", config.ExitFailure, config.ExitCodeFailure)
	}

	return config.ExitCodeSuccess, nil
}

// isRunning check log_manager process running.
//
// Returns:
//   - bool: running(true), stop(false)
func isRunning(pid *int) bool {
	if pid == nil {
		return false
	}

	// Open pid file
	file, err := os.Open(config.PidFilePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read pid
	pidStr, err := io.ReadAll(file)
	if err != nil {
		return false
	}

	// String pid to int pid
	*pid, err = strconv.Atoi(string(pidStr))
	if err != nil {
		return false
	}

	// Check process running
	return process.IsProcessRun(*pid)
}

// setupSignal set signal channel
//
// Returns:
//   - chan os.Signal: signal channel
func setupSignal() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	// Set received signal (SIGINT, SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Set signal to ignore
	signal.Ignore(syscall.SIGABRT, syscall.SIGALRM, syscall.SIGFPE, syscall.SIGHUP,
		syscall.SIGILL, syscall.SIGPROF, syscall.SIGQUIT, syscall.SIGTSTP,
		syscall.SIGVTALRM)

	return sigChan
}

// initialization initialize the resources required for the module operation.
func initialization() {
	// Load configuration
	config.LoadConfig(config.ConfFilePath)
	// Initialize logger
	logger.Log.InitializeLogger()
}

// finalization clean up all resources in use at the end of the module.
func finalization() {
	// Clean up log resources
	logger.Log.FinalizeLogger()
}
