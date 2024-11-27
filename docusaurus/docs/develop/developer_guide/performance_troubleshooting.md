---
sidebar_position: 4
title: Performance troubleshooting
---

# Performance troubleshooting <!-- omit in toc -->

- [What is pprof](#what-is-pprof)
- [`pprof` and Dependencies - Installation](#pprof-and-dependencies---installation)
- [How to Use `pprof`](#how-to-use-pprof)
  - [Available `pprof` Endpoints](#available-pprof-endpoints)
  - [Configure Software to Expose `pprof` Endpoints](#configure-software-to-expose-pprof-endpoints)
    - [Full Nodes and Validator Configuration](#full-nodes-and-validator-configuration)
    - [RelayMiner](#relayminer)
  - [Save the Profiling Data](#save-the-profiling-data)
  - [Explore the Profiling Data](#explore-the-profiling-data)
  - [Explore without saving data](#explore-without-saving-data)
  - [Report Issues](#report-issues)

If you believe you've encountered an issue related to memory, goroutine leaks,
or some sort of synchronization blocking scenario, `pprof` is a good tool to
help identify & investigate the problem.

It is open-source and maintained by Google: [google/pprof](https://github.com/google/pprof)

## What is pprof

`pprof` is a tool for profiling and visualizing profiling data. In modern Go versions,
it is included with the compiler (`go tool pprof`), but it can also be installed as a
standalone binary from [github.com/google/pprof](https://github.com/google/pprof).

```bash
go install
```

More information can be found in the [pprof README](https://github.com/google/pprof/blob/main/doc/README.md).

## `pprof` and Dependencies - Installation

1. [Required] `pprof` - Go compiler or standalone pprof binary:

   1. pprof that comes with Golang is available via `go tool pprof`
   2. A standalone binary can be installed with:

   ```bash
    go install github.com/google/pprof@latest
   ```

2. [Optional] `graphviz` - Recommended for visualization. It can be skipped if you're not planning to use visualizations.

   - [Installation guide](https://graphviz.readthedocs.io/en/stable/#installation)
   - On MacOS, it can be installed with:

   ```bash
    brew install graphviz
   ```

## How to Use `pprof`

`pprof` operates by connecting to an exposed endpoint in the software you want to profile.

It can create snapshots for later examination, or can show information in a browser
for an already running process.

We're going to use `go tool pprof` in the examples below, but if you installed a
standalone binary, just replace `go tool pprof` with `pprof`.

### Available `pprof` Endpoints

Before running `pprof`, you need to decide what kind of profiling you need to do.

The `pprof` package provides several endpoints that are useful for profiling and
debugging. Here are the most commonly used ones:

- `/debug/pprof/heap`: Snapshot of the memory allocation of the heap.
- `/debug/pprof/allocs`: Similar to `/debug/pprof/heap`, but includes all past memory allocations, not just the ones currently in the heap.
- `/debug/pprof/goroutine`: All current go-routines.
- `/debug/pprof/threadcreate`: Records stack traces that led to the creation of new OS threads.
- `/debug/pprof/block`: Displays stack traces that led to blocking on synchronization primitives.
- `/debug/pprof/profile`: Collects 30 seconds of CPU profiling data - configurable via the `seconds` parameter.
- `/debug/pprof/symbol`: Looks up the program counters provided in the request, returning function names.
- `/debug/pprof/trace`: Provides a trace of the program execution.

### Configure Software to Expose `pprof` Endpoints

:::warning Exposing pprof

It is recommended to never expose `pprof` to the internet, as this feature allows
operational control of the software. A malicious actor could potentially disrupt
or DoS your services if these endpoints are exposed to the internet.

:::

#### Full Nodes and Validator Configuration

In `config.toml`, you can configure `pprof_laddr` to expose a `pprof` endpoint
on a particular network interface and port. By default, `pprof` listens on `localhost:6060`.

If the value has been modified, you must restart the process.

#### RelayMiner

The `RelayMiner` can be configured to expose a `pprof` endpoint using a configuration file like this:
<!-- TODO_DOC(red-0ne): Mention PATH Gateway once it has pprof support -->

```yaml
pprof:
  enabled: true
  addr: localhost:6060
```

If any of these values have been modified, you must restart the process.

### Save the Profiling Data

You can save profiling data to a file using by running:

```bash
curl -o <NAME_OF_THE_FILE_TO_CREATE> http://<YOUR_PPROF_LADDR>/<PPROF_ENDPOINT>
```

For example, a command to save a heap profile looks like this:

```bash
curl -o heap_profile.pprof http://localhost:6061/debug/pprof/heap
```

That file can be shared with other people.

### Explore the Profiling Data

Now, you can use the file to get insights into the profiling data, including visualizations.
A command like this will start an HTTP server and open a browser:

```bash
go tool pprof -http=:PORT <path_to_profile_file>
```

For example, to open a `heap_profile.pprof` from the example above, you can run:

```bash
go tool pprof -http=:3333 heap_profile.pprof
```

### Explore without saving data

It is also possible to visualize `pprof` data without saving to the file. For example:

```bash
go tool pprof -http=:3333 http://localhost:6061/debug/pprof/goroutine
```

### Report Issues

If you believe you've found a performance problem, please [open a GitHub Issue](https://github.com/pokt-network/poktroll/issues). Make sure to attach the profiling data.
