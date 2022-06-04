**THIS REPO IS NO LONGER ACTIVE AND HAS BEEN MERGED UPSTREAM TO https://github.com/temporalio/sdk-go/tree/master/contrib/tools/workflowcheck**

# Temporal Determinist

Temporal Determinist is a tool that statically analyzes
[Temporal Go workflows](https://docs.temporal.io/docs/go/workflows/) (i.e. functions passed to
`worker.RegisterWorkflow`) to check for non-deterministic code either directly in the function or in a function called
by the workflow.

**NOTE: This will not catch all cases of non-determinism such as global var mutation. This is just a helper and
developers should still scrutinize workflow code for other non-determinisms.**

## Building

Build like a normal Go app. With this repository cloned and [Go](https://golang.org/) installed and on the `PATH`, run:

    go build

## Usage

The executable has arguments in the form:

    temporal-determinist [-flag] [package]

To simply check all source, navigate to the root of the repository/module to check and, with `temporal-determinist` on
the `PATH`, run:

    temporal-determinist ./...

Note, the source must be valid and compilable Go source for the tool to work. Run `temporal-determinist -help` for more
details on arguments.

## Determinism Rules

This tool uses a default set of non-deterministic functions/vars and an overridden set of functions/vars that are
force-set as deterministic. As of this writing, the qualified set of functions and variables that are considered
non-deterministic are:

* `crypto/rand.Reader` - Using the global crypto random reader is non-deterministic
* `math/rand.globalRand` - Using global pseudorandom is non-deterministic
* `os.Stderr` - Accessing the stderr writer is considered non-deterministic
* `os.Stdin` - Accessing the stdin reader is considered non-deterministic
* `os.Stdout` - Accessing the stdout writer is considered non-deterministic
* `time.Now` - Obtaining the current time is non-deterministic
* `time.Sleep` - Sleeping is non-deterministic

In addition to those functions/vars, the following Go source constructs are considered non-deterministic:

* Starting a goroutine
* Receiving from a channel
* Sending to a channel
* Iterating over a channel via `range`
* Iterating over a map via `range`

Many constructs that are known to be non-deterministic, such as mutating a global variable, are not able to be reliably
distinguished from deterministic use in common cases. This tool intentionally does not flag them.

In some cases, functions that are considered non-deterministic are commonly used in ways that only follow a
deterministic code path. For example if a common library function iterates over a map in a rare case that does not apply
to the situation, it will be flagged as non-deterministic. A few common cases of this have been force-set as
deterministic for common use:

* `(reflect.Value).Interface` - Default considered non-deterministic because deep down in Go internal source, this uses
  `sync.Map` (that does map iteration) as a cache of method layouts
* `runtime.Caller` - Default considered non-deterministic because deep down in Go internal source, some `runtime` source
  starts a goroutine on lazy GC start when building CGo frames
* `go.temporal.io/sdk/internal.propagateCancel` - Default considered non-deterministic because it starts a goroutine
* `(*go.temporal.io/sdk/internal.cancelCtx).cancel` - Default considered non-deterministic because it iterates over a
  map

### Overriding Rules

The `-set-decl` flag can be provided to either force-set a function/var as deterministic or non-deterministic,
overriding any defaults. Using `-set-decl DECL` will mark the function/var as non-deterministic and
`-set-decl DECL=false` will mark the function/var as deterministic. The format of `DECL` is one of:

* `/path/to/package.Function`
* `/path/to/package.Var`
* `(/path/to/package.Receiver).Function`
* `(*/path/to/package.Receiver).Function`

For example, say this function was called from a workflow:

```go
func MetricSum(metrics map[string]*Metric) (count int) {
  for _, metric := range metrics {
    count += metric.Value
  }
  return
}
```

Running `temporal-determinist ./...` might give a result like:

    /path/to/worker/main.go:29:2: path/to/package.MyWorkflow is non-deterministic, reason: calls non-determistic function path/to/package.MetricSum
      path/to/package.MetricSum is non-deterministic, reason: iterates over map

However, reading the function it does not suffer from the non-determinism inherent in map iteration. Adding a
`-set-decl` flag can mark this function as deterministic like so:

    temporal-determinist -set-decl "path/to/package.MetricSum=false" ./...

Now anytime `MetricSum` is called in a workflow, it is considered determinstic and will not be flagged.
