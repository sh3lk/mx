// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"os"
	"path/filepath"
)

// LogsDir returns the default directory for MX logs,
// $DIR/tmp/mx/logs where $DIR is the default directory used for
// temporary files (see [os.TempDir] for details). We recommend that deployers
// store their logs in a directory within this default directory. For example,
// on Unix systems, the "mx multi" deployer stores its data in
// /tmp/mx/logs/multi.
func LogsDir() string {
	return filepath.Join(os.TempDir(), "mx", "logs")
}

// DataDir returns the default directory for MX deployer data. The
// returned directory is $XDG_DATA_HOME/mx, or
// ~/.local/share/mx if XDG_DATA_HOME is not set.
//
// We recommend that deployers store their data in a directory within this
// default directory. For example, the "mx multi" deployer stores its data
// in "DataDir()/multi".
func DataDir() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		// Default to ~/.local/share
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	regDir := filepath.Join(dataDir, "mx")
	if err := os.MkdirAll(regDir, 0700); err != nil {
		return "", err
	}

	return regDir, nil
}

// NewTempDir returns a new directory, e.g., to hold Unix domain sockets for
// internal communication. The new directory is not accessible by other users.
// Caller is responsible for cleaning up the directory when not needed.
func NewTempDir() (string, error) {
	// Make temporary directory.
	tmpDir, err := os.MkdirTemp("", "mx")
	if err != nil {
		return "", err
	}
	if err := os.Chmod(tmpDir, 0o700); err != nil {
		os.Remove(tmpDir)
		return "", err
	}
	return tmpDir, nil
}
