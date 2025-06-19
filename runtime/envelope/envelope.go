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

// Package envelope implements a sidecar-like process that connects a mxn
// to its environment.
package envelope

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/sh3lk/mx/internal/control"
	"github.com/sh3lk/mx/internal/net/call"
	"github.com/sh3lk/mx/runtime"
	"github.com/sh3lk/mx/runtime/codegen"
	"github.com/sh3lk/mx/runtime/deployers"
	"github.com/sh3lk/mx/runtime/metrics"
	"github.com/sh3lk/mx/runtime/protomsg"
	"github.com/sh3lk/mx/runtime/protos"
	"github.com/sh3lk/mx/runtime/version"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	// We rely on the mx.controller component registrattion entry.
	_ "github.com/sh3lk/mx"
)

// EnvelopeHandler handles messages from the mxn. Values passed to the
// handlers are only valid for the duration of the handler's execution.
type EnvelopeHandler interface {
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

	// LogBatches handles a batch of log entries.
	LogBatch(context.Context, *protos.LogEntryBatch) error

	// HandleTraceSpans handles a set of trace spans.
	HandleTraceSpans(context.Context, *protos.TraceSpans) error
}

// Ensure that EnvelopeHandler implements all the DeployerControl methods.
var _ control.DeployerControl = EnvelopeHandler(nil)

// Envelope starts and manages a mxn in a subprocess.
//
// For more information, refer to runtime/protos/runtime.proto and
// https://mx.dev/blog/deployers.html.
type Envelope struct {
	// Fields below are constant after construction.
	ctx         context.Context
	ctxCancel   context.CancelFunc
	logger      *slog.Logger
	tmpDir      string
	tmpDirOwned bool // Did Envelope create tmpDir?
	myUds       string
	mxn         *protos.MXNArgs
	mxnAddr     string
	config      *protos.AppConfig
	child       Child              // mxn process handle
	controller  control.MXNControl // Stub that talks to the mxn controller

	// State needed to process metric updates.
	metricsMu sync.Mutex
	metrics   metrics.Importer
}

// Options contains optional arguments for the envelope.
type Options struct {
	// Override for temporary directory.
	TmpDir string

	// Logger is used for logging internal messages. If nil, a default logger is used.
	Logger *slog.Logger

	// Tracer is used for tracing internal calls. If nil, internal calls are not traced.
	Tracer trace.Tracer

	// Child is used to run the mxn. If nil, a sub-process is created.
	Child Child
}

// NewEnvelope creates a new envelope, starting a mxn subprocess (via child.Start) and
// establishing a bidirectional connection with it. The mxn process can be
// stopped at any time by canceling the passed-in context.
//
// You can issue RPCs *to* the mxn using the returned Envelope. To start
// receiving messages *from* the mxn, call [Serve].
func NewEnvelope(ctx context.Context, wlet *protos.MXNArgs, config *protos.AppConfig, options Options) (*Envelope, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() { cancel() }() // cancel may be changed below if we want to delay it

	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	// Make a temporary directory for unix domain sockets.
	var removeDir bool
	tmpDir := options.TmpDir
	tmpDirOwned := false
	if options.TmpDir == "" {
		var err error
		tmpDir, err = runtime.NewTempDir()
		if err != nil {
			return nil, err
		}
		tmpDirOwned = true
		runtime.OnExitSignal(func() { os.RemoveAll(tmpDir) }) // Cleanup when process exits

		// Arrange to delete tmpDir if this function returns an error.
		removeDir = true // Cleared on a successful return
		defer func() {
			if removeDir {
				os.RemoveAll(tmpDir)
			}
		}()
	}

	myUds := deployers.NewUnixSocketPath(tmpDir)

	wlet = protomsg.Clone(wlet)
	wlet.ControlSocket = deployers.NewUnixSocketPath(tmpDir)
	wlet.Redirects = []*protos.MXNArgs_Redirect{
		// Point mxn at my control.DeployerControl component
		{
			Component: control.DeployerPath,
			Target:    control.DeployerPath,
			Address:   "unix://" + myUds,
		},
	}
	controller, err := getMXNControlStub(ctx, wlet.ControlSocket, options)
	if err != nil {
		return nil, err
	}
	e := &Envelope{
		ctx:         ctx,
		ctxCancel:   cancel,
		logger:      options.Logger,
		tmpDir:      tmpDir,
		tmpDirOwned: tmpDirOwned,
		myUds:       myUds,
		mxn:         wlet,
		config:      config,
		controller:  controller,
	}

	child := options.Child
	if child == nil {
		child = &ProcessChild{}
	}
	if err := child.Start(ctx, e.config, e.mxn); err != nil {
		return nil, fmt.Errorf("NewEnvelope: %w", err)
	}

	reply, err := controller.InitMXN(e.ctx, &protos.InitMXNRequest{
		Sections: config.Sections,
	})
	if err != nil {
		return nil, err
	}
	if err := verifyMXNInfo(reply); err != nil {
		return nil, err
	}
	e.mxnAddr = reply.DialAddr

	e.child = child

	removeDir = false  // Serve() is now responsible for deletion
	cancel = func() {} // Delay real context cancellation
	return e, nil
}

// MXNControl returns the controller component for the mxn managed by this envelope.
func (e *Envelope) MXNControl() control.MXNControl { return e.controller }

// Serve accepts incoming messages from the mxn. RPC requests are handled
// serially in the order they are received. Serve blocks until the connection
// terminates, returning the error that caused it to terminate. You can cancel
// the connection by cancelling the context passed to [NewEnvelope]. This
// method never returns a non-nil error.
func (e *Envelope) Serve(h EnvelopeHandler) error {
	// Cleanup when we are done with the envelope.
	if e.tmpDirOwned {
		defer os.RemoveAll(e.tmpDir)
	}

	uds, err := net.Listen("unix", e.myUds)
	if err != nil {
		return err
	}

	var running errgroup.Group

	var stopErr error
	var once sync.Once
	stop := func(err error) {
		once.Do(func() {
			stopErr = err
		})
		e.ctxCancel()
	}

	// Capture stdout and stderr from the mxn.
	if stdout := e.child.Stdout(); stdout != nil {
		running.Go(func() error {
			err := e.logLines("stdout", stdout, h)
			stop(err)
			return err
		})
	}
	if stderr := e.child.Stderr(); stderr != nil {
		running.Go(func() error {
			err := e.logLines("stderr", stderr, h)
			stop(err)
			return err
		})
	}

	// Start the goroutine watching the context for cancelation.
	running.Go(func() error {
		<-e.ctx.Done()
		err := e.ctx.Err()
		stop(err)
		return err
	})

	// Start the goroutine to handle deployer control calls.
	running.Go(func() error {
		err := deployers.ServeComponents(e.ctx, uds, e.logger, map[string]any{
			control.DeployerPath: h,
		})
		stop(err)
		return err
	})

	running.Wait()

	// Wait for the mxn command to finish. This needs to be done after
	// we're done reading from stdout/stderr pipes, per comments on
	// exec.Cmd.StdoutPipe and exec.Cmd.StderrPipe.
	stop(e.child.Wait())

	return stopErr
}

// Pid returns the process id of the mxn, if it is running in a separate process.
func (e *Envelope) Pid() (int, bool) {
	return e.child.Pid()
}

// MXNAddress returns the address that other components should dial to communicate with the
// mxn.
func (e *Envelope) MXNAddress() string {
	return e.mxnAddr
}

// GetHealth returns the health status of the mxn.
func (e *Envelope) GetHealth() *protos.GetHealthReply {
	reply, err := e.controller.GetHealth(context.TODO(), &protos.GetHealthRequest{})
	if err != nil {
		return &protos.GetHealthReply{Status: protos.HealthStatus_UNKNOWN}
	}
	return reply
}

// GetProfile gets a profile from the mxn.
func (e *Envelope) GetProfile(req *protos.GetProfileRequest) ([]byte, error) {
	reply, err := e.controller.GetProfile(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return reply.Data, nil
}

// GetMetrics returns a mxn's metrics.
func (e *Envelope) GetMetrics() ([]*metrics.MetricSnapshot, error) {
	req := &protos.GetMetricsRequest{}
	reply, err := e.controller.GetMetrics(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	e.metricsMu.Lock()
	defer e.metricsMu.Unlock()
	return e.metrics.Import(reply.Update)
}

// GetLoad gets a load report from the mxn.
func (e *Envelope) GetLoad() (*protos.LoadReport, error) {
	req := &protos.GetLoadRequest{}
	reply, err := e.controller.GetLoad(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return reply.Load, nil
}

// UpdateComponents updates the mxn with the latest set of components it
// should be running.
func (e *Envelope) UpdateComponents(components []string) error {
	req := &protos.UpdateComponentsRequest{
		Components: components,
	}
	_, err := e.controller.UpdateComponents(context.TODO(), req)
	return err
}

// UpdateRoutingInfo updates the mxn with a component's most recent
// routing info.
func (e *Envelope) UpdateRoutingInfo(routing *protos.RoutingInfo) error {
	req := &protos.UpdateRoutingInfoRequest{
		RoutingInfo: routing,
	}
	_, err := e.controller.UpdateRoutingInfo(context.TODO(), req)
	return err
}

func (e *Envelope) logLines(component string, src io.Reader, h EnvelopeHandler) error {
	// Fill partial log entry.
	entry := &protos.LogEntry{
		App:       e.mxn.App,
		Version:   e.mxn.DeploymentId,
		Component: component,
		Node:      e.mxn.Id,
		Level:     component, // Either "stdout" or "stderr"
		File:      "",
		Line:      -1,
	}
	batch := &protos.LogEntryBatch{}
	batch.Entries = append(batch.Entries, entry)

	rdr := bufio.NewReader(src)
	for {
		line, err := rdr.ReadBytes('\n')
		// Note: both line and err may be present.
		if len(line) > 0 {
			entry.Msg = string(dropNewline(line))
			entry.TimeMicros = 0 // In case previous LogBatch mutated it
			if err := h.LogBatch(e.ctx, batch); err != nil {
				return err
			}
		}
		if err != nil {
			return fmt.Errorf("capture %s: %w", component, err)
		}
	}
}

func dropNewline(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	return line
}

// getMXNControlStub returns a control.MXNControl that forwards calls to the controller
// component in the mxn at the specified socket.
func getMXNControlStub(ctx context.Context, socket string, options Options) (control.MXNControl, error) {
	controllerReg, ok := codegen.Find(control.MXNPath)
	if !ok {
		return nil, fmt.Errorf("controller component (%s) not found", control.MXNPath)
	}
	controlEndpoint := call.Unix(socket)
	resolver := call.NewConstantResolver(controlEndpoint)
	opts := call.ClientOptions{Logger: options.Logger}
	conn, err := call.Connect(ctx, resolver, opts)
	if err != nil {
		return nil, err
	}
	// We skip waitUntilReady() and rely on automatic retries of methods
	stub := call.NewStub(control.MXNPath, controllerReg, conn, options.Tracer, 0)
	obj := controllerReg.ClientStubFn(stub, "envelope")
	return obj.(control.MXNControl), nil
}

// verifyMXNInfo verifies the information sent by the mxn.
func verifyMXNInfo(wlet *protos.InitMXNReply) error {
	if wlet == nil {
		return fmt.Errorf(
			"the first message from the mxn must contain mxn info")
	}
	if wlet.DialAddr == "" {
		return fmt.Errorf("empty dial address for the mxn")
	}
	if err := checkVersion(wlet.Version); err != nil {
		return err
	}
	return nil
}

// checkVersion checks that the deployer API version the deployer was built
// with is compatible with the deployer API version the app was built with,
// erroring out if they are not compatible.
func checkVersion(v *protos.SemVer) error {
	if v == nil {
		return fmt.Errorf("version mismatch: nil app version")
	}
	got := version.SemVer{Major: int(v.Major), Minor: int(v.Minor), Patch: int(v.Patch)}
	if got != version.DeployerVersion {
		return fmt.Errorf("version mismatch: deployer's deployer API version %s is incompatible with app' deployer API version %s", version.DeployerVersion, got)
	}
	return nil
}
