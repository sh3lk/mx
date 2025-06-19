// Copyright 2024 Google LLC
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

// DeployerPath is the path used for the deployer control component.
// It points to an internal type in a different package.
const DeployerPath = "github.com/sh3lk/mx/deployerControl"

// DeployerControl is the interface for the mx.deployerControl component. It is
// present in its own package so other packages do not need to copy the interface
// definition.
//
// Arguments and results are protobufs to allow deployers to evolve independently
// of application binaries.
type DeployerControl interface {
	// LogBatch logs the supplied batch of log entries.
	LogBatch(context.Context, *protos.LogEntryBatch) error

	// HandleTraceSpans saves the supplied trace spans.
	HandleTraceSpans(context.Context, *protos.TraceSpans) error

	// ActivateComponent ensures that the provided component is running
	// somewhere. A call to ActivateComponent also implicitly signals that a
	// mxn is interested in receiving routing info for the component.
	ActivateComponent(context.Context, *protos.ActivateComponentRequest) (*protos.ActivateComponentReply, error)

	// GetListenerAddress returns the address the mxn should listen on for
	// a particular listener.
	GetListenerAddress(context.Context, *protos.GetListenerAddressRequest) (*protos.GetListenerAddressReply, error)

	// ExportListener exports the provided listener. Exporting a listener
	// typically, but not always, involves running a proxy that forwards
	// traffic to the provided address.
	ExportListener(context.Context, *protos.ExportListenerRequest) (*protos.ExportListenerReply, error)

	// GetSelfCertificate returns the certificate and the private key the
	// mxn should use for network connection establishment. The mxn
	// will issue this request each time it establishes a connection with
	// another mxn.
	// NOTE: This method is only called if mTLS was enabled for the mxn,
	// by passing it a MXNArgs with mtls=true.
	GetSelfCertificate(context.Context, *protos.GetSelfCertificateRequest) (*protos.GetSelfCertificateReply, error)

	// VerifyClientCertificate verifies the certificate chain presented by
	// a network client attempting to connect to the mxn. It returns an
	// error if the network connection should not be established with the
	// client. Otherwise, it returns the list of mxn components that the
	// client is authorized to invoke methods on.
	//
	// NOTE: This method is only called if mTLS was enabled for the mxn,
	// by passing it a MXNArgs with mtls=true.
	VerifyClientCertificate(context.Context, *protos.VerifyClientCertificateRequest) (*protos.VerifyClientCertificateReply, error)

	// VerifyServerCertificate verifies the certificate chain presented by
	// the server the mxn is attempting to connect to. It returns an
	// error iff the server identity doesn't match the identity of the specified
	// component.
	//
	// NOTE: This method is only called if mTLS was enabled for the mxn,
	// by passing it a MXNArgs with mtls=true.
	VerifyServerCertificate(context.Context, *protos.VerifyServerCertificateRequest) (*protos.VerifyServerCertificateReply, error)
}
