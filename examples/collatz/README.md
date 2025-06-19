# Collatz

This directory contains a MX application that explores the
[Collatz conjecture][collatz]. Given a positive integer `x`, the Collatz process
is the process of repeatedly executing the following operation:

-   If `x` is even, set `x` to `x/2`.
-   If `x` is odd, set `x` to `3x+1`.

For example, if we execute the Collatz process on `x = 10`, then we get the
sequence of numbers `10, 5, 16, 8, 4, 2, 1`. This sequence of numbers is called
the hailstone sequence of 10. The Collatz conjecture states that the hailstone
sequence of every positive number reaches 1. Nobody knows if the Collatz
conjecture is true; it is one of the most famous unsolved problems in
mathematics.

This MX application implements a service that executes the Collatz process.
You can send the service a positive number, and the server replies with that
number's hailstone sequence.

## Components

This MX application has three components: `main`, `Odd` and `Even`. Given a
positive number `x`, main repeatedly calls `Odd` (if `x` is odd) or `Even` (if
`x` is even) to receive the next number in the hailstone sequence.

```mermaid
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
graph TD
    %% Nodes.
    github.com/sh3lk/mx/Main(mx.Main)
    github.com/sh3lk/mx/examples/collatz/Even(collatz.Even)
    github.com/sh3lk/mx/examples/collatz/Odd(collatz.Odd)

    %% Edges.
    github.com/sh3lk/mx/Main --> github.com/sh3lk/mx/examples/collatz/Even
    github.com/sh3lk/mx/Main --> github.com/sh3lk/mx/examples/collatz/Odd
```

This application highlights the benefits of colocation. The performance of the
application improves significantly when `main`, `Odd`, and `Even` are colocated
in the same OS process.

## Running Locally

To run this app locally, run `go run .`. In a separate terminal, curl the
service (which defaults to `localhost:9000`):

```console
$ go run .                  # In one terminal.
$ curl localhost:9000?x=27  # In a different terminal.
```

To run the app across multiple processes, use `mx multi deploy`. The
`mx.toml` config file runs every component in a separate process, and the
`colocated.toml` config file colocates all three components.

```console
$ mx multi deploy mx.toml     # not colocated
$ mx multi deploy colocated.toml # colocated
```

## Running on GKE

To run the app on GKE, use `mx gke deploy`:

```console
$ mx gke deploy mx.toml
```

[collatz]: https://en.wikipedia.org/wiki/Collatz_conjecture
