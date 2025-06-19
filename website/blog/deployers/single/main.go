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

// Package main implements a simple singleprocess deployer. See
// https://mx.dev/blog/deployers.html for corresponding blog post.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/sh3lk/mx/runtime/colors"
	"github.com/sh3lk/mx/runtime/envelope"
	"github.com/sh3lk/mx/runtime/logging"
	"github.com/sh3lk/mx/runtime/protos"
)

// deployer is a simple single process deployer that runs every component in a
// single OS process.
type deployer struct {
	envelope   *envelope.Envelope // envelope to the mxn
	components []string           // activated components
}

// Check that deployer implements the envelope.EnvelopeHandler interface.
var _ envelope.EnvelopeHandler = &deployer{}

// Usage: ./single <service mx binary>
func main() {
	flag.Parse()
	if err := deploy(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}

// deploy deploys the provided application and blocks until it exits.
func deploy(binary string) error {
	// Spawn the mxn.
	info := &protos.MXNArgs{
		App:             "app",               // the application name
		DeploymentId:    uuid.New().String(), // the deployment id
		Id:              uuid.New().String(), // the mxn id
		RunMain:         true,                // should the mxn run main?
		InternalAddress: "localhost:0",       // internal address of the mxn
	}
	config := &protos.AppConfig{
		Name:   "app",  // the application name
		Binary: binary, // the application binary
	}
	envelope, err := envelope.NewEnvelope(context.Background(), info, config, envelope.Options{})
	if err != nil {
		return err
	}

	// Handle messages from the mxn.
	return envelope.Serve(&deployer{envelope: envelope})
}

// Responsibility 1: Components.
func (d *deployer) ActivateComponent(_ context.Context, req *protos.ActivateComponentRequest) (*protos.ActivateComponentReply, error) {
	d.components = append(d.components, req.Component)

	// Tell the mxn to run the component.
	d.envelope.UpdateComponents(d.components)

	// Tell the mxn to route requests to the component locally.
	d.envelope.UpdateRoutingInfo(&protos.RoutingInfo{
		Component: req.Component,
		Local:     true,
	})

	return &protos.ActivateComponentReply{}, nil
}

// Responsibility 2: Listeners.
func (d *deployer) GetListenerAddress(_ context.Context, req *protos.GetListenerAddressRequest) (*protos.GetListenerAddressReply, error) {
	return &protos.GetListenerAddressReply{Address: "localhost:0"}, nil
}

func (d *deployer) ExportListener(_ context.Context, req *protos.ExportListenerRequest) (*protos.ExportListenerReply, error) {
	// This simplified deployer does not proxy network traffic. Listeners
	// should be contacted directly.
	fmt.Printf("MXN listening on %s\n", req.Address)
	return &protos.ExportListenerReply{}, nil
}

// Responsibility 3: Telemetry.
func (d *deployer) LogBatch(_ context.Context, batch *protos.LogEntryBatch) error {
	pp := logging.NewPrettyPrinter(colors.Enabled())
	for _, entry := range batch.Entries {
		fmt.Println(pp.Format(entry))
	}
	return nil
}

func (d *deployer) HandleTraceSpans(context.Context, *protos.TraceSpans) error {
	// This simplified deployer drops traces on the floor.
	return nil
}

// Responsibility 4: Security.
func (*deployer) GetSelfCertificate(context.Context, *protos.GetSelfCertificateRequest) (*protos.GetSelfCertificateReply, error) {
	// This deployer doesn't enable mTLS.
	panic("unused")
}

func (*deployer) VerifyClientCertificate(context.Context, *protos.VerifyClientCertificateRequest) (*protos.VerifyClientCertificateReply, error) {
	// This deployer doesn't enable mTLS.
	panic("unused")
}

func (*deployer) VerifyServerCertificate(context.Context, *protos.VerifyServerCertificateRequest) (*protos.VerifyServerCertificateReply, error) {
	// This deployer doesn't enable network-level security.
	panic("unused")
}
