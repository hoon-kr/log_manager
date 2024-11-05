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
Package cmd is a module execution command processing package.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/hoon-kr/log_manager/config"
	"github.com/hoon-kr/log_manager/procedure"
	"github.com/spf13/cobra"
	"go.uber.org/automaxprocs/maxprocs"
)

// logManagerCmd represents the base command when called without any subcommands
var logManagerCmd = &cobra.Command{
	Use:   "log_manager",
	Short: "log_manager is a Log management module via API.",
	Long: `The log_manager module provides log-related APIs such as log inquiry,
deletion, and addition.`,
	Version: config.Version,
}

// startCmd run server
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run log_manager (normal mode)",
	// Run the log management daemon
	RunE: wrapCommandFuncForCobra(procedure.StartServer),
}

// debugCmd run server (debug)
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run log_manager (debug mode)",
	// Run the log management daemon (debug)
	RunE: wrapCommandFuncForCobra(procedure.StartServer),
}

// stopCmd stop server
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop log_manager",
	// Stop the log management daemon
	RunE: wrapCommandFuncForCobra(procedure.StopServer),
}

// init Initialize when importing cmd packages.
func init() {
	logManagerCmd.AddCommand(startCmd)
	logManagerCmd.AddCommand(debugCmd)
	logManagerCmd.AddCommand(stopCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the logManagerCmd.
func Execute() {
	// Set the GOMAXPROCS value to an optimized value
	undo, err := maxprocs.Set()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[WARNING] failed to set GOMAXPROCS: %s\n", err)
	}
	defer undo()

	// Process command or flags
	err = logManagerCmd.Execute()
	if err != nil {
		var exitErr *config.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode)
		}
		os.Exit(1)
	}
}

// wrapCommandFuncForCobra wraps function for use
// in a cobra command's RunE field.
//
// Parameters:
//   - f: command function
//
// Returns:
//   - error: normal exit(nil), abnormal exit(error)
func wrapCommandFuncForCobra(f func(cmd *cobra.Command) (int, error)) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		status, err := f(cmd)
		if status > 1 {
			cmd.SilenceErrors = true
			return &config.ExitError{ExitCode: status, Err: err}
		}
		return err
	}
}
