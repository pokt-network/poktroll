---
sidebar_position: 4
title: Performance troubleshooting
---

# Performance troubleshooting  <!-- omit in toc -->

- [What is pprof](#what-is-pprof)
- [`pprof` and Dependencies - Installation](#pprof-and-dependencies---installation)
- [How to Use `pprof`](#how-to-use-pprof)
  - [Available `pprof` Endpoints](#available-pprof-endpoints)
  - [Configure Software to Expose `pprof` Endpoints](#configure-software-to-expose-pprof-endpoints)
    - [Full Nodes and Validators Configuration](#full-nodes-and-validators-configuration)
    - [AppGate Server and RelayMiner](#appgate-server-and-relayminer)
  - [Save the Profiling Data](#save-the-profiling-data)
  - [Explore the Profiling Data](#explore-the-profiling-data)
  - [Explore without saving data](#explore-without-saving-data)
  - [Report Issues](#report-issues)


If you believe you've encountered an issue related to memory, goroutine leaks, or some sort of synchronization blocking scenario, `pprof` is a good tool to help identify the problem.

## What is pprof

`pprof` is a tool for profiling and visualizing profiling data. In modern Go versions, it is included with the compiler (`go tool pprof`), but it can also be installed as a standalone binary (`go install github.com/google/pprof@latest`).

More information can be found in the [pprof README](https://github.com/google/pprof/blob/main/doc/README.md).

## `pprof` and Dependencies - Installation

1. `pprof` - Go compiler or standalone pprof binary:
   1. Go compiler pprof command is `go tool pprof`.
   2. Standalone binary can be installed with `go install github.com/google/pprof@latest`.
2. `graphviz` - Optional, but recommended for visualization. It can be skipped if you're not planning to use visualizations. [Installation guide](https://graphviz.readthedocs.io/en/stable/#installation) (on Mac, it can be installed using `brew`: `brew install graphviz`).

## How to Use `pprof`

Our software can be configured to expose an endpoint that can be used by `pprof` to connect to. `pprof` can create snapshots for later examination, or can show information in a browser for an already running process.

We're going to use `go tool pprof` in the examples below, but if you installed a standalone binary, just replace `go tool pprof` with `pprof`.

### Available `pprof` Endpoints

Before running `pprof`, you need to decide what kind of profiling you want to do. The `pprof` package provides several endpoints that are useful for profiling and debugging. Here are the most commonly used ones:

- `/debug/pprof/heap`: Provides a snapshot of the memory allocation of the heap.
- `/debug/pprof/goroutine`: Shows all current goroutines.
- `/debug/pprof/threadcreate`: Records stack traces that led to the creation of new OS threads.
- `/debug/pprof/block`: Displays stack traces that led to blocking on synchronization primitives.
- `/debug/pprof/allocs`: Similar to `/debug/pprof/heap`, but includes all past memory allocations, not just the ones currently in the heap.
- `/debug/pprof/profile`: By default, it collects 30 seconds of CPU profiling data (can be adjusted by the `seconds` parameter).
- `/debug/pprof/symbol`: Looks up the program counters provided in the request, returning function names.
- `/debug/pprof/trace`: Provides a trace of the program execution.

### Configure Software to Expose `pprof` Endpoints

:::tip

It is recommended to never expose `pprof` to the internet, as this feature allows operational control of the software. A malicious actor could potentially disrupt or DoS your services if these endpoints are exposed to the internet.

:::

#### Full Nodes and Validators Configuration

In `config.toml`, you can configure `pprof_laddr` to expose a `pprof` endpoint on a particular network interface and port. By default, `pprof` listens on `localhost:6060`. 

If the value has been modified, you need to restart the process.

#### AppGate Server and RelayMiner

Both AppGate Server and RelayMiner can be configured to expose a pprof endpoint using a configuration file like this:

```yaml
pprof:
  enabled: true
  addr: localhost:6060
```

If any of these values have been modified, you need to restart the process.

### Save the Profiling Data

Save profiling data to a file using a command like this:
```bash
curl -o NAME_OF_THE_FILE_TO_CREATE http://YOUR_PPROF_LADDR/PPROF_ENDPOINT
```

For example, a command to save a heap profile looks like this:

```bash
curl -o heap_profile.pprof http://localhost:6061/debug/pprof/heap
```

That file can be shared with other people.

### Explore the Profiling Data

Now, you can use the file to get insights into the profiling data, including visualizations. A command like this will start an HTTP server and open a browser:

```bash
go tool pprof -http=:PORT path_to_profile_file
```

For example, to open a `heap_profile.pprof` from the example above, you can run:

```bash
go tool pprof -http=:3333 heap_profile.pprof
```

### Explore without saving data

It is also possible to visualize `pprof` data without saving to the file. Example:

```bash
go tool pprof -http=:3333 http://localhost:6061/debug/pprof/goroutine
```

### Report Issues

If you believe you've found a performance problem, please [open a GitHub Issue](https://github.com/pokt-network/poktroll/issues). Make sure to attach the profiling data.
