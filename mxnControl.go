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

package mx

import (
	"context"
	"fmt"

	"github.com/sh3lk/mx/internal/control"
	"github.com/sh3lk/mx/runtime/protos"
)

// mxnControl is a component hosted in every mxn. Deployers make calls to this component
// to fetch information about the mxn, and to make it do various things.
type mxnControl control.MXNControl

// noopMXNControl is a no-op implementation of mxnControl. It exists solely to cause
// mxnControl to be registered as a component. The actual implementation is provided
// by internal/mx/remotemxn.go
type noopMXNControl struct {
	Implements[mxnControl]
}

var _ mxnControl = &noopMXNControl{}

// InitMXN implements mxnControl interface.
func (*noopMXNControl) InitMXN(context.Context, *protos.InitMXNRequest) (*protos.InitMXNReply, error) {
	return nil, fmt.Errorf("mxnControl.InitMXN not implemented")
}

// UpdateComponents implements mxnControl interface.
func (*noopMXNControl) UpdateComponents(context.Context, *protos.UpdateComponentsRequest) (*protos.UpdateComponentsReply, error) {
	return nil, fmt.Errorf("mxnControl.UpdateComponents not implemented")
}

// UpdateRoutingInfo implements mxnControl interface.
func (*noopMXNControl) UpdateRoutingInfo(context.Context, *protos.UpdateRoutingInfoRequest) (*protos.UpdateRoutingInfoReply, error) {
	return nil, fmt.Errorf("mxnControl.UpdateRoutingInfo not implemented")
}

// GetHealth implements mxnControl interface.
func (*noopMXNControl) GetHealth(context.Context, *protos.GetHealthRequest) (*protos.GetHealthReply, error) {
	return nil, fmt.Errorf("mxnControl.GetHealth not implemented")
}

// GetLoad implements mxnControl interface.
func (*noopMXNControl) GetLoad(context.Context, *protos.GetLoadRequest) (*protos.GetLoadReply, error) {
	return nil, fmt.Errorf("mxnControl.GetLoad not implemented")
}

// GetMetrics implements mxnControl interface.
func (*noopMXNControl) GetMetrics(context.Context, *protos.GetMetricsRequest) (*protos.GetMetricsReply, error) {
	return nil, fmt.Errorf("mxnControl.GetMetrics not implemented")
}

// GetProfile implements mxnControl interface.
func (*noopMXNControl) GetProfile(context.Context, *protos.GetProfileRequest) (*protos.GetProfileReply, error) {
	return nil, fmt.Errorf("mxnControl.GetProfile not implemented")
}
