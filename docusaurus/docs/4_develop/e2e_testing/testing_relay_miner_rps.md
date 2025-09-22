---
sidebar_position: 2
title: RelayMiner RPS Testing
---

# ⚡ RelayMiner RPS Testing

High-performance testing suite for evaluating RelayMiner throughput, latency, and performance bottlenecks under extreme load conditions.

- [⚡ RelayMiner RPS Testing](#-relayminer-rps-testing)
  - [Overview](#overview)
  - [🚀 Quick Start](#-quick-start)
  - [🧪 Test Types](#-test-types)
    - [🎯 Baseline Static Server Test](#-baseline-static-server-test)
    - [⚡ RelayMiner Only Load Test](#-relayminer-only-load-test)
  - [🔧 Infrastructure Components](#-infrastructure-components)
    - [📊 Instruction-Level Timing Metrics](#-instruction-level-timing-metrics)
    - [🌐 High-Performance Nginx Server](#-high-performance-nginx-server)
    - [🔨 wrk2 Load Testing Tool](#-wrk2-load-testing-tool)
    - [📈 RelayMiner Custom HTTP Client](#-relayminer-custom-http-client)
  - [🏃 Running Tests](#-running-tests)
    - [⚠️ Prerequisites](#️-prerequisites)
    - [🔧 LocalNet Setup](#-localnet-setup)
    - [📋 Test Parameters](#-test-parameters)
  - [📊 Monitoring Results](#-monitoring-results)
    - [🖥️ Test Output](#️-test-output)
    - [📈 Grafana Dashboards](#-grafana-dashboards)
    - [🔍 Performance Analysis](#-performance-analysis)
  - [⚙️ Configuration](#️-configuration)
    - [🚨 Session Validity Warning](#-session-validity-warning)
  - [🎯 Available Commands](#-available-commands)

## Overview

The RelayMiner RPS testing suite provides high-throughput performance testing capabilities
designed to stress-test individual RelayMiner instances under extreme load.
Unlike the comprehensive load testing suite, these tests focus specifically on
RelayMiner performance bottlenecks, request processing latency, and throughput optimization.

Key capabilities:
- **Instruction-level timing analysis** with granular performance metrics
- **Baseline performance testing** against static servers for comparison
- **Real RelayRequest generation** with proper authentication and signing
- **High-concurrency load testing** with configurable parameters
- **PATH bypass for isolation** - Tests hit RelayMiner directly to isolate performance bottlenecks
- **Real-time Grafana metrics** - Live instruction-level timing and performance dashboards

## 🚀 Quick Start

:::tip Get Started in 3 Steps
The fastest way to test RelayMiner performance:

1. Ensure [LocalNet](../networks/localnet.md) is running
2. Run a baseline test to establish performance limits:
   ```bash
   make test_baseline_static_server_load
   ```
3. Run the RelayMiner load test:
   ```bash
   make test_relay_miner_only_load
   ```
:::

## 🧪 Test Types

### 🎯 Baseline Static Server Test

:::note Performance Baseline
This test establishes the theoretical maximum performance by testing against a
high-performance nginx server that returns static JSON responses.
Use this to identify infrastructure bottlenecks vs. RelayMiner-specific performance issues.
:::

**What it tests:**
- Network infrastructure capacity
- Load testing tool overhead
- Kubernetes networking performance
- Maximum theoretical RPS for the test environment

**Default Configuration:**
- Rate: 100,000 requests per second
- Threads: 16 worker threads
- Connections: 5,000 concurrent connections
- Duration: 30 seconds

**Command:**
```bash
make test_baseline_static_server_load
```

**Custom parameters:**
```bash
# Light load test
make test_baseline_static_server_load R=10000 C=1000

# Quick test
make test_baseline_static_server_load R=5000 C=500 D=10s

# Maximum load test
make test_baseline_static_server_load R=200000 T=32 C=15000 D=45s
```

### ⚡ RelayMiner Only Load Test

:::warning Session Validity Critical
This test generates real RelayRequest data with proper cryptographic signatures.
**Sessions expire periodically** (typically every few blocks), which will cause ALL
requests to fail validation once the session changes during your test.
:::

**What it tests:**
- RelayMiner request processing performance
- Signature verification overhead
- Backend service communication latency
- Request validation and response serialization
- Resource contention under high load

**Default Configuration:**
- Rate: 512 requests per second
- Threads: 16 worker threads
- Connections: 256 concurrent connections
- Duration: 300 seconds (5 minutes)

**Command:**
```bash
make test_relayminer_only_load
```

**Custom parameters:**
```bash
# Higher rate, shorter duration
make test_relayminer_only_load R=1000 d=60s

# Light load test
make test_relayminer_only_load R=100 t=4 c=50 d=30s

# Heavy load test
make test_relayminer_only_load R=2000 t=32 c=1000
```

## 🔧 Infrastructure Components

### 📊 Instruction-Level Timing Metrics

The RelayMiner now includes granular timing instrumentation that measures the
duration between each step of relay processing.

**Key Instructions Tracked:**
- `init_request_logger` - Initial request setup
- `get_start_block` - Blockchain state retrieval
- `new_relay_request` - Request parsing and validation
- `relay_request_basic_validation` - Basic request validation
- `pre_request_verification` - Cryptographic verification
- `build_service_backend_request` - Backend request preparation
- `http_client_do` - Backend service call
- `serialize_http_response` - Response serialization
- `response_sent` - Final response transmission

**Metrics Available:**
- `RelayMiner_instruction_time_seconds` - Histogram of instruction durations
- Average duration per instruction
- 99th percentile latency per instruction

### 🌐 High-Performance Nginx Server

A highly optimized nginx server (`nginx-chainid`) provides static JSON-RPC responses for baseline testing.

**Optimizations:**
- **Worker processes**: Auto-scaled to available CPU cores
- **Connection limits**: 65,536 worker connections with Linux epoll
- **Keep-alive**: 10,000 requests per connection, 300s timeout
- **Logging disabled**: Maximum performance with access_log off
- **Buffer optimization**: Tuned client buffers and output buffers
- **HTTP/2 support**: Enabled for connection multiplexing

**Response:** Returns static `{"jsonrpc":"2.0","id":1,"result":"0x1"}` for all requests

### 🔨 wrk2 Load Testing Tool

Modern HTTP benchmarking tool with constant rate limiting and accurate latency measurement.

**Key Features:**
- **Constant rate limiting**: Maintains precise RPS regardless of latency
- **Latency accuracy**: True latency measurement
- **Lua scripting**: Custom request generation with proper headers
- **Thread scaling**: Configurable worker threads for high concurrency

**Access:** Available in `Tilt` k8s as `wrk2` deployment

### 📈 RelayMiner Custom HTTP Client

Enhanced HTTP client with performance optimizations and detailed debugging capabilities.

**Performance Features:**
- **Connection pooling**: Scaled with concurrency limits
- **Buffer management**: Reusable byte buffers to reduce GC pressure
- **Concurrency limiting**: Semaphore-based admission control
- **Timeout optimization**: Granular timeout control per request

**Debug Capabilities:**
- **Phase timing**: DNS, connection, TLS, request, response phases
- **Connection reuse tracking**: Monitor pool effectiveness
- **Error categorization**: Timeout vs. connection vs. other errors
- **Resource monitoring**: Active requests, goroutine counts

## 🏃 Running Tests

### ⚠️ Prerequisites

Before running RPS tests, ensure your environment is configured:

:::danger Critical Setup Steps
- **Configure consensus timeouts** to 30s in `config.yml` for stable sessions
- [LocalNet](../networks/localnet.md) must be running with proper configuration
- Run `make acc_initialize_pub_keys` to initialize blockchain accounts
- Verify sufficient system resources (CPU, memory, file descriptors)
:::

### 🔧 LocalNet Setup

Proper LocalNet configuration is essential for reliable RPS testing.

:::note Required Configuration
1. **Consensus Timeouts** in `config.yml`:
   ```yaml
   consensus:
     timeout_commit: "30s"
     timeout_propose: "30s"
   ```

2. **Service Configuration**: The "static" service must be configured:
   - Service ID: `static`
   - Compute units per relay: 1 (minimal)
   - Backend: nginx-chainid server
:::

### 📋 Test Parameters

All RPS tests support flexible parameter configuration:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `R` | Requests per second | 512-100k |
| `t` | Worker threads | 16 |
| `c` | Concurrent connections | 256-5000 |
| `d` | Test duration | 30s-300s |

**Parameter Selection Guidelines:**
- **Light load**: R=100-1000, c=50-500, d=30s-60s
- **Medium load**: R=1000-5000, c=500-2000, d=60s-300s
- **Heavy load**: R=5000+, c=2000+, d=300s+

## 📊 Monitoring Results

### 🖥️ Test Output

RPS tests provide detailed performance metrics and diagnostics.

:::info Key Metrics to Monitor
- **Requests/sec**: Actual vs. target RPS
- **Latency percentiles**: 50th, 90th, 99th, 99.9th percentiles
- **Error rates**: Connection errors, timeouts, validation failures
- **Resource usage**: CPU, memory, connection pool utilization
:::

**Sample Output:**
```
Running 30s test @ http://nginx-chainid/
  16 threads and 5000 connections
  Thread calibration: mean lat.: 2.847ms, rate sampling interval: 10ms
  Thread Stats   Avg      StdDev     99%   +/- StdDev
    Latency     2.85ms    1.45ms   8.12ms   89.34%
    Req/Sec     6.31k     0.97k    8.44k    69.23%
  2999847 requests in 30.00s, 629.47MB read
Requests/sec:  99994.90
Transfer/sec:     20.98MB
```

### 📈 Grafana Dashboards

Real-time performance monitoring through specialized dashboards.

:::tip Performance Dashboards
Access comprehensive metrics at [http://localhost:3003](http://localhost:3003):

**[Relay Processing Timings](http://localhost:3003/d/a88221a7-c72c-43ed-be08-b1ccf04e21e0)** - Instruction-level latency analysis
:::

**Key Dashboard Panels:**
- **Instructions Duration AVG**: Average time per processing step
- **Instructions Duration 99p**: 99th percentile latency per instruction

### 🔍 Performance Analysis

Use the instruction-level metrics to identify bottlenecks:

**Common Bottlenecks:**
1. **`http_client_do`**: Backend service performance or network latency
2. **`pre_request_verification`**: Signature verification overhead
3. **`serialize_http_response`**: Large response processing
4. **`new_relay_request`**: Request parsing and validation

**Analysis Steps:**
1. **Compare against baseline**: nginx-chainid performance establishes upper bound
2. **Identify slow instructions**: Check 99th percentile timings
3. **Monitor error rates**: High error rates indicate resource exhaustion
4. **Resource correlation**: Match performance drops with CPU/memory spikes

## ⚙️ Configuration

### 🚨 Session Validity Warning

:::danger Session Expiration Risk
RelayRequest data is generated for the **current session**. Sessions change periodically (typically every few blocks), causing ALL requests to fail validation after session expiration.

**Critical Recommendations:**
- Set block time to 30s in `config.yml` to slow session changes
- Start tests at the beginning of a new session
- Keep test duration shorter than session length
- Monitor for session changes during longer tests
- If RelayMiner starts rejecting requests, stop and regenerate
:::

## 🎯 Available Commands

Comprehensive command reference for RelayMiner RPS testing.

| Command | Purpose | Default RPS | Duration |
|---------|---------|-------------|----------|
| `make test_baseline_static_server_load` | **Baseline** nginx performance test | 100,000 | 30s |
| `make test_relayminer_only_load` | **RelayMiner** performance test with real requests | 512 | 300s |

**Parameter Examples:**
```bash
# Quick baseline check
make test_baseline_static_server_load R=10000 D=10s

# Stress test RelayMiner
make test_relayminer_only_load R=2000 c=1000 d=60s

# Maximum infrastructure test
make test_baseline_static_server_load R=200000 T=32 C=15000
```