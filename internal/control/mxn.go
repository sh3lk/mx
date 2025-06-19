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

package control

import (
	"context"

	"github.com/sh3lk/mx/runtime/protos"
)

// MXNPath is the path used for the mxn control component.
// It points to an internal type in a different package.
const MXNPath = "github.com/sh3lk/mx/mxnControl"

// MXNControl is the interface for the mx.mxnControl component. It is
// present in its own package so other packages do not need to copy the interface
// definition.
//
// Arguments and results are protobufs to allow deployers to evolve independently of
// application binaries.
type MXNControl interface {
	// InitMXN initializes the mxn.
	InitMXN(context.Context, *protos.InitMXNRequest) (*protos.InitMXNReply, error)

	// UpdateComponents updates the mxn with the latest set of components it
	// should be running.
	UpdateComponents(context.Context, *protos.UpdateComponentsRequest) (*protos.UpdateComponentsReply, error)

	// UpdateRoutingInfo updates the mxn with a component's most recent routing info.
	UpdateRoutingInfo(context.Context, *protos.UpdateRoutingInfoRequest) (*protos.UpdateRoutingInfoReply, error)

	// GetHealth fetches mxn health information.
	GetHealth(context.Context, *protos.GetHealthRequest) (*protos.GetHealthReply, error)

	// GetLoad fetches mxn load information.
	GetLoad(context.Context, *protos.GetLoadRequest) (*protos.GetLoadReply, error)

	// GetMetrics fetches metrics from the mxn.
	GetMetrics(context.Context, *protos.GetMetricsRequest) (*protos.GetMetricsReply, error)

	// GetProfile gets a profile from the mxn.
	GetProfile(context.Context, *protos.GetProfileRequest) (*protos.GetProfileReply, error)
}
