<div hidden class="todo">
TODO: Link to code snippets to make sure they are compilable and runnable.
</div>

# What is MX?

MX is a programming framework for writing, deploying, and managing
distributed applications. You can run, test, and debug a MX application
locally on your machine, and then deploy the application to the cloud with a
single command.

```console
$ go run .                       # Run locally.
$ mx ssh deploy mx.toml  # Run on multiple machines.
$ mx gke deploy mx.toml  # Run on Google Cloud.
$ mx kube deploy mx.toml # Run on Kubernetes.
```

A MX application is composed of a number of **components**. A
component is represented as a regular Go [interface][go_interfaces], and
components interact with each other by calling the methods defined by these
interfaces. This makes writing MX applications easy. You don't have
to write any networking or serialization code; you just write Go. MX
also provides libraries for logging, metrics, tracing, routing, testing, and
more.

You can deploy a MX application as easily as running a single command. Under
the covers, MX will dissect your binary along component boundaries, allowing
different components to run on different machines. MX will replicate,
autoscale, and co-locate these distributed components for you. It will also
manage all the networking details on your behalf, ensuring that different
components can communicate with each other and that clients can communicate with
your application.

Refer to the [Installation](#installation) section to install MX on
your machine, or read the [Step by Step Tutorial](#step-by-step-tutorial)
section for a tutorial on how to write MX applications.

# Installation

Ensure you have [Go installed][go_install], version 1.21 or higher. Then, run
the following to install the `mx` command:

```console
$ go install github.com/sh3lk/mx/cmd/mx@latest
```

`go install` installs the `mx` command to `$GOBIN`, which defaults to
`$HOME/go/bin`. Make sure this directory is included in your `PATH`. You can
accomplish this, for example, by adding the following to your `.bashrc` and
running `source ~/.bashrc`:

```console
$ export PATH="$PATH:$HOME/go/bin"
```

If the installation was successful, you should be able to run `mx --help`:

```console
$ mx --help
USAGE

  mx generate                 // mx code generator
  mx version                  // show mx version
  mx single    <command> ...  // for single process deployments
  mx multi     <command> ...  // for multiprocess deployments
  mx ssh       <command> ...  // for multimachine deployments
  ...
```

**Note**: For cloud deployments you should also install the `mx gke` or
`mx kube` command (see the [GKE](#gke), [Kube](#kube) sections for details):

```console
$ go install github.com/sh3lk/mx-gke/cmd/mx-gke@latest
$ go install github.com/sh3lk/mx-kube/cmd/mx-kube@latest
```

**Note**: If you run into issues installing `mx`, `mx gke` or `mx kube`
commands on macOS, you may want to prefix the install command with
`export CGO_ENABLED=1; export CC=gcc`.
For example:
```console
$ export CGO_ENABLED=1; export CC=gcc; go install github.com/sh3lk/mx/cmd/mx@latest
```

# Step by Step Tutorial

In this section, we show you how to write MX applications. To
install MX and follow along, refer to the
[Installation](#installation) section. The full source code presented in this
tutorial can be found [here][hello_app].

## Components

MX's core abstraction is the **component**. A component is like an
[actor][actors], and a MX application is implemented as a set of
components. Concretely, a component is represented with a regular Go
[interface][go_interfaces], and components interact with each other by calling
the methods defined by these interfaces.

In this section, we'll define a simple `hello` component that just prints
a string and returns. First, run `go mod init hello` to create a go module.

```console
$ mkdir hello/
$ cd hello/
$ go mod init hello
```

Then, create a file called `main.go` with the following contents:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sh3lk/mx"
)

func main() {
    if err := mx.Run(context.Background(), serve); err != nil {
        log.Fatal(err)
    }
}

// app is the main component of the application. mx.Run creates
// it and passes it to serve.
type app struct{
    mx.Implements[mx.Main]
}

// serve is called by mx.Run and contains the body of the application.
func serve(context.Context, *app) error {
    fmt.Println("Hello")
    return nil
}
```

`mx.Run(...)` initializes and runs the MX application. In
particular, `mx.Run` finds the main component, creates it, and passes it to
a supplied function. In this example,`app` is the main component since it
contains a `mx.Implements[mx.Main]` field.

Before we build and run the app, we need to run MX's code generator,
called `mx generate`. `mx generate` writes a `mx_gen.go` file that
contains code needed by the MX runtime. We'll elaborate on what
exactly `mx generate` does and why we need to run it later. Finally, run the
app!

```console
$ go mod tidy
$ mx generate .
$ go run .
Hello
```

Components are the core abstraction of MX. All code in a Service
MX application runs as part of some component. The main advantage of
components is that they decouple how you *write* your code from how you *run*
your code. They let you write your application as a monolith, but when you go to
run your code, you can run components in a separate process or on a different
machine entirely. Here's a diagram illustrating this concept:

![A diagram showing off various types of MX deployments](assets/images/components.svg)

When we `go run` a MX application, all components run together in a
single process, and method calls between components are executed as regular Go
method calls. In a moment, we'll describe how to run each component in a
separate process with method calls between components executed as RPCs.

## Multiple Components

In a MX application, any component can call any other component. To
demonstrate this, we introduce a second `Reverser` component. Create a file
`reverser.go` with the following contents:

```go
package main

import (
    "context"

    "github.com/sh3lk/mx"
)

// Reverser component.
type Reverser interface {
    Reverse(context.Context, string) (string, error)
}

// Implementation of the Reverser component.
type reverser struct{
    mx.Implements[Reverser]
}

func (r *reverser) Reverse(_ context.Context, s string) (string, error) {
    runes := []rune(s)
    n := len(runes)
    for i := 0; i < n/2; i++ {
        runes[i], runes[n-i-1] = runes[n-i-1], runes[i]
    }
    return string(runes), nil
}
```

The `Reverser` component is represented by a `Reverser` interface with,
unsurprisingly, a `Reverse` method that reverses strings. The `reverser` struct
is our implementation of the `Reverser` component (as indicated by the
`mx.Implements[Reverser]` field it contains).

Next, edit the app component in `main.go` to use the `Reverser` component:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sh3lk/mx"
)

func main() {
    if err := mx.Run(context.Background(), serve); err != nil {
        log.Fatal(err)
    }
}

type app struct{
    mx.Implements[mx.Main]
    reverser mx.Ref[Reverser]
}

func serve(ctx context.Context, app *app) error {
    // Call the Reverse method.
    var r Reverser = app.reverser.Get()
    reversed, err := r.Reverse(ctx, "!dlroW ,olleH")
    if err != nil {
        return err
    }
    fmt.Println(reversed)
    return nil
}
```

The `app` struct has a new field of type `mx.Ref[Reverser]` that provides
access to the `Reverser` component.

In general, if component X uses component Y, the implementation struct for X
should contain a field of type `mx.Ref[Y]`. When an X component instance is
created, MX will automatically create the Y component as well and
will fill the `mx.Ref[Y]` field with a handle to the Y component.  The
implementation of X can call `Get()` on the `mx.Ref[Y]` field to get the Y
component, as demonstrated by the following lines in the preceding examples:

```go
    var r Reverser = app.reverser.Get()
    reversed, err := r.Reverse(ctx, "!dlroW ,olleH")
```

## Listeners

MX is designed for writing serving systems. In this section, we'll
augment our app to serve HTTP traffic using a network listener. Rewrite
`main.go` with the following contents:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/sh3lk/mx"
)

func main() {
    if err := mx.Run(context.Background(), serve); err != nil {
        log.Fatal(err)
    }
}

type app struct {
    mx.Implements[mx.Main]
    reverser mx.Ref[Reverser]
    hello    mx.Listener
}

func serve(ctx context.Context, app *app) error {
    // The hello listener will listen on a random port chosen by the operating
    // system. This behavior can be changed in the config file.
    fmt.Printf("hello listener available on %v\n", app.hello)

    // Serve the /hello endpoint.
    http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        name := r.URL.Query().Get("name")
        if name == "" {
            name = "World"
        }
        reversed, err := app.reverser.Get().Reverse(ctx, name)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        fmt.Fprintf(w, "Hello, %s!\n", reversed)
    })
    return http.Serve(app.hello, nil)
}
```

Here's an explanation of the code:

- The `hello` field in the `app` struct declares a network listener, similar to
  [`net.Listen`][net_listen].
- `http.HandleFunc(...)` registers an HTTP handler for the `/hello?name=<name>`
  endpoint that returns a reversed greeting by calling the `Reverser.Reverse`
  method.
- `http.Serve(lis, nil)` runs the HTTP server on the provided listener.

By default, all application listeners listen on a random port chosen by the
operating system. Here, we want to change this default behavior and assign a
fixed local listener port for the `hello` listener. To do so, create a
[TOML](https://toml.io) config file named `mx.toml` with
the following contents:

```toml
[single]
listeners.hello = {address = "localhost:12345"}
```

Note that the name of the listener, `hello` in this case, is derived from the
field name. You can override this behavior and specify a specific listener name
using a `"mx"` field tag like this:

```go
type app struct {
    mx.Implements[mx.Main]
    reverser mx.Ref[Reverser]
    hello    mx.Listener `mx:"my_custom_listener_name"`
}
```

Listener names must be valid [Go identifiers][identifiers]. For example, the
names `"foo"`, `"bar42"`, and `"_moo"` are legal, while `""`, `"foo bar"`, and
`"foo-bar"` are illegal.

Run `mx generate`, then `go mod tidy`, and then
`MX_CONFIG=mx.toml go run .`.
The program should print out the name of the application and a unique
deployment id. It should then block serving HTTP requests on `localhost:12345`.

```console
$ mx generate
$ go mod tidy
$ go run .
╭───────────────────────────────────────────────────╮
│ app        : hello                                │
│ deployment : 5c9753e4-c476-4f93-97a0-0ea599184178 │
╰───────────────────────────────────────────────────╯
hello listener available on 127.0.0.1:12345
...
```

In a separate terminal, curl the server to receive a reversed greeting:

```console
$ curl "localhost:12345/hello?name=MX"
Hello, revaeW!
```

Run `mx single status` to view the status of the MX application.
The status shows every deployment, component, and listener.

```console
$ mx single status
╭────────────────────────────────────────────────────╮
│ DEPLOYMENTS                                        │
├───────┬──────────────────────────────────────┬─────┤
│ APP   │ DEPLOYMENT                           │ AGE │
├───────┼──────────────────────────────────────┼─────┤
│ hello │ 5c9753e4-c476-4f93-97a0-0ea599184178 │ 1s  │
╰───────┴──────────────────────────────────────┴─────╯
╭────────────────────────────────────────────────────╮
│ COMPONENTS                                         │
├───────┬────────────┬────────────────┬──────────────┤
│ APP   │ DEPLOYMENT │ COMPONENT      │ REPLICA PIDS │
├───────┼────────────┼────────────────┼──────────────┤
│ hello │ 5c9753e4   │ main           │ 691625       │
│ hello │ 5c9753e4   │ hello.Reverser │ 691625       │
╰───────┴────────────┴────────────────┴──────────────╯
╭─────────────────────────────────────────────────╮
│ LISTENERS                                       │
├───────┬────────────┬──────────┬─────────────────┤
│ APP   │ DEPLOYMENT │ LISTENER │ ADDRESS         │
├───────┼────────────┼──────────┼─────────────────┤
│ hello │ 5c9753e4   │ hello    │ 127.0.0.1:12345 │
╰───────┴────────────┴──────────┴─────────────────╯
```

You can also run `mx single dashboard` to open a dashboard in a web browser.

## Multiprocess Execution

We've seen how to run a MX application in a single process with `go
run`. Now, we'll run our application in multiple processes, with method calls
between components executed as RPCs. First, create a [TOML](https://toml.io)
config file named `mx.toml` with the following contents:

```toml
[mx]
binary = "./hello"

[multi]
listeners.hello = {address = "localhost:12345"}
```

This config file specifies the binary of the MX application, as
well as a fixed address for the hello listener. Next, build and run the app
using `mx multi deploy`:

```console
$ go build                        # build the ./hello binary
$ mx multi deploy mx.toml # deploy the application
╭───────────────────────────────────────────────────╮
│ app        : hello                                │
│ deployment : 6b285407-423a-46cc-9a18-727b5891fc57 │
╰───────────────────────────────────────────────────╯
S1205 10:21:15.450917 stdout  26b601c4] hello listener available on 127.0.0.1:12345
S1205 10:21:15.454387 stdout  88639bf8] hello listener available on 127.0.0.1:12345
```

**Note**: `mx multi` replicates every component twice, which is why you see
two log entries. We elaborate on replication more in the
[Components](#components) section later.

In a separate terminal, curl the server:

```console
$ curl "localhost:12345/hello?name=MX"
Hello, revaeW!
```

When the main component receives your `/hello` HTTP request, it calls the
`reverser.Reverse` method. This method call is executed as an RPC to the
`Reverser` component running in a different process. Remember earlier when we
ran `mx generate`, the MX code generator? One thing that `mx
generate` does is generate RPC clients and servers for every component to make
this communication possible.

Run `mx multi status` to view the status of the MX application.
Note that the `main` and `Reverser` components are replicated twice, and every
replica is run in its own OS process.

```console
$ mx multi status
╭────────────────────────────────────────────────────╮
│ DEPLOYMENTS                                        │
├───────┬──────────────────────────────────────┬─────┤
│ APP   │ DEPLOYMENT                           │ AGE │
├───────┼──────────────────────────────────────┼─────┤
│ hello │ 6b285407-423a-46cc-9a18-727b5891fc57 │ 3s  │
╰───────┴──────────────────────────────────────┴─────╯
╭──────────────────────────────────────────────────────╮
│ COMPONENTS                                           │
├───────┬────────────┬────────────────┬────────────────┤
│ APP   │ DEPLOYMENT │ COMPONENT      │ REPLICA PIDS   │
├───────┼────────────┼────────────────┼────────────────┤
│ hello │ 6b285407   │ main           │ 695110, 695115 │
│ hello │ 6b285407   │ hello.Reverser │ 695136, 695137 │
╰───────┴────────────┴────────────────┴────────────────╯
╭─────────────────────────────────────────────────╮
│ LISTENERS                                       │
├───────┬────────────┬──────────┬─────────────────┤
│ APP   │ DEPLOYMENT │ LISTENER │ ADDRESS         │
├───────┼────────────┼──────────┼─────────────────┤
│ hello │ 6b285407   │ hello    │ 127.0.0.1:12345 │
╰───────┴────────────┴──────────┴─────────────────╯
```

You can also run `mx multi dashboard` to open a dashboard in a web browser.

## Deploying to the Cloud

The ability to run MX applications locally&mdash;either in a single
process with `go run` or across multiple processes with `mx multi
deploy`&mdash;makes it easy to quickly develop, debug, and test your
applications. When your application is ready for production, however, you'll
often want to deploy it to the cloud. MX makes this easy too.

For example, we can deploy our "Hello, World" application to [Google Kubernetes
Engine][gke], Google Cloud's hosted Kubernetes offering, as easily as running a
single command (see the [GKE](#gke) section for details):

```console
$ mx gke deploy mx.toml
```

When you run this command, MX will

- wrap your application binary into a container;
- upload the container to the cloud project of your choosing;
- create and provision the appropriate Kubernetes clusters;
- set up all load balancers and networking infrastructure; and
- deploy your application on Kubernetes, with components distributed across
  machines in multiple regions.

MX also integrates your application with existing cloud tooling.
Logs are uploaded to [Google Cloud Logging][cloud_logging], metrics are uploaded
to [Google Cloud Monitoring][cloud_metrics], traces are uploaded to [Google
Cloud Tracing][cloud_trace], etc.

## Next Steps

- Work through the exercises in our [codelab](#codelab) to get experience
  writing MX apps.
- Continue reading the docs to get a better understanding of
  [components](#components) and learn about other fundamental features of
  MX like [logging](#logging), [metrics](#metrics),
  [routing](#routing), and so on.
- Read [our blog](/blog).
- Read through [example MX applications][mx_examples] that
  demonstrate what MX has to offer.
- Dive deeper into the various ways you can deploy a MX application,
  including [single process](#single-process), [multiprocess](#multiprocess),
  [SSH](#ssh), [GKE](#gke), [Kube](#kube), and [Cloud Run](#cloud-run) deployers.
- Check out [MX's source code on GitHub][mx_github].
- Chat with us on [Discord](https://discord.gg/FzbQ3SM8R5) or send us an
  [email](mx@google.com).

# Codelab

Check out the [MX codelab][workshop] hosted on GitHub. The codelab
includes a set of exercises (with solutions) that walk you through the
implementation of [an emoji search engine application][emojis] backed by
ChatGPT. The [Step by Step Tutorial](#step-by-step-tutorial) section walked you
through the fundamentals of MX, and the codelab puts these
fundamentals to practice, giving you hands-on experience writing fully fledged
MX applications.

# Components

**Components** are MX's core abstraction. A component is a
long-lived, possibly replicated entity that exposes a set of methods.
Concretely, a component is represented as a Go interface and corresponding
implementation of that interface. Consider the following `Adder` component for
example:

```go
type Adder interface {
    Add(context.Context, int, int) (int, error)
}

type adder struct {
    mx.Implements[Adder]
}

func (*adder) Add(_ context.Context, x, y int) (int, error) {
    return x + y, nil
}
```

`Adder` defines the component's interface, and `adder` defines the component's
implementation. The two are linked with the embedded `mx.Implements[Adder]`
field. You can call `mx.Ref[Adder].Get()` to get a client to the `Adder`
component. The returned client implements the component's interface, so you can
invoke the component's methods as you would any regular Go method. When you
invoke a component's method, the method call is performed by one of the possibly
many component replicas.

Components are generally long-lived, but the MX runtime may scale up
or scale down the number of replicas of a component over time based on load.
Similarly, component replicas may fail and get restarted. MX may
also move component replicas around, co-locating two chatty components in the
same OS process, for example, so that communication between the components is
done locally rather than over the network.

When invoking a component's method, be prepared that it may be executed via
a remote procedure call. As a result, your call may fail with a network error
instead of an application error. If you don't want to deal with network errors,
you can explicitly place the two components in the same
[colocation group](#config-files), ensuring that they always run in the
same OS process.

## Interfaces

Every method in a component interface must receive a `context.Context` as its
first argument and return an `error` as its final result. All other arguments
must be [serializable](#serializable-types). These are all valid component
methods:

```go
a(context.Context) error
b(context.Context, int) error
c(context.Context) (int, error)
d(context.Context, int) (int, error)
```

These are all *invalid* component methods:

```go
a() error                          // no context.Context argument
b(context.Context)                 // no error result
c(int, context.Context) error      // first argument isn't context.Context
d(context.Context) (error, int)    // final result isn't error
e(context.Context, chan int) error // chan int isn't serializable
```

## Implementation

A component implementation must be a struct that looks like:

```go
type foo struct{
    mx.Implements[Foo]
    // ...
}
```

-   It must be a struct.
-   It must embed a `mx.Implements[T]` field where `T` is the component
    interface it implements.

If a component implementation implements an `Init(context.Context) error`
method, it will be called when an instance of the component is created.

```go
func (f *foo) Init(context.Context) error {
    // ...
}
```

If a component implementation implements an `Shutdown(context.Context) error`
method, it will be called when an instance of the component is destroyed.

```go
func (f *foo) Shutdown(context.Context) error {
    // ...
}
```

**Note**: There is no guarantee that the `Shutdown` method will always
be called. `Shutdown` is called **iff** your application receives a
`SIGINT` or a `SIGTERM` signal. However, if the machine where your application runs
crashes unexpectedly or becomes unresponsive, the `Shutdown` method is never called.

## Semantics

When implementing a component, there are a few semantic details to keep in mind:

1.  A component's state is not persisted.
2.  A component's methods may be invoked concurrently.
3.  There may be multiple replicas of a component.
4.  Component methods may be retried automatically by default.

Take the following `Cache` component for example, which maintains an in-memory
key-value cache.

```go
type Cache interface {
    Put(ctx context.Context, key, value string) error
    Get(ctx context.Context, key string) (string, error)
}

type cache struct {
    mu sync.Mutex
    data map[string]string
}

func (c *Cache) Put(_ context.Context, key, value string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
    return nil
}

func (c *Cache) Get(_ context.Context, key string) (string, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.data[key], nil
}
```

Noting the points above:

1.  A `Cache`'s state is not persisted, so if a `Cache` replica fails, its data
    is lost. Any state that needs to be persisted should be persisted
    explicitly.
2.  A `Cache`'s methods may be invoked concurrently, so it's essential that we
    guard access to `data` with the mutex `mu`.
3.  There may be multiple replicas of a `Cache` component, so it is not
    guaranteed that one client's `Get` will be routed to the same replica as
    another client's `Put`. For this example, this means that the `Cache` has
    [weak consistency][weak_consistency].

If a remote method call fails to execute properly&mdash;because of a machine
crash or a network partition, for example&mdash;it returns an error with an
embedded `mx.RemoteCallError`. Here's an illustrative example:

```go
// Call the cache.Get method.
value, err := cache.Get(ctx, "key")
if errors.Is(err, mx.RemoteCallError) {
    // cache.Get did not execute properly.
} else if err != nil {
    // cache.Get executed properly, but returned an error.
} else {
    // cache.Get executed properly and did not return an error.
}
```

Note that if a method call returns an error with an embedded
`mx.RemoteCallError`, it does *not* mean that the method never executed. The
method may have executed partially, or fully, or multiple times due to automatic
retries.

On network errors, a component method call may be retried automatically by
MX. This may cause a single method call to turn into multiple
executions of that method. In practice, many methods (e.g., read-only or
idempotent methods) work correctly even when executed more than once per call,
and this automatic retrying can help make the application more robust in the
presence of failures.

However some methods should not be retried automatically. E.g., if our cache was
extended with a method that appends a string to a cached value, automatic
retrying could cause multiple copies of the argument to be appended to the
cached value. Such methods can be specially marked to prevent automatic retries.

```go
type Cache interface{
    ...
    Append(context.Context, key, val string) error
}

// Do not retry Cache.Append.
var _ mx.NotRetriable = Cache.Append
```

## Listeners

A component implementation may wish to use one or more network listeners, e.g.,
to serve HTTP network traffic. To do so, named `mx.Listener` fields must
be added to the implementation struct. For example, the following component
implementation creates two network listeners:

```go
type impl struct{
    mx.Implements[MyComponent]
    foo mx.Listener
    Bar mx.Listener
}
```

With MX, listeners are named. By default, listeners are named
after their corresponding struct fields (e.g., `"foo"` and `"bar"` in the
above example). Alternatively, a special ````mx:"name"```` struct tag
can be added to the struct field to specify the listener name explicitly:

```go
type impl struct{
    mx.Implements[MyComponent]
    foo mx.Listener
    lis mx.Listener `mx:"bar"`
}
```

Listener names must be unique inside a given application binary, regardless of
which components they are specified in. For example, it is illegal to declare a
Listener field `"foo"` in two different component implementations structs,
unless one is renamed using the ````mx:"name"```` struct tag.

By default, all application listeners will listen on a random port chosen
by the operating system. This behavior, as well as other customization options,
can be modified in the respective deployers' configuration file. For example,
the following config file will assign addresses `"localhost:12345"` and
`"localhost:12346"` to `"foo"` and `"bar"`, respectively, when the application
is deployed using the [multiprocess](#multiprocess) deployer.

```toml
[multi]
listeners.foo = {address = "localhost:12345"}
listeners.bar = {address = "localhost:12346"}
```

## Config

MX uses [config files](#config-files), written in [TOML](#toml), to
configure how applications are run. A minimal config file, for example, simply
lists the application binary:

```toml
[mx]
binary = "./hello"
```

A config file may additionally contain deployer-specific configuration sections,
which allow you to configure the execution when a given deployer is used.
For example, the following multiprocess config will enable encrypted secure
communication via `mTLS` between components when the application is deployed using the
[multiprocess](#multiprocess) deployer:

```toml
[multi]
mtls = true
```

A config file may also contain component-specific configuration
sections, which allow you to configure the components in your application. For
example, consider the following `Greeter` component.

```go
type Greeter interface {
    Greet(context.Context, string) (string, error)
}

type greeter struct {
    mx.Implements[Greeter]
}

func (g *greeter) Greet(_ context.Context, name string) (string, error) {
    return fmt.Sprintf("Hello, %s!", name), nil
}
```

Rather than hard-coding the greeting `"Hello"`, we can provide a greeting in a
config file. First, we define a options struct.

```go
type greeterOptions struct {
    Greeting string
}
```

Next, we associate the options struct with the `greeter` implementation by
embedding the `mx.WithConfig[T]` struct.

```go
type greeter struct {
    mx.Implements[Greeter]
    mx.WithConfig[greeterOptions]
}
```

Now, we can add a `Greeter` section to the config file. The section is keyed by
the full path-prefixed name of the component.

```toml
["example.com/mypkg/Greeter"]
Greeting = "Bonjour"
```

When the `Greeter` component is created, MX will automatically parse
the `Greeter` section of the config file into a `greeterOptions` struct. You can
access the populated struct via the `Config` method of the embedded `WithConfig`
struct. For example:

```go
func (g *greeter) Greet(_ context.Context, name string) (string, error) {
    greeting := g.Config().Greeting
    if greeting == "" {
        greeting = "Hello"
    }
    return fmt.Sprintf("%s, %s!", greeting, name), nil
}
```

You can use `toml` struct tags to specify the name that should be used for a
field in a config file. For example, we can change the `greeterOptions` struct
to the following.

```go
type greeterOptions struct {
    Greeting string `toml:"my_custom_name"`
}
```

And change the config file accordingly:

```toml
["example.com/mypkg/Greeter"]
my_custom_name = "Bonjour"
```

If you run an application directly (i.e. using `go run`), you can pass the
config file using the `MX_CONFIG` environment variable:

```console
$ MX_CONFIG=mx.toml go run .
```

Or, use `mx single deploy`:

```console
$ mx single deploy mx.toml
```

## Context Propagation

You can propagate metadata information from a component method caller to the
callee. The metadata is propagated to the callee even if the caller and the callee
are not colocated in the same process.

The metadata is a map from string to string, stored in context.Context. You can
add the map to a context by calling `NewContext` and retrieve it by calling
`FromContext`:

```go
...
// Attach metadata with key "save_operation" and value "true" to the context.
// Call the Add method on the adder component.
ctx := context.Background()
ctx = metadata.NewContext(ctx, map[string]string{"save_operation": "true"})
adder.Add(ctx, 1, 2)
...
// Retrieve the metadata from the context
func (*adder) Add(ctx context.Context, x, y int) (int, error) {
    meta, ok := metadata.FromContext(ctx)
    if ok {
        save := meta["save_operation"]
        ...
    }
    ...
}
```

# Logging

<div hidden class="todo">
TODO(mwhittaker): Pick a better name for node ids?
</div>

MX provides a logging API, `mx.Logger`. By using Service
MX's logging API, you can cat, tail, search, and filter logs from every one
of your MX applications (past or present). MX also
integrates the logs into the environment where your application is deployed. If
you [deploy a MX application to Google Cloud](#gke), for example,
logs are automatically exported to [Google Cloud Logging][cloud_logging].

Use the `Logger` method of a component implementation to get a logger scoped to
the component. For example:

```go
type Adder interface {
    Add(context.Context, int, int) (int, error)
}

type adder struct {
    mx.Implements[Adder]
}

func (a *adder) Add(ctx context.Context, x, y int) (int, error) {
    // adder embeds mx.Implements[Adder] which provides the Logger method.
    logger := a.Logger(ctx)
    logger.Debug("A debug log.")
    logger.Info("An info log.")
    logger.Error("An error log.", fmt.Errorf("an error"))
    return x + y, nil
}
```

Logs look like this:

```console
D1103 08:55:15.650138 main.Adder 73ddcd04 adder.go:12 │ A debug log.
I1103 08:55:15.650149 main.Adder 73ddcd04 adder.go:13 │ An info log.
E1103 08:55:15.650158 main.Adder 73ddcd04 adder.go:14 │ An error log. err="an error"
```

The first character of a log line indicates whether the log is a [D]ebug,
[I]nfo, or [E]rror log entry. Then comes the date in `MMDD` format, followed by
the time. Then comes the component name followed by a logical node id. If two
components are co-located in the same OS process, they are given the same node
id. Then comes the file and line where the log was produced, followed finally by
the contents of the log.

MX also allows you to attach key-value attributes to log entries.
These attributes can be useful when searching and filtering logs.

```go
logger.Info("A log with attributes.", "foo", "bar")  // adds foo="bar"
```

If you find yourself adding the same set of key-value attributes repeatedly, you
can pre-create a logger that will add those attributes to all log entries:

```go
fooLogger = logger.With("foo", "bar")
fooLogger.Info("A log with attributes.")  // adds foo="bar"
```

**Note**: You can also add normal print statements to your code. These prints
will be captured and logged by MX, but they won't be associated with
a particular component, they won't have `file:line` information, and they won't
have any attributes, so we recommend you use a `mx.Logger` whenever
possible.

```console
S1027 14:40:55.210541 stdout d772dcad] This was printed by fmt.Println
```

Refer to the deployer-specific documentation to learn how to search and filter
logs for [single process](#single-process-logging),
[multiprocess](#multiprocess-logging), and [GKE](#gke-logging) deployments.

# Metrics

MX provides an API for [metrics][metric_types]; specifically
[counters][prometheus_counter], [gauges][prometheus_gauge], and
[histograms][prometheus_histogram].

- A **counter** is a number that can only increase over time. It never
  decreases. You can use a counter to measure things like the number of HTTP
  requests your program has processed so far.
- A **gauge** is a number that can increase *or* decrease over time. You can use
  a gauge to measure things like the current amount of memory your program is
  using, in bytes.
- A **histogram** is a collection of numbers that are grouped into buckets. You
  can use a histogram to measure things like the latency of every HTTP request
  your program has received so far.

MX integrates these metrics into the environment where your application is
deployed. If you [deploy a MX application to Google Cloud](#gke), for
example, metrics are automatically exported to the [Google Cloud Metrics
Explorer][metrics_explorer] where they can be queried, aggregated, and graphed.

Here's an example of how to add metrics to a simple `Adder` component.

```go
var (
    addCount = metrics.NewCounter(
        "add_count",
        "The number of times Adder.Add has been called",
    )
    addConcurrent = metrics.NewGauge(
        "add_concurrent",
        "The number of concurrent Adder.Add calls",
    )
    addSum = metrics.NewHistogram(
        "add_sum",
        "The sums returned by Adder.Add",
        []float64{1, 10, 100, 1000, 10000},
    )
)

type Adder interface {
    Add(context.Context, int, int) (int, error)
}

type adder struct {
    mx.Implements[Adder]
}

func (*adder) Add(_ context.Context, x, y int) (int, error) {
    addCount.Add(1.0)
    addConcurrent.Add(1.0)
    defer addConcurrent.Sub(1.0)
    addSum.Put(float64(x + y))
    return x + y, nil
}
```

Refer to the deployer-specific documentation to learn how to view metrics for
[single process](#single-process-metrics), [multiprocess](#multiprocess-metrics),
and [GKE](#gke-metrics) deployments.

## Labels

Metrics can also have a set of key-value labels. MX represents
labels using structs. Here's an example of how to declare and use a labeled
counter to count the parity of the argument to a `Halve` method.

```go
type halveLabels struct {
    Parity string // "odd" or "even"
}

var (
    halveCounts = metrics.NewCounterMap[halveLabels](
        "halve_count",
        "The number of values that have been halved",
    )
    oddCount = halveCounts.Get(halveLabels{"odd"})
    evenCount = halveCounts.Get(halveLabels{"even"})
)

type Halver interface {
    Halve(context.Context, int) (int, error)
}

type halver struct {
    mx.Implements[Halver]
}

func (halver) Halve(_ context.Context, val int) (int, error) {
    if val % 2 == 0 {
        evenCount.Add(1)
    } else {
        oddCount.Add(1)
    }
    return val / 2, nil
}
```

To adhere to [popular metric naming conventions][prometheus_naming], Service
MX lowercases the first letter of every label by default. The `Parity` field
for example is exported as `parity`. You can override this behavior and provide
a custom label name using a `mx` annotation.

```go
type labels struct {
    Foo string                           // exported as "foo"
    Bar string `mx:"my_custom_name"` // exported as "my_custom_name"
}
```

## Auto-Generated Metrics

MX automatically creates and maintains the following set of metrics,
which measure the count, latency, and chattiness of every component method
invocation. Every metric is labeled by the calling component as well as the
invoked component and method, and whether or not the call was local or remote.

-   `mx_method_count`: Count of MX component
    method invocations.
-   `mx_method_error_count`: Count of MX component
    method invocations that result in an error.
-   `mx_method_latency_micros`: Duration, in microseconds, of
    MX component method execution.
-   `mx_method_bytes_request`: Number of bytes in Service
    MX remote component method requests.
-   `mx_method_bytes_reply`: Number of bytes in MX
    remote component method replies.

## HTTP Metrics

MX declares the following set of HTTP related metrics.

-   `mx_http_request_count`: Count of HTTP requests.
-   `mx_http_error_count`: Count of HTTP requests resulting in a 4XX or 5XX
    response. This metric is also labeled with the returned status code.
-   `mx_http_request_latency_micros`: Duration, in microseconds, of HTTP
    request execution.
-   `mx_http_request_bytes_received`: Estimated number of bytes *received* by
    an HTTP handler.
-   `mx_http_request_bytes_returned`: Estimated number of bytes *returned* by
    an HTTP handler.

If you pass an [`http.Handler`](https://pkg.go.dev/net/http#Handler) to the
`mx.InstrumentHandler` function, it will return a new `http.Handler` that
updates these metrics automatically, labeled with the provided label. For
example:

```go
// Metrics are recorded for fooHandler with label "foo".
var mux http.ServeMux
var fooHandler http.Handler = ...
mux.Handle("/foo", mx.InstrumentHandler("foo", fooHandler))
```

# Tracing

MX relies on [OpenTelemetry][otel] to trace your application.
MX exports these traces into the environment where your application
is deployed. If you [deploy a MX application to Google Cloud](#gke),
for example, traces are automatically exported to
[Google Cloud Trace][cloud_trace].

If you pass an [`http.Handler`](https://pkg.go.dev/net/http#Handler) to the
`mx.InstrumentHandler` function, it will return a new `http.Handler` that
traces an HTTP request every second.

```go
// Tracing is enabled for one request every second.
var mux http.ServeMux
var fooHandler http.Handler = ...
mux.Handle("/foo", mx.InstrumentHandler("foo", fooHandler))
```

Alternatively, you can enable tracing manually using the [OpenTelemetry][otel]
libraries:

```go
import (
    "context"
    "fmt"
    "log"
    "net/http"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "github.com/sh3lk/mx"
)

func main() {
    if err := mx.Run(context.Background(), serve); err != nil {
        log.Fatal(err)
    }
}

type app struct {
    mx.Implements[mx.Main]
    lis mx.Listener
}

func serve(ctx context.Context, app *app) error {
    fmt.Printf("hello listener available on %v\n", app.lis)

    // Serve the /hello endpoint.
    http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, %s!\n", r.URL.Query().Get("name"))
    })

    // Create an otel handler to manually enable tracing.
    otelHandler := otelhttp.NewHandler(http.DefaultServeMux, "http")
    return http.Serve(lis, otelHandler)
}
```

Regardless of whether you use `mx.InstrumentHandler` or you manually enable
tracing, once tracing is enabled for a given HTTP request, that request
and the resulting component method calls will be automatically traced. Service
MX will collect and export the traces for you. Refer to the
deployer-specific documentation for [single process](#single-process-tracing),
[multiprocess](#multiprocess-tracing), and [GKE](#gke-tracing) to learn about
deployer-specific exporters.

The steps above are all you need to get started with tracing. If you want to add
more application-specific details to your traces, you can add attributes,
events, and errors using the context passed to registered HTTP handlers and
component methods. For example, in our `hello` example, you can add an event as
follows:

```go
http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, %s!\n", r.URL.Query().Get("name"))
    trace.SpanFromContext(r.Context()).AddEvent("writing response",
        trace.WithAttributes(
            label.String("content", "hello "),
            label.String("answer", r.URL.Query().Get("name")),
        ))
})
```

Refer to [OpenTelemetry Go: All you need to know][otel_all_you_need] to learn
more about how to add more application-specific details to your traces.

# Profiling

MX allows you to profile an entire MX application, even
one that is deployed in multiple processes across multiple machines. Service
MX profiles every individual binary and aggregates them into a single
profile that captures the performance of the application as a whole. Refer to
the deployer-specific documentation for details on how to collect profiles for
[single process](#single-process-profiling),
[multiprocess](#multiprocess-profiling), and [GKE](#gke-profiling) deployments.

# Routing

By default, when a client invokes a remote component's method, this method call
will be performed by one of possibly many component replicas, selected
arbitrarily. It is sometimes beneficial for method invocations to be routed to
*a particular* replica based on the arguments provided to the method. For
example, consider a `Cache` component that maintains an in-memory cache in front
of an underlying disk-backed key-value store:

```go
type Cache interface {
    Get(ctx context.Context, key string) (string, error)
    Put(ctx context.Context, key, value string) error
}

type cache struct {
    mx.Implements[Cache]
    // ...
}
```

To increase the cache hit ratio, we may want to route every request for a given
key to the same replica. MX supports this affinity based routing by allowing
the application to specify a router type associated with the component
implementation. For example:

```go
type cacheRouter struct{}
func (cacheRouter) Get(_ context.Context, key string) string { return key }
func (cacheRouter) Put(_ context.Context, key, value string) string { return key }
```

For every component method that needs to be routed (e.g., `Get` and `Put`), the
router type should implement an equivalent method (i.e., same name and
argument types) whose return type is the routing key. When a component's routed
method is invoked, its corresponding router method is invoked to produce a
routing key. Method invocations that produce the same key are routed to the same
replica.

A routing key can be

-   any integer (e.g., `int`, `int32`), float (i.e. `float32`, `float64`), or
    string; or
-   a struct that may optionally embed `mx.AutoMarshal`, and all remaining
    fields must be either integers, floats, or strings. (e.g.
    `struct{mx.AutoMarshal; x int; y string}`, `struct{x int; y string}`, etc )

Every router method must return the same routing key type. The following, for
example, is invalid:

```go
// ERROR: Get returns a string, but Put returns an int.
func (cacheRouter) Get(_ context.Context, key string) string { return key }
func (cacheRouter) Put(_ context.Context, key, value string) int { return 42 }
```

To associate a router with its component, embed a `mx.WithRouter[T]` field in
the component implementation where `T` is the type of the router.

```go
type cache struct {
    mx.Implements[Cache]
    mx.WithRouter[cacheRouter]
    // ...
}
```

**NOTE**: Routing is done on a best-effort basis. MX will try to route
method invocations with the same key to the same replica, but this is *not*
guaranteed. As a corollary, you should *never* depend on routing for
correctness. Only use routing to increase performance in the common case.

Also note that if a component invokes a method on a co-located component, the
method call will always be executed by the co-located component and won't be
routed.

# Storage

We expect most MX applications to persist their data in some way. For
example, an e-commerce application may store its products catalog and user
information in a database and access them while serving user requests.

By default, MX leaves the storage and retrieval of application data
up to the developer. If you're using a database, for example, you have to create
the database, pre-populate it with data, and write the code to access the
database from your MX application.

Below is an example of how database information can be passed to a simple
`Adder` component using a [config file](#components-config). First, the config
file:

```toml
["example.com/mypkg/Adder"]
Driver = "mysql"
Source = "root:@tcp(localhost:3306)/"
```

And the application that uses it:

```go
type Adder interface {
    Add(context.Context, int, int) (int, error)
}

type adder struct {
    mx.Implements[Adder]
    mx.WithConfig[config]

    db *sql.DB
}

type config struct {
    Driver string // Name of the DB driver.
    Source string // DB data source.
}

func (a *adder) Init(_ context.Context) error {
    db, err := sql.Open(a.Config().Driver, a.Config().Source)
    a.db = db
    return err
}

func (a *Adder) Add(ctx context.Context, x, y int) (int, error) {
    // Check in the database first.
    var sum int
    const q = "SELECT sum FROM table WHERE x=? AND y=?;"
    if err := a.db.QueryRowContext(ctx, q, x, y).Scan(&sum); err == nil {
        return sum, nil
    }

    // Make a best-effort attempt to store in the database.
    q = "INSERT INTO table(x, y, sum) VALUES (?, ?, ?);"
    a.db.ExecContext(ctx, q, x, y, x + y)
    return x + y, nil
}
```

A similar process can be followed to pass database information using Go flags or
environment variables.

# Testing

MX includes a `mxtest` package that you can use to test your
MX applications. The package provides a `Runner` type with `Test`
and `Bench` methods. Tests use `Runner.Test` instead of `mx.Run`. To test an
`Adder` component with an `Add` method, for example, create an `adder_test.go`
file with the following contents.

```go
package main

import (
    "context"
    "testing"

    "github.com/sh3lk/mx"
    "github.com/sh3lk/mx/mxtest"
)

func TestAdd(t *testing.T) {
     runner := mxtest.Local  // A runner that runs components in a single process
     runner.Test(t, func(t *testing.T, adder Adder) {
         ctx := context.Background()
         got, err := adder.Add(ctx, 1, 2)
         if err != nil {
             t.Fatal(err)
         }
         if want := 3; got != want {
             t.Fatalf("got %q, want %q", got, want)
         }
     })
}
```

Run `go test` to run the test. `runner.Test` will create a sub-test and within
it will create an `Adder` component and pass it to the supplied function. If you
want to test the implementation of a component, rather than its interface,
specify a pointer to the implementing struct as an argument. For example, if the
`adderImpl` struct implemented the `Adder` interface, we could write the following:

```go
runner.Test(t, func(t *testing.T, adder *adderImpl) {
    // Test adder...
})
```

Tests that want to exercise multiple components can pass a function with a
separate argument per component. Each of those components will be created and
passed to the function. Each argument can be a component interface or a pointer
to a component implementation.

```go
func TestArithmetic(t *testing.T) {
    mxtest.Local.Test(t, func(t *testing.T, adder *adderImpl, multiplier Multiplier) {
        // ...
    })
}
```

## Runners

`mxtest` provides a set of builtin Runners that differ in how they partition
components across processes and how the components communicate with each other:

1. **mxtest.Local**: Every component will be placed in the test process, and
   all component method calls will use local procedure calls, happens when you
   `go run` a MX application.
2. **mxtest.Multi**: Every component will be placed in a
   different process. This is similar to what happens when you run `mx multi
   deploy`.
3. **mxtest.RPC**: Every component will be placed in the test process, but
   all component method calls will use remote even though the callee is
   local. This mode is most useful when collecting profiles or coverage data.

Tests run using `mxtest.Local` are easier to debug and troubleshoot, but do
not test distributed execution. You should test with different runners to get
the best of both worlds (each Runner.Test call will create a new sub-test):

```go
func TestAdd(t *testing.T) {
    for _, runner := range mxtest.AllRunners() {
        runner.Test(t, func(t *testing.T, adder Adder) {
            // ...
        })
    }
}
```

## Fakes

You can replace a component implementation with a fake implementation in a test
using [`mxtest.Fake`][mxtest.Fake]. Here's an example where we replace
the real implementation of a `Clock` component with a fake implementation that
always returns a fixed time.

```go
// fakeClock is a fake implementation of the Clock component.
type fakeClock struct {
    now int64
}

// Now implements the Clock component interface. It returns the current time, in
// microseconds, since the unix epoch.
func (f *fakeClock) Now(context.Context) (int64, error) {
    return f.now, nil
}

func TestClock(t *testing.T) {
    for _, runner := range mxtest.AllRunners() {
        // Register a fake Clock implementation with the runner.
        fake := &fakeClock{100}
        runner.Fakes = append(runner.Fakes, mxtest.Fake[Clock](fake))

        // When a fake is registered for a component, all instances of that
        // component dispatch to the fake.
        runner.Test(t, func(t *testing.T, clock Clock) {
            now, err := clock.UnixMicro(context.Background())
            if err != nil {
                t.Fatal(err)
            }
            if now != 100 {
                t.Fatalf("bad time: got %d, want %d", now, 100)
            }

            fake.now = 200
            now, err = clock.UnixMicro(context.Background())
            if err != nil {
                t.Fatal(err)
            }
            if now != 200 {
                t.Fatalf("bad time: got %d, want %d", now, 200)
            }
        })
    }
}
```

## Config

You can also provide the contents of a [config file](#config-files) to a runner
by setting the `Runner.Config` field:

```go
func TestArithmetic(t *testing.T) {
    runner := mxtest.Local()
    runner.Name = "Custom"
    runner.Config = `[mx] ...`
    runner.Test(t, func(t *testing.T, adder Adder, multiplier Multiplier) {
        // ...
    })
}
```

# Versioning

Serving systems evolve over time. Whether you're fixing bugs or adding new
features, it is inevitable that you will have to roll out a new version of your
system to replace the currently running version. To maintain the availability of
their systems, people typically perform **rolling updates**, where the nodes in
a deployment are updated from the old version to the new version one by one.

During a rolling update, nodes running the old version of the code will have to
communicate with other nodes running the new version of the code. Ensuring that
a system is correct despite the possibility of these cross-version interactions
is very challenging. In
[*Understanding and Detecting Software Upgrade Failures in Distributed Systems*][update_failures_paper],
Zhang et al. perform a case study of 123 failed updates in 8 widely used
systems. They found that the majority of failures were caused by the
interactions between multiple versions of a system:

>    _About two thirds of update failures are caused by interaction between two
>    software versions that hold incompatible data syntax or semantics
>    assumption._


MX takes a different approach to rollouts and sidesteps these
complex cross-version interactions. MX ensures that client requests
are executed entirely within a single version of a system. A component in one
version will *never* communicate with a component in a different version. This
eliminates the leading cause of update failures, allowing you to roll out new
versions of your MX application safely and with less headache.

Avoiding cross-version communication is trivial for applications deployed using
[`go run`](#single-process) or [`mx multi deploy`](#multiprocess) because
every deployment runs independently from one another. Refer to the
[GKE Deployments](#gke-multi-region) and
[GKE Versioning](#gke-versioning) sections to learn how MX uses a combination
of [blue/green deployments][blue_green] and autoscaling to slowly shift traffic
from an old version of a MX application running on GKE to a new version,
avoiding cross-version communication in a resource-efficient manner.

# Single Process

## Getting Started

The simplest and easiest way to deploy a MX application is to run it
directly via `go run`. When you `go run` a MX application, every
component is co-located in a single process, and method calls between components
are executed as regular Go method calls. Refer to the [Step by Step
Tutorial](#step-by-step-tutorial) section for a full example.

```console
$ go run .
```

If you run an application using `go run`, you can provide a config file using
the `MX_CONFIG` environment variable:

```console
$ MX_CONFIG=mx.toml go run .
```

Or, you can use the `mx single deploy` command. `mx single deploy` is
practically identical to `go run .`, but it makes it easier to provide a config
file.

```console
$ mx single deploy mx.toml
```

You can run `mx single status` to view the status of all active Service
MX applications deployed using `go run`.

```console
$ mx single status
╭────────────────────────────────────────────────────╮
│ DEPLOYMENTS                                        │
├───────┬──────────────────────────────────────┬─────┤
│ APP   │ DEPLOYMENT                           │ AGE │
├───────┼──────────────────────────────────────┼─────┤
│ hello │ a4bba25b-6312-4af1-beec-447c33b8e805 │ 26s │
│ hello │ a4d4c71b-a99f-4ade-9586-640bd289158f │ 19s │
│ hello │ bc663a25-c70e-440d-b022-04a83708c616 │ 12s │
╰───────┴──────────────────────────────────────┴─────╯
╭─────────────────────────────────────────────────────╮
│ COMPONENTS                                          │
├───────┬────────────┬─────────────────┬──────────────┤
│ APP   │ DEPLOYMENT │ COMPONENT       │ REPLICA PIDS │
├───────┼────────────┼─────────────────┼──────────────┤
│ hello │ a4bba25b   │ main            │ 123450       │
│ hello │ a4bba25b   │ hello.Reverser  │ 123450       │
│ hello │ a4d4c71b   │ main            │ 903510       │
│ hello │ a4d4c71b   │ hello.Reverser  │ 903510       │
│ hello │ bc663a25   │ main            │ 489102       │
│ hello │ bc663a25   │ hello.Reverser  │ 489102       │
╰───────┴────────────┴─────────────────┴──────────────╯
╭────────────────────────────────────────────╮
│ LISTENERS                                  │
├───────┬────────────┬──────────┬────────────┤
│ APP   │ DEPLOYMENT │ LISTENER │ ADDRESS    │
├───────┼────────────┼──────────┼────────────┤
│ hello │ a4bba25b   │ hello    │ [::]:33541 │
│ hello │ a4d4c71b   │ hello    │ [::]:41619 │
│ hello │ bc663a25   │ hello    │ [::]:33319 │
╰───────┴────────────┴──────────┴────────────╯
```

You can also run `mx single dashboard` to open a dashboard in a web browser.

## Listeners

You can add `mx.Listener` fields to the component implementation to trigger
creation of network listeners (see the
[Step by Step Tutorial](#step-by-step-tutorial) section for context).

```go
type app struct {
    mx.Implements[mx.Main]
    hello    mx.Listener
}
```

When you deploy an application using `go run`, the network listeners will be
automatically created by the MX runtime. Each listener will listen
on a random port chosen by the operating system, unless a concrete address
has been specified in the singleprocess section of the
[config file](#components-config), e.g.:

```toml
[single]
listeners.hello = { address = "localhost:12345" }
```

## Logging

When you deploy a MX application with `go run`, [logs](#logging) are
printed to standard out. These logs are not persisted. You can optionally save
the logs for later analysis using basic shell constructs:

```console
$ go run . | tee mylogs.txt
```

## Metrics

Run `mx single dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via `go run
.`.  Every deployment's page has a link to the deployment's [metrics](#metrics).
The metrics are exported in [Prometheus format][prometheus] and looks something
like this:

```txt
# Metrics in Prometheus text format [1].
#
# To visualize and query the metrics, make sure Prometheus is installed on
# your local machine and then add the following stanza to your Prometheus yaml
# config file:
#
# scrape_configs:
# - job_name: 'prometheus-mx-scraper'
#   scrape_interval: 5s
#   metrics_path: /debug/mx/prometheus
#   static_configs:
#     - targets: ['127.0.0.1:43087']
#
# [1]: https://prometheus.io

# HELP example_count An example counter.
# TYPE example_count counter
example_count{mx_node="bbc9beb5"} 42
example_count{mx_node="00555c38"} 9001

# ┌─────────────────────────────────────┐
# │ MX AUTOGENERATED METRICS │
# └─────────────────────────────────────┘
# HELP mx_method_count Count of MX component method invocations
# TYPE mx_method_count counter
mx_method_count{caller="main",component="main.Example",mx_node="9fa07495",method="Foo"} 0
mx_method_count{caller="main",component="main.Example",mx_node="ee76816d",method="Foo"} 1
...
```

As the header explains, you can visualize and query the metrics by installing
Prometheus and configuring it, using the provided stanza, to periodically scrape
the `/debug/mx/prometheus` endpoint of the provided target
(`127.0.0.1:43087` in the example above). You can also inspect the metrics
manually. The metrics page shows the latest value of every metric in your
application followed by [the metrics that MX automatically creates
for you](#metrics-auto-generated-metrics).

## Profiling

Use the `mx single profile` command to collect a profile of your MX
application. Invoke the command with the id of your deployment. For example,
imagine you `go run` your MX application and it gets a deployment id
`28807368-1101-41a3-bdcb-9625e0f02ca0`.

```console
$ go run .
╭───────────────────────────────────────────────────╮
│ app        : hello                                │
│ deployment : 28807368-1101-41a3-bdcb-9625e0f02ca0 │
╰───────────────────────────────────────────────────╯
```

In a separate terminal, you can run the `mx single profile` command.

```console
$ mx single profile 28807368               # Collect a CPU profile.
$ mx single profile --duration=1m 28807368 # Adjust the duration of the profile.
$ mx single profile --type=heap 28807368   # Collect a heap profile.
```

`mx single profile` prints out the filename of the collected profile. You can
use the `go tool pprof` command to visualize and analyze the profile. For
example:

```console
$ profile=$(mx single profile <deployment>) # Collect the profile.
$ go tool pprof -http=localhost:9000 $profile   # Visualize the profile.
```

Refer to `mx single profile --help` for more details. Refer to `go tool pprof
--help` for more information on how to use pprof to analyze your profiles. Refer
to [*Profiling Go Programs*][pprof_blog] for a tutorial.

## Tracing

Run `mx single dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via `go run
.`.  Every deployment's page has a link to the deployment's [traces](#tracing)
accessible via [Perfetto][perfetto]. Here's an example of what the tracing page
looks like:

![An example trace page](assets/images/trace_single.png)

Refer to [Perfetto UI Docs](https://perfetto.dev/docs/visualization/perfetto-ui)
to learn more about how to use the tracing UI.

# Multiprocess

## Getting Started

You can use `mx multi` to deploy a MX application across
multiple processes on your local machine, with each component replica running in
a separate OS process. Create [a config file](#config-files), say `mx.toml`,
that points to your compiled MX application.

```toml
[mx]
binary = "./your_compiled_mx_binary"
```

Deploy the application using `mx multi deploy`:

```console
$ mx multi deploy mx.toml
```

Refer to the [Step by Step Tutorial](#step-by-step-tutorial) section for a full
example.

When `mx multi deploy` terminates (e.g., when you press `ctrl+c`), the
application is destroyed and all processes are terminated.

You can run `mx multi status` to view the status of all active MX
applications deployed using `mx multi`.

```console
$ mx multi status
╭────────────────────────────────────────────────────╮
│ DEPLOYMENTS                                        │
├───────┬──────────────────────────────────────┬─────┤
│ APP   │ DEPLOYMENT                           │ AGE │
├───────┼──────────────────────────────────────┼─────┤
│ hello │ a4bba25b-6312-4af1-beec-447c33b8e805 │ 26s │
│ hello │ a4d4c71b-a99f-4ade-9586-640bd289158f │ 19s │
│ hello │ bc663a25-c70e-440d-b022-04a83708c616 │ 12s │
╰───────┴──────────────────────────────────────┴─────╯
╭───────────────────────────────────────────────────────╮
│ COMPONENTS                                            │
├───────┬────────────┬─────────────────┬────────────────┤
│ APP   │ DEPLOYMENT │ COMPONENT       │ REPLICA PIDS   │
├───────┼────────────┼─────────────────┼────────────────┤
│ hello │ a4bba25b   │ main            │ 695110, 695115 │
│ hello │ a4bba25b   │ hello.Reverser  │ 193720, 398751 │
│ hello │ a4d4c71b   │ main            │ 847020, 292745 │
│ hello │ a4d4c71b   │ hello.Reverser  │ 849035, 897452 │
│ hello │ bc663a25   │ main            │ 245702, 157455 │
│ hello │ bc663a25   │ hello.Reverser  │ 997520, 225023 │
╰───────┴────────────┴─────────────────┴────────────────╯
╭────────────────────────────────────────────╮
│ LISTENERS                                  │
├───────┬────────────┬──────────┬────────────┤
│ APP   │ DEPLOYMENT │ LISTENER │ ADDRESS    │
├───────┼────────────┼──────────┼────────────┤
│ hello │ a4bba25b   │ hello    │ [::]:33541 │
│ hello │ a4d4c71b   │ hello    │ [::]:41619 │
│ hello │ bc663a25   │ hello    │ [::]:33319 │
╰───────┴────────────┴──────────┴────────────╯
```

You can also run `mx multi dashboard` to open a dashboard in a web browser.

## Listeners

You can add `mx.Listener` fields to the component implementation to trigger
creation of network listeners (see the
[Step by Step Tutorial](#step-by-step-tutorial) section for context).

```go
type app struct {
    mx.Implements[mx.Main]
    hello    mx.Listener
}
```

When you deploy an application using `mx multi deploy`, the network
listeners will be automatically created by the MX runtime.
In particular, for each listener specified in the application binary,
the runtime:

1. Creates a localhost network listener listening on a random port chosen
   by the operating system (i.e. listening on `localhost:0`).
2. Ensures that an HTTP proxy is created. This proxy forwards traffic to the
   listener. In fact, the proxy balances traffic across every replica of the
   listener. (Recall that components may be replicated, and so every component
   replica will have a different instance of the listener.)

The proxy address is by default `:0`, unless a concrete address has been
specified in the multiprocess section of the [config file](#components-config),
e.g.:

```toml
[multi]
listeners.hello = { address = "localhost:12345" }
```

## Logging

`mx multi deploy` logs to stdout. It additionally persists all log entries in
a set of files in `/tmp/mx/logs/mx-multi`. Every file contains a stream of
log entries encoded as protocol buffers. You can cat, follow, and filter these
logs using `mx multi logs`. For example:

```shell
# Display all of the application logs
mx multi logs

# Follow all of the logs (similar to tail -f).
mx multi logs --follow

# Display all of the logs for the "todo" app.
mx multi logs 'app == "todo"'

# Display all of the debug logs for the "todo" app.
mx multi logs 'app=="todo" && level=="debug"'

# Display all of the logs for the "todo" app in files called foo.go.
mx multi logs 'app=="todo" && source.contains("foo.go")'

# Display all of the logs that contain the string "error".
mx multi logs 'msg.contains("error")'

# Display all of the logs that match a regex.
mx multi logs 'msg.matches("error: file .* already closed")'

# Display all of the logs that have an attribute "foo" with value "bar".
mx multi logs 'attrs["foo"] == "bar"'

# Display all of the logs in JSON format. This is useful if you want to
# perform some sort of post-processing on the logs.
mx multi logs --format=json

# Display all of the logs, including internal system logs that are hidden by
# default.
mx multi logs --system
```

Refer to `mx multi logs --help` for a full explanation of the query language,
along with many more examples.

## Metrics

Run `mx multi dashboard` to open a dashboard in a web browser. The dashboard
has a page for every MX application deployed via `mx muli
deploy`.  Every deployment's page has a link to the deployment's
[metrics](#metrics). The metrics are exported in [Prometheus
format][prometheus] and looks something like this:

```txt
# Metrics in Prometheus text format [1].
#
# To visualize and query the metrics, make sure Prometheus is installed on
# your local machine and then add the following stanza to your Prometheus yaml
# config file:
#
# scrape_configs:
# - job_name: 'prometheus-mx-scraper'
#   scrape_interval: 5s
#   metrics_path: /debug/mx/prometheus
#   static_configs:
#     - targets: ['127.0.0.1:43087']
#
#
# [1]: https://prometheus.io

# HELP example_count An example counter.
# TYPE example_count counter
example_count{mx_node="bbc9beb5"} 42
example_count{mx_node="00555c38"} 9001

# ┌─────────────────────────────────────┐
# │ MX AUTOGENERATED METRICS │
# └─────────────────────────────────────┘
# HELP mx_method_count Count of MX component method invocations
# TYPE mx_method_count counter
mx_method_count{caller="main",component="main.Example",mx_node="9fa07495",method="Foo"} 0
mx_method_count{caller="main",component="main.Example",mx_node="ee76816d",method="Foo"} 1
...
```

As the header explains, you can visualize and query the metrics by installing
Prometheus and configuring it, using the provided stanza, to periodically scrape
the `/debug/mx/prometheus` endpoint of the provided target (e.g.,
`127.0.0.1:43087`). You can also inspect the metrics manually. The metrics page
shows the latest value of every metric in your application followed by [the
metrics that MX automatically creates for
you](#metrics-auto-generated-metrics).

## Profiling

Use the `mx multi profile` command to collect a profile of your MX
application. Invoke the command with the id of your deployment. For example,
imagine you `mx multi deploy` your MX application and it gets a deployment
id `28807368-1101-41a3-bdcb-9625e0f02ca0`.

```console
$ mx multi deploy mx.toml
╭───────────────────────────────────────────────────╮
│ app        : hello                                │
│ deployment : 28807368-1101-41a3-bdcb-9625e0f02ca0 │
╰───────────────────────────────────────────────────╯
```

In a separate terminal, you can run the `mx multi profile` command.

```console
$ mx multi profile 28807368               # Collect a CPU profile.
$ mx multi profile --duration=1m 28807368 # Adjust the duration of the profile.
$ mx multi profile --type=heap 28807368   # Collect a heap profile.
```

`mx multi profile` prints out the filename of the collected profile. You can
use the `go tool pprof` command to visualize and analyze the profile. For
example:

```console
$ profile=$(mx multi profile <deployment>) # Collect the profile.
$ go tool pprof -http=localhost:9000 $profile # Visualize the profile.
```

Refer to `mx multi profile --help` for more details. Refer to `go tool pprof
--help` for more information on how to use pprof to analyze your profiles. Refer
to [*Profiling Go Programs*][pprof_blog] for a tutorial.

## Tracing

Run `mx multi dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via
`mx multi deploy`. Every deployment's page has a link to the deployment's
[traces](#tracing) accessible via [Perfetto][perfetto]. Here's an example of
what the tracing page looks like:

![An example trace page](assets/images/trace_multi.png)

Trace events are grouped by colocation group and their corresponding replicas.
Each event has a label associated with it, based on whether the event was due to
a local or remote call. Note that the user can filter the set of events for a
particular trace by clicking on an event's `traceID` and choosing `Find slices
with the same arg value`.

Refer to [Perfetto UI Docs](https://perfetto.dev/docs/visualization/perfetto-ui)
to learn more about how to use the tracing UI.

# Kube

[Kube][kube] is a deployer that allows you to run MX applications in
any [Kubernetes][kubernetes] environment, i.e. [GKE][gke], [EKS][eks], [AKS][aks],
[minikube][minikube], etc.

Features:
* You control how to run your application (e.g., resource requirements, scaling
specifications, volumes).
* You decide how to export telemetry (e.g., traces to Jaeger, metrics to Prometheus, write custom plugins).
* You can use existing tools to deploy your application (e.g., [kubectl][kubectl],
CI/CD pipelines like [Github Actions][github_actions], [Argo CD][argocd] or
[Jenkins][jenkins]).

## Overview

The figure below shows a high level overview of the `Kube` deployer. The user
provides an application binary and a configuration file `config.yaml`. The deployer
builds a container image for the application, and generates Kubernetes resources that
enable the application to run in a Kubernetes cluster.

![Kube Overview](assets/images/kube_overview.png)

Finally, the user can use [kubectl][kubectl] or a CI/CD pipeline to deploy the application.

```console
$ kubectl apply -f deployment.yaml
```

Note that the generated Kubernetes resources encapsulate information provided by
the user in the `config.yaml`. For example, the user can colocate components into
groups ([`Foo`, `Bar`]), specify resource requirements for running pods, min and
max replicas, mount volumes, etc. More details on configuration options [here](#kube-config).

By default, the `Kube` deployer exports logs to `stdout` and discards metrics
and traces. To customize how to export telemetry data, you have to use the
`Kube` [plugin API][kube_telemetry_api] to register plugins that contains
implementations on how to export logs, metrics, and traces. [Here][kube_telemetry]
is an example of how to export metrics to [Prometheus][prometheus] and traces to
[Jaeger][jaeger]. More details on how to write plugins [here](#kube-telemetry).

**Note** that the `Kube` deployer allows you to deploy a MX application
in a single region.

## Installation

First, [ensure you have MX installed](#installation). Next, install
[Docker][docker] and [kubectl][kubectl]. Finally, install the `mx-kube` command:

```console
$ go install github.com/sh3lk/mx-kube/cmd/mx-kube@latest
```

**Note**: Make sure you've created a Kubernetes cluster before you attempt to
deploy using the `Kube` deployer.

## Getting Started

Consider again the "Hello, World!" MX application from the [Step by
Step Tutorial](#step-by-step-tutorial) section. The application runs an HTTP
server on a listener named `hello` with a `/hello?name=<name>` endpoint that
returns a `Hello, <name>!` greeting. To deploy this application on Kubernetes, first
create a [MX application config file](#config-files), say `mx.toml`,
with the following contents:

```toml
[mx]
binary = "./hello"
```

The `[mx]` section of the config file specifies the compiled Service
MX binary.

Then, create a `Kube` configuration file say `config.yaml`, with the following
contents:

```yaml
appConfig: mx.toml
repo: docker.io/mydockerid

listeners:
  - name: hello
    public: true
```

The `Kube` configuration file contains a pointer to the application config
file. It also declares the list of listeners the application should export, and
which listeners should be **public**, i.e., which listeners should be accessible
from the public internet. By default, all listeners are **private**, i.e.,
accessible only from the cluster's internal network. In our example, we declare
that the`hello` listener is public.

Deploy the application using `mx kube deploy`:

```console
$ go build .
$ mx kube deploy config.yaml
...
Building image hello:ffa65856...
...
Uploading image to docker.io/mydockerid/...
...
Generating kube deployment info ...
...
kube deployment information successfully generated
/tmp/kube_ffa65856.yaml
```

`/tmp/kube_ffa65856.yaml` contains the generated Kubernetes resources for the
"Hello, World!" application.

```yaml
# Listener Service for group github.com/sh3lk/mx/Main
apiVersion: v1
kind: Service
spec:
  type: LoadBalancer
...

---
# Deployment for group github.com/sh3lk/mx/Main
apiVersion: apps/v1
kind: Deployment
...

---
# Autoscaler for group github.com/sh3lk/mx/Main
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
...

---
# Deployment for group github.com/sh3lk/mx/examples/hello/Reverser
apiVersion: apps/v1
kind: Deployment
...

---
# Autoscaler for group github.com/sh3lk/mx/examples/hello/Reverser
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
...
```

You can simply deploy `/tmp/kube_ffa65856.yaml` as follows:

```console
$ kubectl apply -f /tmp/kube_ffa65856.yaml

role.rbac.authorization.k8s.io/pods-getter created
rolebinding.rbac.authorization.k8s.io/default-pods-getter created
configmap/config-ffa65856 created
service/hello-ffa65856 created
deployment.apps/mx-main-ffa65856-acfd658f created
horizontalpodautoscaler.autoscaling/mx-main-ffa65856-acfd658f created
deployment.apps/hello-reverser-ffa65856-58d0b71e created
horizontalpodautoscaler.autoscaling/hello-reverser-ffa65856-58d0b71e created
```

To see whether your application has been deployed, you can run `kubectl get all`.

```console
$ kubectl get all

NAME                                                   READY   STATUS    RESTARTS   AGE
pod/hello-reverser-ffa65856-58d0b71e-5c96fb875-zsjrb   1/1     Running   0          4m
pod/mx-main-ffa65856-acfd658f-86684754b-w94vc      1/1     Running   0          4m

NAME                     TYPE           CLUSTER-IP       EXTERNAL-IP      PORT(S)        AGE
service/hello-ffa65856   LoadBalancer   10.103.133.111   10.103.133.111   80:30410/TCP   4m1s

NAME                                               READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/hello-reverser-ffa65856-58d0b71e   1/1     1            1           4m1s
deployment.apps/mx-main-ffa65856-acfd658f      1/1     1            1           4m1s

NAME                                                         DESIRED   CURRENT   READY   AGE
replicaset.apps/hello-reverser-ffa65856-58d0b71e-5c96fb875   1         1         1       4m1s
replicaset.apps/mx-main-ffa65856-acfd658f-86684754b      1         1         1       4m1s

NAME                                                                   REFERENCE                                     TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
horizontalpodautoscaler.autoscaling/hello-reverser-ffa65856-58d0b71e   Deployment/hello-reverser-ffa65856-58d0b71e    1%/80%     1         10        1        4m
horizontalpodautoscaler.autoscaling/mx-main-ffa65856-acfd658f      Deployment/mx-main-ffa65856-acfd658f       2%/80%     1         10        1        4m
```

Note that by default, the `Kube` deployer generates a deployment for each
component; in this example, deployments for the `Main` and `Reverser` components.

`Kube` configures your application to autoscale using the [Kubernetes Horizontal Pod Autoscaler][hpa].
As the load on your application increases, the number of replicas of the
overloaded components will increase. Conversely, as the load on your application
decreases, the number of replicas decreases. MX can independently
scale the different components of your application, meaning that heavily loaded
components can be scaled up while lesser loaded components can simultaneously be
scaled down.

For an application running in production, you will likely want to configure DNS
to map your domain name (e.g. `hello.com`), to the address of the load balancer
(e.g., `http://10.103.133.111`). When testing and debugging an application, however,
we can also simply curl the load balancer. For example:

```console
$ curl "http://10.103.133.111/hello?name=MX"
Hello, MX!
```

The `/tmp/kube_ffa65856.yaml` header contains more details on the generated
Kubernetes resources and how to view/delete resources. For example, to delete
the resources associated with this deployment, you can run:

```console
$ kubectl delete all,configmaps --selector=mx/version=ffa65856
```

To view the application logs, you can run:

```console
$ kubectl logs -l mx/app=hello --all-containers=true

D1107 23:39:38.096525 mxn             643fc8a3 remotemxn.go:231                │ 🧶 mxn started addr="tcp://[::]:10000"
D1107 23:39:38.097369 mxn             643fc8a3 remotemxn.go:485                │ Updating components="hello.Reverser"
D1107 23:39:38.097398 mxn             643fc8a3 remotemxn.go:330                │ Constructing component="hello.Reverser"
D1107 23:39:38.097438 mxn             643fc8a3 remotemxn.go:336                │ Constructed component="hello.Reverser"
D1107 23:39:38.097443 mxn             643fc8a3 remotemxn.go:491                │ Updated component="hello.Reverser"
D1107 23:39:37.295945 mxn             49c6e04e remotemxn.go:273                │ Activated component="hello.Reverser"
D1107 23:39:38.349496 mxn             49c6e04e remotemxn.go:415                │ Connecting to remote component="hello.Reverser"
D1107 23:39:38.349587 mxn             49c6e04e remotemxn.go:515                │ Updated routing info addr="[tcp://10.244.2.74:10000]" component="hello.Reverser"
I1107 23:39:38.349646 mxn             49c6e04e call.go:690                        │ connection addr="tcp://10.244.2.74:10000" from="missing" to="disconnected"
I1107 23:39:38.350108 mxn             49c6e04e call.go:690                        │ connection addr="tcp://10.244.2.74:10000" from="disconnected" to="checking"
I1107 23:39:38.350252 mxn             49c6e04e call.go:690                        │ connection addr="tcp://10.244.2.74:10000" from="checking" to="idle"
D1107 23:39:38.358632 mxn             49c6e04e remotemxn.go:429                │ Connected to remote component="hello.Reverser"
S0101 00:00:00.000000 stdout               49c6e04e                       │ hello listener available on [::]:20000
D1107 23:39:38.360294 mxn             49c6e04e remotemxn.go:336                │ Constructed component="mx.Main"
D1107 23:39:38.360337 mxn             49c6e04e remotemxn.go:491                │ Updated component="mx.Main"
```

## Config

You can configure the `Kube` deployer using the knobs exported in the [config file][kube_config_file].

| Field          | Required? | Description                                                                                                                                                                                                                                                                              |
|----------------|-----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| appConfig      | required  | Path to the MX application config file.                                                                                                                                                                                                                                      |
| baseImage      | optional  | Name of the base image used to build the application container image. If absent, the base image is `ubuntu:rolling`.                                                                                                                                                                     |
| image          | optional  | Name of the container image `Kube` creates. If absent, the image name defaults to `<app_name>:<app_version>`.                                                                                                                                                                            |
| buildTool      | optional  | Name of the tool used to build the image `Kube` creates. If absent, the build tool name defaults to `docker`.                                                                                                                                                                            |
| repo           | optional  | Name of the repository where the container image is uploaded. If empty, the image is not pushed to a repository.                                                                                                                                                                         |
| namespace      | optional  | Name of the Kubernetes namespace where the application should be deployed. Defaults to `default`.                                                                                                                                                                                        |
| serviceAccount | optional  | Name of the Kubernetes service account under which to run the pods. If absent, it uses the default service account for your namespace.                                                                                                                                                   |
| listeners      | optional  | Options for the application listeners. If absent, default options will be used.                                                                                                                                                                                                          |
| groups         | optional  | Options for groups of colocated components. If absent, each component runs in its own group.                                                                                                                                                                                             |
| resourceSpec   | optional  | Resource requirements needed to run the pods. Should satisfy the Kubernetes [resource format][kubernetes_resources]. If absent, `Kube` will use the default resource requirements as configured by Kubernetes.                                                                           |
| scalingSpec    | optional  | Specifications on how to scale the pods using the [Kubernetes Horizontal Pod Autoscaler][hpa]. Should satisfy the Kubernetes HPA [spec format][kubernetes_hpa_spec]. If absent, default options will be used (`minReplicas=1`, `maxReplicas=10`, `CPU` metric, `averageUtilization=80)`. |
| probeSpec      | optional  | Configure Kubernetes [probes][kubernetes_probes] to monitor the healthiness, liveness and readiness of the pods. Should satisfy the Kubernetes [probes format][kubernetes_probes]. If absent, no probe is configured.                                                                    |
| storageSpec    | optional  | Options to configure Kubernetes [volumes][kubernetes_volumes] and [volume mounts][kubernetes_volumes]. If absent, no storage is configured.                                                                                                                                              |
| affinitySpec   | optional  | Options to configure Kubernetes [pod affinity][kubernetes_affinity]. By default, it ensures that different replicas of the same service are not scheduled on the same node.                                                                                                              |
| useHostNetwork | optional  | If true, application listeners use the underlying nodes' network. This behavior is generally discouraged, but it may be useful when running the application in a minikube environment.                                                                                                   |
| telemetry      | optional  | Various options how to export telemetry to your telemetry plugins.                                                                                                                                                                                                                       |

For more details on specific subfields of each configuration knob, please check
all the [configuration options][kube_config_file].

**Note**: Configuration knobs such as `resourceSpec`, `scalingSpec`, `storageSpec`
can be configured both per deployment and per group of colocated components.
However, if a field has definitions both per deployment and per group, the `Kube`
deployer will consider the per group value of the field (except for the `storageSpec`
where it considers the concatenation of both). For example in the example below,
the `Kube` deployer will run two colocation groups, where the pods that run the
`Reverser` component require at least `256Mi` memory while the pods that run the
`Main` component require at least `64Mi` memory.

```yaml
appConfig: mx.toml
repo: docker.io/mydockerid

listeners:
- name: hello
  public: true

resourceSpec:
  requests:
    memory: "64Mi"

groups:
  - name: reverser-group
    components:
      -  github.com/sh3lk/mx/examples/hello/Reverser
    resourceSpec:
      requests:
        memory: "256Mi"
```

## Telemetry

The `Kube` deployer allows you to customize how to export logs, metrics and
traces. To do that, you need to implement a wrapper deployer on top of the
`Kube` deployer using the [Kube tool][kube_telemetry_api] abstraction.

[Here][kube_telemetry] is an example on how we export metrics to [Prometheus][prometheus] and traces to
[Jaeger][jaeger].

For example, to export traces to Jaeger, you have to do the following:
1. Deploy [Jaeger][jaeger] in the Kubernetes cluster as typical Kubernetes services.
This is what someone will have to do in practice as well.
```console
$ kubectl apply -f jaeger.yaml
```
2. Write a simple binary that implements the plugin to export traces to Jaeger. The code looks as follows:

```go
// ./examples/customkube
...

const jaegerPort = 14268 // Port on which the Jaeger service is receiving traces.

func main() {
  // Implementation of how to export the traces to Jaeger.
  jaegerURL := fmt.Sprintf("http://jaeger:%d/api/traces", jaegerPort)
  endpoint := jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerURL))
  traceExporter, err := jaeger.New(endpoint)
  if err != nil {
    panic(err)
  }
  handleTraceSpans := func(ctx context.Context, spans []trace.ReadOnlySpan) error {
    return traceExporter.ExportSpans(ctx, spans)
  }

  // Invokes the `Kube` deployer with the plugin to export traces as instructed
  // by handleTraceSpans.
  tool.Run("customkube", tool.Plugins{
    HandleTraceSpans: handleTraceSpans,
  })
}
```

3. Build and deploy the application using the `customkube` deployer.
```console
$ go build
$ kubectl apply -f $(customkube deploy config.yaml)
```
4. You can access the Jaeger UI to see the MX traces for your application.

## CI/CD Pipelines

The `Kube` deployer should integrate easily with your CI/CD pipeline.
[Here][kube_github_actions] is an example on how to integrate with [Github Actions][github_actions].

We've also tried [ArgoCD][argocd] and [Jenkins][jenkins]. Please contact us on
[Discord](https://discord.gg/FzbQ3SM8R5) if you have issues integrating `Kube` with
your own CI/CD pipeline.

## Versioning

To roll out a new version of your application, simply rebuild your application and
run `mx kube deploy` again. Once you deploy the newly generated Kubernetes resources,
it will start a new tree that runs the new application version.

**Note** that it is the responsibility of the user to make sure that the new
application version behaves well, and the traffic is shifted to the new version.

We found out that typically the user starts the new version in the test cluster first.
Once it has enough confidence that the new version behaves as expected, it rollouts
the new version in the production cluster, and triggers an atomic rollout. You can
do this with the `Kube` deployer by preserving the external listener service name
across versions.

For example, if you want to do atomic rollouts across multiple versions of the
"Hello, World!" application mentioned above, you can configure the `hello`
listener as follows:

```yaml
appConfig: mx.toml
repo: docker.io/mydockerid

listeners:
  - name: hello
    public: true
    serviceName: uniqueServiceName
```

This will guarantee that every time you release a new version of the "Hello, World!"
application, the load balancer service that runs the `hello` listener will always
point out to the newest version of your application version.

# GKE

[Google Kubernetes Engine (GKE)][gke] is a Google Cloud managed service that
implements the full [Kubernetes][kubernetes] API. It supports autoscaling and
multi-cluster development, and allows you to run containerized applications in
the cloud.

You can use `mx gke` to deploy a MX application to GKE, with components
running on different machines across multiple cloud regions. The `mx gke`
command does a lot of the heavy lifting to set up GKE on your behalf. It
containerizes your application; it creates the appropriate GKE clusters; it
plumbs together all the networking infrastructure; and so on. This makes
deploying your MX application to the cloud as easy as running `mx gke
deploy`. In this section, we show you how to deploy your application using
`mx gke`. Refer to the [Local GKE](#local-gke) section to see how to simulate
a GKE deployment locally on your machine.

## Installation

First, [ensure you have MX installed](#installation). Next, install
the `mx-gke` command:

```console
$ go install github.com/sh3lk/mx-gke/cmd/mx-gke@latest
```

Install the `gcloud` command to your local machine. To do so, follow [these
instructions][gcloud_install], or run the following command and follow its
prompts:

```console
$ curl https://sdk.cloud.google.com | bash
```

After installing `gcloud`, install the required GKE authentication plugin:

```console
$ gcloud components install gke-gcloud-auth-plugin
```

, and then run the following command to initialize your local environment:

```console
$ gcloud init
```

The above command will prompt you to select the Google account and cloud project
you wish to use. If you don't have a cloud project, the command will prompt you
to create one. Make sure to select a unique project name or the command will
fail. If that happens, follow [these instructions][gke_create_project] to create
a new project, or simply run:

```console
$ gcloud projects create my-unique-project-name
```

Before you can use your cloud project, however, you must add a billing account
to it. Go to [this page][gcloud_billing] to create a new billing account, and
[this page][gcloud_billing_projects] to associate a billing account with your
cloud project.

## Getting Started

Consider again the "Hello, World!" MX application from the [Step by
Step Tutorial](#step-by-step-tutorial) section. The application runs an HTTP
server on a listener named `hello` with a `/hello?name=<name>` endpoint that
returns a `Hello, <name>!` greeting. To deploy this application to GKE, first
create a [MX config file](#config-files), say `mx.toml`, with
the following contents:

```toml
[mx]
binary = "./hello"

[gke]
regions = ["us-west1"]
listeners.hello = {is_public = true, hostname = "hello.com"}
```

The `[mx]` section of the config file specifies the compiled Service
MX binary. The `[gke]` section configures the regions where the application
is deployed (`us-west1` in this example). It also declares which listeners
should be **public**, i.e., which listeners should be accessible from the public
internet. By default, all listeners are **private**, i.e., accessible only from
the cloud project's internal network. In our example, we declare that the
`hello` listener is public.

**Note**: Per RFC1035 and RFC1123, listener names should respect the following
pattern: `^(\*\.)?[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`.
Listener names must consist of lower case alphanumeric characters or '-', and
must start and end with an alphanumeric character. No other punctuation is allowed.

All listeners deployed to GKE are configured to be health-checked by GKE
load-balancers on the `/debug/mx/healthz` URL path. MX
automatically registers a health-check handler under this URL path in the
default ServerMux, so the `hello` application requires no changes.

Deploy the application using `mx gke deploy`:

```console
$ GOOS=linux GOARCH=amd64 go build
$ mx gke deploy mx.toml
...
Deploying the application... Done
Version "8e1c640a-d87b-4020-b3dd-4efc1850756c" of app "hello" started successfully.
Note that stopping this binary will not affect the app in any way.
Tailing the logs...
...
```

The first time you deploy a MX application to a cloud project, the process
may be slow, since MX needs to configure your cloud project, create the
appropriate GKE clusters, etc. Subsequent deployments should be significantly
faster.

When `mx gke` deploys your application, it creates a global, externally
accessibly load balancer that forwards traffic to the public listeners in your
application. `mx gke deploy` prints out the IP address of this load balancer
as well as instructions on how to interact with it:

```text
NOTE: The applications' public listeners will be accessible via an
L7 load-balancer managed by MX running at the public IP address:

    http://34.149.225.62

This load-balancer uses hostname-based routing to route request to the
appropriate listeners. As a result, all HTTP(s) requests reaching this
load-balancer must have the correct "Host" header field populated. This can be
achieved in one of two ways:
...
```

For an application running in production, you will likely want to configure DNS
to map your domain name (e.g. `hello.com`), to the address of the load balancer
(e.g., `http://34.149.225.62`). When testing and debugging an application,
however, we can also simply curl the load balancer with the appropriate hostname
header. Since we configured our application to associate host name `hello.com`
with the `hello` listener, we use the following command:

```console
$ curl --header 'Host: hello.com' "http://34.149.225.63/hello?name=MX"
Hello, MX!
```

We can inspect the MX applications running on GKE using the `mx gke
status` command.

```console
$ mx gke status
╭───────────────────────────────────────────────────────────────╮
│ Deployments                                                   │
├───────┬──────────────────────────────────────┬───────┬────────┤
│ APP   │ DEPLOYMENT                           │ AGE   │ STATUS │
├───────┼──────────────────────────────────────┼───────┼────────┤
│ hello │ 20c1d756-80b5-42a7-9e73-b0d3e717516e │ 1m10s │ ACTIVE │
╰───────┴──────────────────────────────────────┴───────┴────────╯
╭──────────────────────────────────────────────────────────╮
│ COMPONENTS                                               │
├───────┬────────────┬──────────┬────────────────┬─────────┤
│ APP   │ DEPLOYMENT │ LOCATION │ COMPONENT      │ HEALTHY │
├───────┼────────────┼──────────┼────────────────┼─────────┤
│ hello │ 20c1d756   │ us-west1 │ hello.Reverser │ 2/2     │
│ hello │ 20c1d756   │ us-west1 │ main           │ 2/2     │
╰───────┴────────────┴──────────┴────────────────┴─────────╯
╭─────────────────────────────────────────────────────────────────────────────────────╮
│ TRAFFIC                                                                             │
├───────────┬────────────┬───────┬────────────┬──────────┬─────────┬──────────────────┤
│ HOST      │ VISIBILITY │ APP   │ DEPLOYMENT │ LOCATION │ ADDRESS │ TRAFFIC FRACTION │
├───────────┼────────────┼───────┼────────────┼──────────┼─────────┼──────────────────┤
│ hello.com │ public     │ hello │ 20c1d756   │ us-west1 │         │ 0.5              │
├───────────┼────────────┼───────┼────────────┼──────────┼─────────┼──────────────────┤
│ hello.com │ public     │ hello │ 20c1d756   │ us-west1 │         │ 0.5              │
╰───────────┴────────────┴───────┴────────────┴──────────┴─────────┴──────────────────╯
╭────────────────────────────╮
│ ROLLOUT OF hello           │
├─────────────────┬──────────┤
│                 │ us-west1 │
├─────────────────┼──────────┤
│ TIME            │ 20c1d756 │
│ Feb 27 21:23:07 │ 1.00     │
╰─────────────────┴──────────╯
```

`mx gke status` reports information about every app, deployment, component,
and listener in your cloud project. In this example, we have a single deployment
(with id `20c1d756`) of the `hello` app. Our app has two components (`main` and
`hello.Reverser`) each with two healthy replicas running in the `us-west1`
region. The two replicas of the `main` component each export a `hello` listener.
The global load balancer that we curled earlier balances traffic evenly across
these two listeners. The final section of the output details the rollout
schedule of the application. We'll discuss rollouts later in the
[Rollouts](#gke-multi-region) section. You can also run `mx gke dashboard`
to open a dashboard in a web browser.

<div hidden class="todo">
TODO(mwhittaker): Remove rollout section?
</div>

**Note**: `mx gke` configures GKE to autoscale your application. As the load
on your application increases, the number of replicas of the overloaded
components will increase. Conversely, as the load on your application decreases,
the number of replicas decreases. MX can independently scale the different
components of your application, meaning that heavily loaded components can be
scaled up while lesser loaded components can simultaneously be scaled down.

You can use the `mx gke kill` command to kill your deployed application.

```console
$ mx gke kill hello
WARNING: You are about to kill every active deployment of the "hello" app.
The deployments will be killed immediately and irrevocably. Are you sure you
want to proceed?

Enter (y)es to continue: y
```

## Logging

`mx gke deploy` logs to stdout. It additionally exports all log entries to
[Cloud Logging][cloud_logging].  You can cat, follow, and filter these logs from
the command line using `mx gke logs`. For example:

```shell
# Display all of the application logs
mx gke logs

# Follow all of the logs (similar to tail -f).
mx gke logs --follow

# Display all of the logs for the "todo" app.
mx gke logs 'app == "todo"'

# Display all of the debug logs for the "todo" app.
mx gke logs 'app=="todo" && level=="debug"'

# Display all of the logs for the "todo" app in files called foo.go.
mx gke logs 'app=="todo" && source.contains("foo.go")'

# Display all of the logs that contain the string "error".
mx gke logs 'msg.contains("error")'

# Display all of the logs that match a regex.
mx gke logs 'msg.matches("error: file .* already closed")'

# Display all of the logs that have an attribute "foo" with value "bar".
mx gke logs 'attrs["foo"] == "bar"'

# Display all of the logs in JSON format. This is useful if you want to
# perform some sort of post-processing on the logs.
mx gke logs --format=json

# Display all of the logs, including internal system logs that are hidden by
# default.
mx gke logs --system
```

Refer to `mx gke logs --help` for a full explanation of the query language,
along with many more examples.

You can also run `mx gke dashboard` to open a dashboard in a web browser.
The dashboard has a page for every MX application deployed via
`mx gke deploy`. Every deployment's page has a link to the deployment's logs
on [Google Cloud's Logs Explorer][logs_explorer] as shown below.

![A screenshot of MX logs in the Logs Explorer](assets/images/logs_explorer.png)

## Metrics

`mx gke` exports metrics to the
[Google Cloud Monitoring console][cloud_metrics]. You can view and graph these
metrics using the [Cloud Metrics Explorer][metrics_explorer]. When you open the
Metrics Explorer, click `SELECT A METRIC`.

![A screenshot of the Metrics Explorer](assets/images/cloud_metrics_1.png)

All MX metrics are exported under the `custom.googleapis.com` domain. Query
for `mx` to view these metrics and select the metric you're interested in.

![A screenshot of selecting a metric in Metrics Explorer](assets/images/cloud_metrics_2.png)

You can use the Metrics Explorer to graph the metric you selected.

![A screenshot of a metric graph in Metrics Explorer](assets/images/cloud_metrics_3.png)

Refer to the [Cloud Metrics][cloud_metrics] documentation for more information.

## Profiling

Use the `mx gke profile` command to collect a profile of your MX
application. Invoke the command with the name (and optionally version) of the
app you wish to profile. For example:

```console
# Collect a CPU profile of the latest version of the hello app.
$ mx gke profile hello

# Collect a CPU profile of a specific version of the hello app.
$ mx gke profile --version=8e1c640a-d87b-4020-b3dd-4efc1850756c hello

# Adjust the duration of a CPU profile.
$ mx gke profile --duration=1m hello

# Collect a heap profile.
$ mx gke profile --type=heap hello
```

`mx gke profile` prints out the filename of the collected profile. You can
use the `go tool pprof` command to visualize and analyze the profile. For
example:

```console
$ profile=$(mx gke profile <app>)         # Collect the profile.
$ go tool pprof -http=localhost:9000 $profile # Visualize the profile.
```

Refer to `mx gke profile --help` for more details.

## Tracing

Run `mx gke dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via
`mx gke deploy`. Every deployment's page has a link to the deployment's
[traces](#tracing) accessible via [Google Cloud Trace][trace_service] as shown
below.

![A screenshot of a Google Cloud Trace page](assets/images/trace_gke.png)

## Multi-Region

`mx gke` allows you to deploy a MX application to multiple
[cloud regions](https://cloud.google.com/compute/docs/regions-zones). Simply
include the regions where you want to deploy in your config file. For example:

```toml
[gke]
regions = ["us-west1", "us-east1", "asia-east2", "europe-north1"]
```

When `mx gke` deploys an application to multiple regions, it intentionally
does not deploy the application to every region right away. Instead, it performs
a **slow rollout** of the application. `mx gke` first deploys the application
to a small subset of the regions, which act as [canaries][canary]. The
application runs in the canary clusters for some time before being rolled out to
a larger subset of regions. `mx gke` continues this incremental
rollout---iteratively increasing the number of regions where the application is
deployed---until the application has been rolled out to every region specified
in the config file. Within each region, `mx gke` also slowly shifts traffic
from old application versions to new versions. We discuss this in
[the next section](#versioning).

By slowly rolling out an application across regions, `mx gke` allows you to
catch buggy releases early and mitigate the amount of damage they can cause. The
`rollout` field in a [config file](#config-files) determines the length of a
slow rollout. For example:

```toml
[mx]
rollout = "1h" # Perform a one hour slow rollout.
...
```

<div hidden class="todo">
TODO(mwhittaker): Remove this part?
</div>

You can monitor the rollout of an application using `mx gke status`. For
example, here is the rollout schedule produced by `mx gke status` for a one
hour deployment of the `hello` app across the us-central1, us-west1, us-south1,
and us-east1 regions.

```console
[ROLLOUT OF hello]
                 us-west1  us-central1  us-south1  us-east1
TIME             a838cf1d  a838cf1d     a838cf1d   a838cf1d
Nov  8 22:47:30  1.00      0.00         0.00       0.00
        +15m00s  0.50      0.50         0.00       0.00
        +30m00s  0.33      0.33         0.33       0.00
        +45m00s  0.25      0.25         0.25       0.25
```

Every row in the schedule shows the fraction of traffic each region receives
from the global load balancer. The top row is the current traffic assignment,
and each subsequent row shows the projected traffic assignment at some point in
the future. Noting that only regions with a deployed application receive
traffic, we can see the application is initially deployed in us-west1, then
slowly rolls out to us-central1, us-south1, and us-east1 in 15 minute
increments.

Also note that while the global load balancer balances traffic across regions,
once a request is received within a region, it is processed entirely within that
region. As with slow rollouts and canarying, avoiding cross-region communication
is a form of [isolation][isolation] that helps minimize the blast radius of a
misbehaving application.

## Versioning

To roll out a new version of your application as a replacement of an existing
version, simply rebuild your application and run `mx gke deploy` again.
`mx gke` will slowly roll out the new version of the application to the
regions provided in the config file, as described in the previous section. In
addition to slowly rolling out *across* regions, `mx gke` also slowly rolls
out *within* regions. Within each region, `mx gke` updates the global load
balancer to slowly shift traffic from the old version of the application to the
new version.

<div hidden class="todo">
TODO(mwhittaker): Remove this part?
</div>

We can again use `mx gke status` to monitor the rollout of a new application
version. For example, here is the rollout schedule produced by `mx gke
status` for a one hour update of the `hello` app across the us-west1 and
us-east1 regions. The new version of the app `45a521a3` is replacing the old
version `def1f485`.

```console
[ROLLOUT OF hello]
                 us-west1  us-west1  us-east1  us-east1
TIME             def1f485  45a521a3  def1f485  45a521a3
Nov  9 00:54:59  0.45      0.05      0.50      0.00
         +4m46s  0.38      0.12      0.50      0.00
         +9m34s  0.25      0.25      0.50      0.00
        +14m22s  0.12      0.38      0.50      0.00
        +19m10s  0.00      0.50      0.50      0.00
        +29m58s  0.00      0.50      0.45      0.05
        +34m46s  0.00      0.50      0.38      0.12
        +39m34s  0.00      0.50      0.25      0.25
        +44m22s  0.00      0.50      0.12      0.38
        +49m10s  0.00      0.50      0.00      0.50
```

Every row in the schedule shows the fraction of traffic that every deployment
receives in every region. The schedule shows that the new application is rolled
out in us-west1 before us-east1. Initially, the new version receives
increasingly more traffic in the us-west1 region, transitioning from 5% of the
global traffic (10% of the us-west1 traffic) to 50% of the global traffic (100%
of the us-west1 traffic) over the course of roughly 20 minutes. Ten minutes
later, this process repeats in us-east1 over the course of another 20 minutes
until the new version is receiving 100% of the global traffic. After the full
one hour rollout is complete, the old version is considered obsolete and is
deleted automatically.

**Note**: While the load balancer balances traffic across application versions,
once a request is received, it is processed entirely by the version that
received it. There is no cross-version communication.

Superficially, `mx gke`'s rollout scheme seems to require a lot of resources
because it runs two copies of the application side-by-side. In reality,
`mx gke`'s use of autoscaling makes this type of
[blue/green rollout][blue_green] resource efficient. As traffic is shifted away
from the old version, its load decreases, and the autoscaler reduces its
resource allocation. Simultaneously, as the new version receives more traffic,
its load increases, and the autoscaler begins to increase its resource
allocation. These two transitions cancel out causing the rollout to use a
roughly constant number of resources.

<div hidden class="todo">
TODO(mwhittaker): What if the new version doesn't have the same regions as
the old version? Explain what happens in this case.
</div>

## Config

You can configure `mx gke` using the `[gke]` section of a
[config file](#config-files).

```toml
[gke]
regions = ["us-west1", "us-east1"]
listeners.cat = {is_public = true, hostname = "cat.com"}
listeners.hat = {is_public = true, hostname = "hat.gg"}
```

| Field       | Required? | Description                                                                                                            |
|-------------|-----------|------------------------------------------------------------------------------------------------------------------------|
| regions     | yes       | Regions in which the MX application should be deployed. Note that at least one region should be specified. |
| image       | optional      | Base image used to build the application container image. If not specified, `image = ubuntu:rolling`.                  |
| minreplicas | optional      | Minimum number of running pods for each component. If not specified, `minreplicas = 1`.                                |
| listeners   | optional  | The application's listener options, e.g., the listeners' hostnames.                                                    |
| telemetry   | optional  | Various options how to export telemetry.                                                                               |

**Note** that by default, your MX application will be deployed in the
currently active project using the currently active account; i.e., in the project
returned by running the `gcloud config get-value project` command, and using the
account returned by running the `gcloud config get-value account` command.

If you want to deploy in a different project then you should pass the name of the
new Google Cloud project as a flag when deploying your application:

```console
$ mx gke deploy --project=new_project_name mx.toml
```

If you want to deploy using a different account then you should pass the name of the
new Google Cloud Account as a flag when deploying your application:

```console
$ mx gke deploy --account=new_account_name mx.toml
```

### Telemetry

You can configure how to export the metrics; in particular, how often to export
metrics to [Google Cloud Monitoring][cloud_metrics], and whether the framework should
export the [auto-generated metrics](#metrics-auto-generated-metrics). For example,
to export the metrics every `1 hour`, and also export the auto-generated metrics,
you can add the following configuration:

```toml
[gke]
regions = ["us-west1", "us-east1"]
...
telemetry.metrics = {export_interval = "1h", auto_generate_metrics = true}
```

By default, MX doesn't export the auto-generated metrics, and it exports
the user defined metrics every `30 seconds`.

You can also configure the minimum log level for a log entry to be exported to the
[Google Cloud Logging][cloud_logging]. For example, to export only log entries that
are `WARN` or `ERROR`, you can add the following configuration:

```toml
[gke]
regions = ["us-west1", "us-east1"]
...
telemetry.logging = {min_export_level = "WARN"}
```

**Note** that `min_export_level` takes log level values defined by [slog][slog_levels].
If not specified, MX exports all logs (`min_export_level = "DEBUG"`).

## Local GKE

[`mx gke`](#gke) lets you deploy MX applications to GKE. `mx gke-local`
is a drop-in replacement for `mx gke` that allows you to simulate GKE
deployments locally on your machine. Every `mx gke` command can be replaced
with an equivalent `mx gke-local` command. `mx gke deploy` becomes
`mx gke-local deploy`; `mx gke status` becomes `mx gke-local status`;
and so on. `mx gke-local` runs your components in simulated GKE clusters and
launches a local proxy to emulate GKE's global load balancer. `mx gke-local`
also uses [the same config as a `mx gke`](#gke-config), meaning that after you
test your application locally using `mx gke-local`, you can deploy the same
application to GKE without any code *or* config changes.

### Installation

First, [ensure you have MX installed](#installation). Next, install
the `mx-gke-local` command:

```console
$ go install github.com/sh3lk/mx-gke/cmd/mx-gke-local@latest
```

### Getting Started

In the [`mx gke`](#gke-getting-started) section, we deployed a "Hello,
World!" application to GKE using `mx gke deploy`. We can deploy the same app
locally using `mx gke-local deploy`:

```console
$ cat mx.toml
[mx]
binary = "./hello"

[gke]
regions = ["us-west1"]
listeners.hello = {is_public = true, hostname = "hello.com"}

$ mx gke-local deploy mx.toml
Deploying the application... Done
Version "a2bc7a7a-fcf6-45df-91fe-6e6af171885d" of app "hello" started successfully.
Note that stopping this binary will not affect the app in any way.
Tailing the logs...
...
```

You can run `mx gke-local status` to check the status of all the applications
deployed using `mx gke-local`.

```console
$ mx gke-local status
╭─────────────────────────────────────────────────────────────╮
│ Deployments                                                 │
├───────┬──────────────────────────────────────┬─────┬────────┤
│ APP   │ DEPLOYMENT                           │ AGE │ STATUS │
├───────┼──────────────────────────────────────┼─────┼────────┤
│ hello │ af09030c-b3a6-4d15-ba47-cd9e9e9ec2e7 │ 13s │ ACTIVE │
╰───────┴──────────────────────────────────────┴─────┴────────╯
╭──────────────────────────────────────────────────────────╮
│ COMPONENTS                                               │
├───────┬────────────┬──────────┬────────────────┬─────────┤
│ APP   │ DEPLOYMENT │ LOCATION │ COMPONENT      │ HEALTHY │
├───────┼────────────┼──────────┼────────────────┼─────────┤
│ hello │ af09030c   │ us-west1 │ hello.Reverser │ 2/2     │
│ hello │ af09030c   │ us-west1 │ main           │ 2/2     │
╰───────┴────────────┴──────────┴────────────────┴─────────╯
╭─────────────────────────────────────────────────────────────────────────────────────────────╮
│ TRAFFIC                                                                                     │
├───────────┬────────────┬───────┬────────────┬──────────┬─────────────────┬──────────────────┤
│ HOST      │ VISIBILITY │ APP   │ DEPLOYMENT │ LOCATION │ ADDRESS         │ TRAFFIC FRACTION │
├───────────┼────────────┼───────┼────────────┼──────────┼─────────────────┼──────────────────┤
│ hello.com │ public     │ hello │ af09030c   │ us-west1 │ 127.0.0.1:46539 │ 0.5              │
│ hello.com │ public     │ hello │ af09030c   │ us-west1 │ 127.0.0.1:43439 │ 0.5              │
╰───────────┴────────────┴───────┴────────────┴──────────┴─────────────────┴──────────────────╯
╭────────────────────────────╮
│ ROLLOUT OF hello           │
├─────────────────┬──────────┤
│                 │ us-west1 │
├─────────────────┼──────────┤
│ TIME            │ af09030c │
│ Feb 27 20:33:10 │ 1.00     │
╰─────────────────┴──────────╯
```

The output is, unsurprisingly, identical to that of `mx gke status`. There is
information about every app, component, and listener. Note that for this
example, `mx gke-local` is running the "Hello, World!" application in a fake
us-west1 "region", as specified in the `mx.toml` config file.

`mx gke-local` runs a proxy on port 8000 that simulates the global load
balancer used by `mx gke`. We can curl the proxy in the same way we curled
the global load balancer. Since we configured our application to associate host
name `hello.com` with the `hello` listener, we use the following command:

```console
$ curl --header 'Host: hello.com' "localhost:8000/hello?name=MX"
Hello, MX!
```

You can use the `mx gke-local kill` command to kill your deployed
application.

```console
$ mx gke-local kill hello
WARNING: You are about to kill every active deployment of the "hello" app.
The deployments will be killed immediately and irrevocably. Are you sure you
want to proceed?

Enter (y)es to continue: y
```

<div hidden class="todo">
TODO(mwhittaker): Have `mx gke-local` print instructions on how to curl the
proxy.
</div>

### Logging

`mx gke-local deploy` logs to stdout. It additionally persists all log
entries in a set of files in `/tmp/mx/logs/mx-gke-local`. Every file
contains a stream of log entries encoded as protocol buffers. You can cat,
follow, and filter these logs using `mx gke-local logs`. For example:

```shell
# Display all of the application logs
mx gke-local logs

# Follow all of the logs (similar to tail -f).
mx gke-local logs --follow

# Display all of the logs for the "todo" app.
mx gke-local logs 'app == "todo"'

# Display all of the debug logs for the "todo" app.
mx gke-local logs 'app=="todo" && level=="debug"'

# Display all of the logs for the "todo" app in files called foo.go.
mx gke-local logs 'app=="todo" && source.contains("foo.go")'

# Display all of the logs that contain the string "error".
mx gke-local logs 'msg.contains("error")'

# Display all of the logs that match a regex.
mx gke-local logs 'msg.matches("error: file .* already closed")'

# Display all of the logs that have an attribute "foo" with value "bar".
mx gke-local logs 'attrs["foo"] == "bar"'

# Display all of the logs in JSON format. This is useful if you want to
# perform some sort of post-processing on the logs.
mx gke-local logs --format=json

# Display all of the logs, including internal system logs that are hidden by
# default.
mx gke-local logs --system
```

Refer to `mx gke-local logs --help` for a full explanation of the query
language, along with many more examples.

### Metrics

In addition to running the proxy on port 8000 (see the [Getting
Started](#local-gke-getting-started)), `mx gke-local` also runs a status
server on port 8001. This server's `/metrics` endpoint exports the metrics of
all running MX applications in [Prometheus format][prometheus],
which looks like this:

```console
# HELP example_count An example counter.
# TYPE example_count counter
example_count{mx_node="bbc9beb5"} 42
example_count{mx_node="00555c38"} 9001
```

To visualize and query the metrics, make sure Prometheus is installed on your
local machine and then add the following stanza to your Prometheus yaml config
file:

```yaml
scrape_configs:
- job_name: 'prometheus-mx-scraper'
  scrape_interval: 5s
  metrics_path: /metrics
  static_configs:
    - targets: ['localhost:8001']
```

### Profiling

Use the `mx gke-local profile` command to collect a profile of your MX
application. Invoke the command with the name (and optionally version) of the
app you wish to profile. For example:

```shell
# Collect a CPU profile of the latest version of the hello app.
$ mx gke-local profile hello

# Collect a CPU profile of a specific version of the hello app.
$ mx gke-local profile --version=8e1c640a-d87b-4020-b3dd-4efc1850756c hello

# Adjust the duration of a CPU profile.
$ mx gke-local profile --duration=1m hello

# Collect a heap profile.
$ mx gke-local profile --type=heap hello
```

`mx gke-local profile` prints out the filename of the collected profile. You
can use the `go tool pprof` command to visualize and analyze the profile. For
example:

```console
$ profile=$(mx gke-local profile <app>)    # Collect the profile.
$ go tool pprof -http=localhost:9000 $profile # Visualize the profile.
```

Refer to `mx gke-local profile --help` for more details.

### Tracing

Run `mx gke-local dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via
`mx gke-local deploy`. Every deployment's page has a link to the
deployment's [traces](#tracing) accessible via [Perfetto][perfetto]. Here's an
example of what the tracing page looks like:

![An example trace page](assets/images/trace_gke_local.png)

Refer to [Perfetto UI Docs](https://perfetto.dev/docs/visualization/perfetto-ui)
to learn more about how to use the tracing UI.

### Versioning

Recall that `mx gke` performs slow rollouts
[across regions](#gke-multi-region) and
[across application versions](#versioning). `mx gke-local` simulates this
behavior locally. When you `mx gke-local deploy` an application, the
application is first rolled out to a number of canary regions before being
slowly rolled out to all regions. And within a region, the locally running proxy
slowly shifts traffic from old versions of the application to the new version of
the application. You can use `mx gke-local status`, exactly like how you use
`mx gke status`, to monitor the rollouts of your applications.

# Cloud Run

[Cloud Run][cloud_run] is a Google Cloud managed compute platform that enables
you to run stateless containers that are invocable via HTTP requests.

We provide instructions on how to run a MX application in a single
container on Cloud Run.

## Build and upload a Docker Container

First, you should create a [Docker][docker] container and upload it to [Google Artifact
Registry][gar]:

```console
$ docker build -t REGION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG .
$ docker push REGION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG
```

[These instructions][cloud_run_instr] contain more details on how to build a
container to run on [Cloud Run][cloud_run].

## Deploy to Cloud Run

Next, deploy the container to [Cloud Run][cloud_run] using `gcloud run deploy`:

```console
$ gcloud run deploy NAME --image=REGION-docker.pkg.dev/PROJECT_ID/REPO_NAME/PATH:TAG --region=REGION --allow-unauthenticated
```

This command should print out a URL that you can use to access the service. Alternatively,
you can curl the service from the command line:

```console
$ curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" URL
```

[This][cloud_run_repository] repository contains an example of a MX
application that can run on [Cloud Run][cloud_run].

# SSH [experimental]

[SSH][ssh] is a deployer that allows you to run MX applications on
a set of machines reachable via `ssh`. Note that the `SSH` deployer runs your
application's components as standalone OS processes, so you don't need
[Kubernetes][kubernetes], [Docker][docker], etc.

## Getting Started

Prerequisites:
* A set of machines reachable via `ssh`.
* You may want to set up passwordless `ssh` between your machines, otherwise you
will have to type the password for each machine when you deploy/stop an application.

Consider again the "Hello, World!" MX application from the [Step by
Step Tutorial](#step-by-step-tutorial) section. The application runs an HTTP
server on a listener named `hello` with a `/hello?name=<name>` endpoint that
returns a `Hello, <name>!` greeting. To deploy this application using the `SSH`
deployer, first create a [MX application config file](#config-files),
say `mx.toml`, with the following contents:

```toml
[mx]
binary = "./hello"

[ssh]
listeners.hello = {address = "localhost:9000"}
locations = "./ssh_locations.txt"
```

The `[mx]` section of the config file specifies the compiled Service
MX binary. The `[ssh]` section contains the set of machines where your
application should be deployed, as well as per listener configuration. The set of
machines is specified as follows in `ssh_locations.txt`:

```txt
10.100.12.31
10.100.12.32
10.100.12.33
...
```

Deploy the application using `mx ssh deploy`:

```console
$ mx ssh deploy mx.toml
```

When `mx ssh deploy` terminates (e.g., when you press `ctrl+c`), the
application is destroyed and all processes are terminated.

## Logging

`mx ssh logs` logs to stdout. Refer to `mx ssh logs --help` for details.

## Metrics

Run `mx ssh dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via `mx ssh deploy`.
Every deployment's page has a link to the deployment's [metrics](#metrics).
The metrics are exported in [Prometheus format][prometheus] and looks something
like this:

```txt
# Metrics in Prometheus text format [1].
#
# To visualize and query the metrics, make sure Prometheus is installed on
# your local machine and then add the following stanza to your Prometheus yaml
# config file:
#
# scrape_configs:
# - job_name: 'prometheus-mx-scraper'
#   scrape_interval: 5s
#   metrics_path: /debug/mx/prometheus
#   static_configs:
#     - targets: ['127.0.0.1:43087']
#
# [1]: https://prometheus.io

# HELP example_count An example counter.
# TYPE example_count counter
example_count{mx_node="bbc9beb5"} 42
example_count{mx_node="00555c38"} 9001

# ┌─────────────────────────────────────┐
# │ MX AUTOGENERATED METRICS │
# └─────────────────────────────────────┘
# HELP mx_method_count Count of MX component method invocations
# TYPE mx_method_count counter
mx_method_count{caller="main",component="main.Example",mx_node="9fa07495",method="Foo"} 0
mx_method_count{caller="main",component="main.Example",mx_node="ee76816d",method="Foo"} 1
...
```

As the header explains, you can visualize and query the metrics by installing
Prometheus and configuring it, using the provided stanza, to periodically scrape
the `/debug/mx/prometheus` endpoint of the provided target
(`127.0.0.1:43087` in the example above). You can also inspect the metrics
manually. The metrics page shows the latest value of every metric in your
application followed by [the metrics that MX automatically creates
for you](#metrics-auto-generated-metrics).

## Tracing

Run `mx ssh dashboard` to open a dashboard in a web browser. The
dashboard has a page for every MX application deployed via `mx ssh deploy`.
Every deployment's page has a link to the deployment's [traces](#tracing)
accessible via [Perfetto][perfetto]. This is similar to how you access the traces
when using the [single process](#single-process) or the [multiprocess](#multiprocess) deployer.

Refer to [Perfetto UI Docs](https://perfetto.dev/docs/visualization/perfetto-ui)
to learn more about how to use the tracing UI.

## Limitations

**Note** that the `SSH` deployer is not production ready yet, but rather it serves
as a playground to deploy a MX application on a set of machines. We
welcome contributions to make it production ready. Some limitations:

* Each component is deployed on all the machines.
* No scale up/down mechanism based on health/load signals.
* Slow rollouts not supported.
* `mx ssh profile` command not implemented.
* No integration with existing frameworks to export logs, metrics and traces.

# Serializable Types

When you invoke a component's method, the arguments to the method (and the
results returned by the method) may be serialized and sent over the network.
Thus, a component's methods may only receive and return types that Service
MX knows how to serialize, types we call **serializable**. If a component
method receives or returns a type that isn't serializable, `mx generate`
will raise an error during code generation time. The following types are
serializable:

-   All primitive types (e.g., `int`, `bool`, `string`) are serializable.
-   Pointer type `*t` is serializable if `t` is serializable.
-   Array type `[N]t` is serializable if `t` is serializable.
-   Slice type `[]t` is serializable if `t` is serializable.
-   Map type `map[k]v` is serializable if `k` and `v` are serializable.
-   Named type `t` in `type t u` is serializable if it is not recursive and one
    or more of the following are true:
    -   `t` is a protocol buffer (i.e. `*t` implements `proto.Message`);
    -   `t` implements [`encoding.BinaryMarshaler`][binary_marshaler] and
        [`encoding.BinaryUnmarshaler`][binary_unmarshaler];
    -   `u` is serializable; or
    -   `u` is a struct type that embeds `mx.AutoMarshal` (see below).

The following types are not serializable:

-   Chan type `chan t` is *not* serializable.
-   Struct literal type `struct{...}` is *not* serializable.
-   Function type `func(...)` is *not* serializable.
-   Interface type `interface{...}` is *not* serializable.

**Note**: Named struct types that don't implement `proto.Message` or
`BinaryMarshaler` and `BinaryUnmarshaler` are *not* serializable by default.
However, they can trivially be made serializable by embedding
`mx.AutoMarshal`.

```go
type Pair struct {
    mx.AutoMarshal
    x, y int
}
```

The `mx.AutoMarshal` embedding instructs `mx generate` to generate
serialization methods for the struct. Note, however, that `mx.AutoMarshal`
cannot magically make *any type* serializable. For example, `mx generate`
will raise an error for the following code because the `NotSerializable` struct
is fundamentally not serializable.

```go
// ERROR: NotSerializable cannot be made serializable.
type NotSerializable struct {
    mx.AutoMarshal
    f func()   // functions are not serializable
    c chan int // chans are not serializable
}
```

Also note that `mx.AutoMarshal` can *not* be embedded in generic structs.

```go
// ERROR: Cannot embed mx.AutoMarshal in a generic struct.
type Pair[A any] struct {
    mx.AutoMarshal
    x A
    y A
}
```

To serialize generic structs, implement `BinaryMarshaler` and
`BinaryUnmarshaler`.

## Errors

MX requires every component method to [return an
error](#components-interfaces).  If a non-nil error is returned, MX
by default transmits the textual representation of the error. Therefore any
custom information stored in the error value, or custom `Is` or `As` methods,
are not available to the caller.

Applications that need custom error information can embed a `mx.AutoMarshal`
in their custom error type. MX will then serialize and deserialize
such errors properly and make them available to the caller.

# mx generate

`mx generate` is MX's code generator. Before you compile and run a MX
application, you should run `mx generate` to generate the code MX needs
to run an application. For example, `mx generate` generates code to marshal
and unmarshal any types that may be sent over the network.

From the command line, `mx generate` accepts a list of package paths. For
example, `mx generate . ./foo` will generate code for the MX applications
in the current directory and in the `./foo` directory. For every package, the
generated code is placed in a `mx_gen.go` file in the package's directory.
Running `mx generate .  ./foo`, for example, will create `./mx_gen.go`
and `./foo/mx_gen.go`. You specify packages for `mx generate` in the same
way you specify packages for `go build`, `go test`, `go vet`, etc. Run `go help
packages` for more information.

While you can invoke `mx generate` directly, we recommend that you instead
place a line of the following form in one of the `.go` files in the root of
your module:

```go
//go:generate mx generate ./...
```

Then, you can use the [`go generate`][go_generate] command to generate all of
the `mx_gen.go` files in your module.

# Config Files

MX config files are written in [TOML](https://toml.io/en/) and look
something like this:

```toml
[mx]
name = "hello"
binary = "./hello"
args = ["these", "are", "command", "line", "arguments"]
env = ["PUT=your", "ENV=vars", "HERE="]
colocate = [
    ["main/Rock", "main/Paper", "main/Scissors"],
    ["github.com/example/sandy/PeanutButter", "github.com/example/sandy/Jelly"],
]
rollout = "1m"
```

A config file includes a `[mx]` section followed by a subset of the
following fields:

| Field | Required? | Description |
| --- | --- | --- |
| name | optional | Name of the MX application. If absent, the name of the app is derived from the name of the binary. |
| binary | required | Compiled MX application. The binary path, if not absolute, should be relative to the directory that contains the config file. |
| args | optional | Command line arguments passed to the binary. |
| env | optional | Environment variables that are set before the binary executes. |
| colocate | optional | List of colocation groups. When two components in the same colocation group are deployed, they are deployed in the same OS process, where all method calls between them are performed as regular Go method calls. To avoid ambiguity, components must be prefixed by their full package path (e.g., `github.com/example/sandy/`). Note that the full package path of the main package in an executable is `main`. |
| rollout | optional | How long it will take to roll out a new version of the application. See the [GKE Deployments](#gke-multi-region) section for more information on rollouts. |

A config file may additionally contain listener-specific and component-specific
configuration sections. See the [Component Config](#components-config) section
for details.

<div hidden class="todo">
Architecture
TODO: Explain the internals of MX.
</div>

# FAQ

### Do I need to worry about network errors when using MX?

Yes. While MX allows you to *write* your application as a single
binary, a distributed deployer (e.g., [multiprocess](#multiprocess),
[gke](#gke)), may place your components on separate processes/machines.
This means that method calls between those components will be executed as remote
procedure calls, resulting in possible network errors surfacing in your
application.

To be safe, we recommend that you assume that all cross-component method calls
involve a network, regardless of the actual component placement. If this is
overly burdensome, you can explicitly place relevant components in the same
[colocation group](#config-files), ensuring that they always run in the same OS
process.

**Note**: MX guarantees that all system errors are surfaced to the
application code as `mx.RemoteCallError`, which can be handled as described
in an [earlier section](#components-semantics).

### What types of distributed applications does MX target?

MX primarily targets distributed serving systems. These are online
systems that need to handle user requests as they arrive. A web application or
an API server are serving systems, for example. MX tailors its
feature set and runtime assumptions towards serving systems in the following
ways:

* *Network servers are integrated into the framework*. The application can
easily obtain a network listener and create an HTTP server on top of it.
* *Rollouts are built into the framework*. The user specifies the rollout
duration and the framework gradually shifts network traffic from the old
version to the new.
* *All components are replicated*. A request for a component can go to any one
of its replicas. Replicas may automatically be scaled up and down depending on
the load.

### What about data-processing applications? Can I use MX for those?

In theory, you may be able to use MX for data-processing
applications, though you will find that it provides little support for some of
the common data-processing features such as checkpointing, failure recovery,
restarts etc.

Additionally, MX's replication model means that component replicas
may automatically be scaled up and down depending on the load. This is likely
something that you wouldn't want in your data-processing application. This
scale-up/scale-down behavior translates even to the application's `main()`
function and may cause your data-processing program to run multiple times.

### Why doesn't MX provide its own data storage?

Different applications have different storage needs (e.g., global replication,
performance, SQL/NoSQL). There are also a [myriad][db_engines] of storage
systems out there that make different tradeoffs along various dimensions
(e.g., price, performance, API).

We didn't feel like we could provide enough value by inserting ourselves
into the application's data model. We also didn't want to restrict how
applications interact with their data (e.g., offline DB updates). For those
reasons, we left the choice of data storage up to the application.

### Doesn't the lack of data storage integration limit the portability of MX applications?

Yes, to a degree. If you use a globally reachable data storage system, then you
can truly run your application anywhere, removing any portability concerns.

If, however, you run your storage system inside your deployment environment
(e.g., a MySQL instance running in the Cloud VPN), then if you start your
application in a different environment (e.g., your desktop), it may not have
access to the storage system. In such cases, we generally recommend that you
create different storage systems for different application environments, and
use MX [config files](#config-files) to point your application to
the right storage system for the given execution environment.

If you're using SQL, Go's [sql package][sql_package] helps isolate your
code from some differences in the underlying storage systems. See the
MX's [chat application example][chat_example] for how to setup
your application to use the environment-local storage systems.

### Does the MX versioning approach mean I will end up running multiple instances of my app during a rollout? Isn't that expensive?

As we described in the GKE [versioning](#versioning) section, we utilize the
combination of auto-scaling and blue/green deployment to minimize the cost
of running multiple versions of the same application during rollouts.

In general, it is up to the deployer implementation to ensure that the
rollout cost is minimized. We envision that most cloud deployers will use a
similar technique to GKE to minimize their rollout costs. Other deployers
may choose to simply run full per-version serving trees, like the
[multiprocess](#multiprocess) deployer.

### MX's microservice development model is quite unique. Is it making a stand against traditional microservices development?

No. We acknowledge that there are still valid reasons why developers may
choose to run separate binaries for different microservices (e.g.,
different teams controlling their own binaries). We believe, however,
that MX's *modular monolith* model is applicable to a lot of
common use-cases and can be used in conjunction with the traditional
microservices model.

For example, a team may decide to unify all of the services in their control
into a single MX application. Cross-team interactions will still
be handled in the traditional model, with all of the versioning and development
implications that come with that model.

### Isn't writing "monoliths" a step in the wrong direction for distributed application development?

MX is trying to encourage a *modular monolith* model, where
the application is written as a single modularized binary that runs as separate
microservices. This is different from the monolith model, where the binary runs
as a single (replicated) service.

We believe that the MX's *modular monolith* model has the best of
both worlds: the ease of development of monolithic applications, with the
runtime benefits of microservices.

[actors]: https://en.wikipedia.org/wiki/Actor_model
[aks]: https://azure.microsoft.com/en-us/products/kubernetes-service
[argocd]: https://argoproj.github.io/cd/
[jenkins]: https://www.jenkins.io/
[binary_marshaler]: https://pkg.go.dev/encoding#BinaryMarshaler
[binary_unmarshaler]: https://pkg.go.dev/encoding#BinaryUnmarshaler
[blue_green]: https://docs.aws.amazon.com/whitepapers/latest/overview-deployment-options/bluegreen-deployments.html
[canary]: https://sre.google/workbook/canarying-releases/
[chat_example]: https://github.com/sh3lk/mx/tree/main/examples/chat/
[chrome_tracing]: https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU/preview
[cloud_logging]: https://cloud.google.com/logging
[cloud_metrics]: https://cloud.google.com/monitoring/api/metrics_gcp
[cloud_trace]: https://cloud.google.com/trace
[cloud_run]: https://cloud.google.com/run
[cloud_run_instr]: https://cloud.google.com/run/docs/building/containers
[cloud_run_repository]: https://github.com/mwhittaker/cloudrun
[db_engines]: https://db-engines.com/en/ranking
[docker]: https://docs.docker.com/engine/install/
[emojis]: https://emojis.mx.dev/
[eks]: https://aws.amazon.com/eks/
[gcloud_billing]: https://console.cloud.google.com/billing
[gcloud_billing_projects]: https://console.cloud.google.com/billing/projects
[gcloud_install]: https://cloud.google.com/sdk/docs/install
[github_actions]: https://github.com/features/actions
[gar]: https://cloud.google.com/artifact-registry
[gke]: https://cloud.google.com/kubernetes-engine
[gke_create_project]: https://cloud.google.com/resource-manager/docs/creating-managing-projects#gcloud
[go_generate]: https://pkg.go.dev/cmd/go/internal/generate
[go_install]: https://go.dev/doc/install
[go_interfaces]: https://go.dev/tour/methods/9
[hello_app]: https://github.com/sh3lk/mx/tree/main/examples/hello
[hpa]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
[http_pprof]: https://pkg.go.dev/net/http/pprof
[identifiers]: https://go.dev/ref/spec#Identifiers
[isolation]: https://sre.google/workbook/canarying-releases/#dependencies-and-isolation
[jaeger]: https://www.jaegertracing.io/
[kube]: https://github.com/sh3lk/mx-kube
[kubectl]: https://kubernetes.io/docs/reference/kubectl/
[kubernetes]: https://kubernetes.io/
[kube_telemetry]: https://github.com/sh3lk/mx-kube/tree/main/examples/telemetry
[kube_telemetry_api]: https://github.com/sh3lk/mx-kube/blob/main/tool/tool.go
[kube_github_actions]: https://github.com/sh3lk/mx-kube/blob/main/.github/workflows/integration.yml
[kube_config_file]: https://github.com/sh3lk/mx-kube/blob/main/internal/impl/config.go
[kubernetes_resources]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
[kubernetes_volumes]: https://kubernetes.io/docs/concepts/storage/volumes/
[kubernetes_affinity]: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
[kubernetes_hpa_spec]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-resource-metrics
[kubernetes_probes]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
[logs_explorer]: https://cloud.google.com/logging/docs/view/logs-explorer-interface
[metric_types]: https://prometheus.io/docs/concepts/metric_types/
[metrics_explorer]: https://cloud.google.com/monitoring/charts/metrics-explorer
[minikube]: https://minikube.sigs.k8s.io/docs/
[n_queens]: https://en.wikipedia.org/wiki/Eight_queens_puzzle
[net_listen]: https://pkg.go.dev/net#Listen
[otel]: https://opentelemetry.io/docs/instrumentation/go/getting-started/
[otel_all_you_need]: https://lightstep.com/blog/opentelemetry-go-all-you-need-to-know#adding-detail
[perfetto]: https://ui.perfetto.dev/
[pprof]: https://github.com/google/pprof
[pprof_blog]: https://go.dev/blog/pprof
[prometheus]: https://prometheus.io
[prometheus_counter]: https://prometheus.io/docs/concepts/metric_types/#counter
[prometheus_gauge]: https://prometheus.io/docs/concepts/metric_types/#gauge
[prometheus_histogram]: https://prometheus.io/docs/concepts/metric_types/#histogram
[prometheus_naming]: https://prometheus.io/docs/practices/naming/
[sql_package]: https://pkg.go.dev/database/sql
[ssh]: https://github.com/sh3lk/mx/tree/main/internal/tool/ssh
[slog_levels]: https://pkg.go.dev/log/slog#Level
[trace_service]: https://cloud.google.com/trace
[update_failures_paper]: https://scholar.google.com/scholar?cluster=4116586908204898847
[weak_consistency]: https://mwhittaker.github.io/consistency_in_distributed_systems/1_baseball.html
[mx_examples]: https://github.com/sh3lk/mx/tree/main/examples
[mx_github]: https://github.com/sh3lk/mx
[mxtest.Fake]: https://pkg.go.dev/github.com/sh3lk/mx/mxtest#Fake
[workshop]: https://github.com/mx/workshops
[xdg]: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
