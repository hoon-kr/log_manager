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
Package config implements setting-related functions necessary
for module operation.
*/
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	BuildTime  = "unknown"     // Set at the time of build
	Version    = "1.0.0"       // Module version info
	ModuleName = "log_manager" // Module name
)

const (
	ConfFilePath       = "conf/log_manager.properties"
	PidFilePath        = "var/log_manager.pid"
	ConsoleLogFilePath = "log/log_manager.log"
	JsonLogFilePath    = "log/log_manager_json.log"
)

// Exit Code
const (
	ExitCodeSuccess = iota
	ExitCodeFailure
	ExitCodeFatal
)

// Exit message
const (
	ExitSuccess = "exit success"
	ExitFailure = "exit failure"
	ExitFatal   = "exit fatal"
)

// ExitError carries the exit code
type ExitError struct {
	ExitCode int
	Err      error
}

// Error It serves to return the contents of ExitError as a string.
//
// Returns:
//   - string: error string
func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exiting with status %d", e.ExitCode)
	}
	return e.Err.Error()
}

// Config is a global configuration structure
type Config struct {
	// Maximum size per log file (DEF:100MB, MIN:1MB, MAX:1000MB)
	MaxLogFileSize int
	// Maximum number of log file backups (DEF:10, MIN:1, MAX:100)
	MaxLogFileBackup int
	// Number of days to keep backup log files (DEF:90, MIN:1, MAX:365)
	MaxLogFileAge int
	// Whether backup log files are compressed (DEF:true, ENABLE:true, DISABLE:false)
	CompBakLogFile bool
}

// RunConfig is a global running configuration structure
type RunConfig struct {
	DebugMode bool
	Pid       int
}

var Conf Config
var RunConf RunConfig

// init Initialize when importing config packages.
func init() {
	Conf.MaxLogFileSize = 100
	Conf.MaxLogFileBackup = 10
	Conf.MaxLogFileAge = 90
	Conf.CompBakLogFile = true
}

// LoadConfig loads configuration.
//
// Parameters:
//   - filePath: config file path
//
// Returns:
//   - error: success(nil), failure(error)
func LoadConfig(filePath string) error {
	// Parse configuration file
	config, err := parseConfig(filePath)
	if err != nil {
		return err
	}

	if valueStr, exists := config["MaxLogFileSize"]; exists {
		value, err := strconv.Atoi(valueStr)
		if err != nil && value >= 1 && value <= 1000 {
			Conf.MaxLogFileSize = value
		}
	}

	if valueStr, exists := config["MaxLogFileBackup"]; exists {
		value, err := strconv.Atoi(valueStr)
		if err != nil && value >= 1 && value <= 100 {
			Conf.MaxLogFileBackup = value
		}
	}

	if valueStr, exists := config["MaxLogFileAge"]; exists {
		value, err := strconv.Atoi(valueStr)
		if err != nil && value >= 1 && value <= 365 {
			Conf.MaxLogFileAge = value
		}
	}

	if valueStr, exists := config["CompressBackupLogFile"]; exists {
		if strings.ToLower(valueStr) == "no" {
			Conf.CompBakLogFile = false
		}
	}

	return nil
}

// parseConfig parse the configuration file and return it to the map.
//
// Parameters:
//   - filePath: config file path
//
// Returns:
//   - map[string]string: config map
//   - error: success(nil), failure(error)
func parseConfig(filePath string) (map[string]string, error) {
	config := make(map[string]string)

	// Open config file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}
	defer file.Close()

	// Read files by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty line or annotate
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Separate line to key, value
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		// append key, value to config map
		config[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}

	return config, nil
}
