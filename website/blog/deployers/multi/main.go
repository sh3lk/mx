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

// Package main implements a simple multiprocess deployer. See
// https://mx.dev/blog/deployers.html for corresponding blog post.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/sh3lk/mx/runtime"
	"github.com/sh3lk/mx/runtime/colors"
	"github.com/sh3lk/mx/runtime/envelope"
	"github.com/sh3lk/mx/runtime/logging"
	"github.com/sh3lk/mx/runtime/protos"
)

// deployer is a simple multiprocess deployer that doesn't implement
// co-location or replication. That is, every component is run in its own OS
// process, and there is only one replica of every component.
type deployer struct {
	mu       sync.Mutex          // guards handlers
	handlers map[string]*handler // handlers, by component
}

// A handler handles messages from a mxn. It implements the
// EnvelopeHandler interface.
type handler struct {
	deployer *deployer          // underlying deployer
	envelope *envelope.Envelope // envelope to the mxn
	address  string             // mxn's address
}

// Check that handler implements the envelope.EnvelopeHandler interface.
var _ envelope.EnvelopeHandler = &handler{}

// The unique id of the application deployment.
var deploymentId = uuid.New().String()

// Usage: ./multi <service mx binary>
func main() {
	flag.Parse()
	d := &deployer{handlers: map[string]*handler{}}
	if _, err := d.spawn(runtime.Main); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	select {} // block forever
}

// spawn spawns a mxn to host the provided component (if one hasn't
// already spawned) and returns a handler to the mxn.
func (d *deployer) spawn(component string) (*handler, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if a mxn has already been spawned.
	if h, ok := d.handlers[component]; ok {
		// The mxn has already been spawned.
		return h, nil
	}

	// Spawn a mxn in a subprocess to host the component.
	info := &protos.MXNArgs{
		App:             "app",                     // the application name
		DeploymentId:    deploymentId,              // the deployment id
		Id:              uuid.New().String(),       // the mxn id
		Mtls:            false,                     // don't enable mtls
		RunMain:         component == runtime.Main, // should the mxn run main?
		InternalAddress: "localhost:0",             // internal address of the mxn
	}
	config := &protos.AppConfig{
		Name:   "app",       // the application name
		Binary: flag.Arg(0), // the application binary
	}
	envelope, err := envelope.NewEnvelope(context.Background(), info, config, envelope.Options{})
	if err != nil {
		return nil, err
	}
	h := &handler{
		deployer: d,
		envelope: envelope,
		address:  envelope.MXNAddress(),
	}

	go func() {
		// Inform the mxn of the component it should host.
		envelope.UpdateComponents([]string{component})
	}()

	go func() {
		// Handle messages from the mxn.
		envelope.Serve(h)
	}()

	// Return the handler.
	d.handlers[component] = h
	return h, nil
}

// Responsibility 1: Components.
func (h *handler) ActivateComponent(_ context.Context, req *protos.ActivateComponentRequest) (*protos.ActivateComponentReply, error) {
	// Spawn a mxn to host the component, if one hasn't already been
	// spawned.
	spawned, err := h.deployer.spawn(req.Component)
	if err != nil {
		return nil, err
	}

	// Tell the mxn the address of the requested component.
	h.envelope.UpdateRoutingInfo(&protos.RoutingInfo{
		Component: req.Component,
		Replicas:  []string{spawned.address},
	})

	return &protos.ActivateComponentReply{}, nil
}

// Responsibility 2: Listeners.
func (h *handler) GetListenerAddress(_ context.Context, req *protos.GetListenerAddressRequest) (*protos.GetListenerAddressReply, error) {
	return &protos.GetListenerAddressReply{Address: "localhost:0"}, nil
}

func (h *handler) ExportListener(_ context.Context, req *protos.ExportListenerRequest) (*protos.ExportListenerReply, error) {
	// This simplified deployer does not proxy network traffic. Listeners
	// should be contacted directly.
	fmt.Printf("MXN listening on %s\n", req.Address)
	return &protos.ExportListenerReply{}, nil
}

// Responsibility 3: Telemetry.
func (h *handler) LogBatch(_ context.Context, batch *protos.LogEntryBatch) error {
	pp := logging.NewPrettyPrinter(colors.Enabled())
	for _, entry := range batch.Entries {
		fmt.Println(pp.Format(entry))
	}
	return nil
}

func (h *handler) HandleTraceSpans(context.Context, *protos.TraceSpans) error {
	// This simplified deployer drops traces on the floor.
	return nil
}

// Responsibility 4: Security.
func (*handler) GetSelfCertificate(context.Context, *protos.GetSelfCertificateRequest) (*protos.GetSelfCertificateReply, error) {
	// This deployer doesn't enable mTLS.
	panic("unused")
}

func (*handler) VerifyClientCertificate(context.Context, *protos.VerifyClientCertificateRequest) (*protos.VerifyClientCertificateReply, error) {
	// This deployer doesn't enable mTLS.
	panic("unused")
}

func (*handler) VerifyServerCertificate(context.Context, *protos.VerifyServerCertificateRequest) (*protos.VerifyServerCertificateReply, error) {
	// This deployer doesn't enable mTLS.
	panic("unused")
}
