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

// Package runtime contains code suitable for deployer implementers but not
// MX application developers.
package runtime

import (
	"context"
	"fmt"
	"os"

	"github.com/sh3lk/mx/internal/proto"
	"github.com/sh3lk/mx/runtime/protos"
)

const (
	// MXNArgsKey is the environment variable that holds the base64 encoded
	// protos.MXNArgs message for a mxn started by an envelope. For internal
	// use by MX infrastructure.
	MXNArgsKey = "MXN_ARGS"
)

// Bootstrap holds configuration information used to start a process execution.
type Bootstrap struct {
	Args *protos.MXNArgs
}

// GetBootstrap returns information needed to configure process
// execution. For normal execution, this comes from the environment. For
// mxtest, it comes from a context value.
func GetBootstrap(ctx context.Context) (Bootstrap, error) {
	argsEnv := os.Getenv(MXNArgsKey)
	if argsEnv == "" {
		return Bootstrap{}, nil
	}
	args := &protos.MXNArgs{}
	if err := proto.FromEnv(argsEnv, args); err != nil {
		return Bootstrap{}, fmt.Errorf("decoding mxn args: %w", err)
	}
	return Bootstrap{
		Args: args,
	}, nil
}

// Exists returns true if bootstrap information has been supplied. This
// is true except in the case of singleprocess.
func (b Bootstrap) Exists() bool {
	return b.Args != nil
}
