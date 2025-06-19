// Copyright 2022 Google LLC
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
	"fmt"

	"github.com/sh3lk/mx/runtime/protos"
)

// Main is the name of the main component.
const Main = "github.com/sh3lk/mx/Main"

// CheckMXNArgs checks that MXNArgs is well-formed.
func CheckMXNArgs(w *protos.MXNArgs) error {
	if w == nil {
		return fmt.Errorf("MXNArgs: nil")
	}
	if w.App == "" {
		return fmt.Errorf("MXNArgs: missing app name")
	}
	if w.DeploymentId == "" {
		return fmt.Errorf("MXNArgs: missing deployment id")
	}
	if w.Id == "" {
		return fmt.Errorf("MXNArgs: missing mxn id")
	}
	if w.ControlSocket == "" {
		return fmt.Errorf("MXNArgs: missing control socket")
	}
	return nil
}
