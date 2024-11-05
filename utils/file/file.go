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
Package file provides file processing-related functions.
*/
package file

import (
	"fmt"
	"os"
	"path/filepath"
)

// ChangeWorkPathToModulePath change the working path to the module path.
//
// Returns:
//   - error: success(nil), failure(error)
func ChangeWorkPathToModulePath() error {
	// Get absolute path of the current process
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to absolute path: %s", err)
	}

	// Get path's directory
	dirPath := filepath.Dir(exePath)

	// Change working directory
	err = os.Chdir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to change dir: %s", err)
	}

	return nil
}

// WriteDataToTextFile is a generic text file write function.
//
// Parameters:
//   - filePath: file path to be written
//   - data: generic type data
//   - isMakeDir: option to create file path directory if it does not exist
//
// Returns:
//   - error: success(nil), failure(error)
func WriteDataToTextFile[T any](filePath string, data T, isMakeDir bool) error {
	if isMakeDir {
		// If directory does not exist, create directory
		dir := filepath.Dir(filePath)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to make directory: %s", err)
		}
	}

	// Creates or truncates the named file (write only)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %s", err)
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%v", data)
	if err != nil {
		return fmt.Errorf("failed to write file: %s", err)
	}

	return nil
}
