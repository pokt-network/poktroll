---
sidebar_position: 1
title: Load Testing
---

# Load Testing <!-- omit in toc -->

Poktroll load-testing suite.

- [Overview](#overview)
- [Dependencies](#dependencies)
- [How to run tests](#how-to-run-tests)
  - [Common k6 CLI flags](#common-k6-cli-flags)
- [Understanding Output](#understanding-output)
  - [CLI Output](#cli-output)
  - [HTML Report](#html-report)
  - [Adding a new load test](#adding-a-new-load-test)
  - [Sophisticated Reporting](#sophisticated-reporting)
- [Troubleshooting](#troubleshooting)
- [File structure](#file-structure)

## Overview

We use [k6](https://k6.io/) for load testing. For detailed information about k6 internals and best practices, please refer to the [k6 documentation](https://grafana.com/docs/k6/latest/).

## Dependencies

- [k6](https://grafana.com/docs/k6/latest/get-started/installation/)
- (For local suite execution) [LocalNet](../infrastructure/localnet.md)

## How to run tests

Tests are stored in the [load-testing/tests](https://github.com/pokt-network/poktroll/tree/main/load-testing/tests) folder, covering various use cases.

For instance, here's a basic load test sending requests through `AppGateServer`, which proxies them to `RelayMiner`:

```bash
k6 run load-testing/tests/appGateServerEtherium.js
```

### Common k6 CLI flags

The `k6` load testing tool provides various command-line flags to customize and control your load tests. Below are some of the key flags that you can use:

- `--vus`: Specifies the number of virtual users (VUs) to simulate. This flag allows you to set the concurrency level of your load test.

  - Example: `k6 run script.js --vus=50` runs the test with 50 virtual users.

- `--duration`: Defines the duration for which the test should run. You can specify the time in seconds (s), minutes (m), or hours (h).

  - Example: `k6 run script.js --duration=1m` runs the test for 1 minute.

- `--http-debug`: Enables HTTP debugging. This flag can be set to `full` for detailed logging of all HTTP requests and responses. Useful for troubleshooting and debugging your tests.
  - Use `--http-debug` for a summary of HTTP requests.
  - Use `--http-debug=full` for detailed request and response logging.
  - Example: `k6 run script.js --http-debug=full` provides a detailed log of HTTP transactions.

The default configurations for VUs and duration are in the [config file](https://github.com/pokt-network/poktroll/tree/main/load-testing/config/index.js). Override these defaults by passing the appropriate flags to the `k6` command. For example:

```bash
k6 run load-testing/tests/appGateServerEtherium.js --vus=300 --duration=30s
```

For a comprehensive list of available flags and their usage, refer to the [k6 documentation](https://grafana.com/docs/k6/latest/).

## Understanding Output

### CLI Output

Post-test, basic performance metrics are available:

```
  scenarios: (100.00%) 1 scenario, 100 max VUs, 1m30s max duration (incl. graceful stop):
           * default: 100 looping VUs for 1m0s (gracefulStop: 30s)

INFO[0061] [k6-reporter v2.3.0] Generating HTML summary report  source=console
     ✓ is status 200
     ✓ is successful JSON-RPC response

     checks.........................: 100.00% ✓ 12000     ✗ 0
     data_received..................: 1.6 MB  27 kB/s
     data_sent......................: 1.2 MB  19 kB/s
     http_req_blocked...............: avg=52.52µs min=0s     med=2µs    max=7.08ms  p(90)=4µs     p(95)=6µs
     http_req_connecting............: avg=37.73µs min=0s     med=0s     max=5.48ms  p(90)=0s      p(95)=0s
     http_req_duration..............: avg=8.51ms  min=1.58ms med=5.98ms max=81.98ms p(90)=15.45ms p(95)=19.97ms
       { expected_response:true }...: avg=8.51ms  min=1.58ms med=5.98ms max=81.98ms p(90)=15.45ms p(95)=19.97ms
     http_req_failed................: 0.00%   ✓ 0         ✗ 6000
     http_req_receiving.............: avg=42.18µs min=5µs    med=25µs   max=5.71ms  p(90)=65µs    p(95)=125µs
     http_req_sending...............: avg=26.5µs  min=2µs    med=9µs    max=7.2ms   p(90)=21µs    p(95)=42µs
     http_req_tls_handshaking.......: avg=0s      min=0s     med=0s     max=0s      p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=8.44ms  min=1.55ms med=5.93ms max=81.28ms p(90)=15.38ms p(95)=19.82ms
     http_reqs......................: 6000    98.915624/s
     iteration_duration.............: avg=1s      min=1s     med=1s     max=1.08s   p(90)=1.01s   p(95)=1.02s
     iterations.....................: 6000    98.915624/s
     vus............................: 100     min=100     max=100

running (1m00.7s), 000/100 VUs, 6000 complete and 0 interrupted iterations
default ✓ [======================================] 100 VUs  1m0s
```

### HTML Report

An HTML report is generated in the execution directory. Open it in your default browser:

```bash
open summary.html
```

### Adding a new load test

TODO_DOCUMENT(@okdas): Add link to PR next time a new type of load test is added.

### Sophisticated Reporting

We're developing advanced reporting that integrates additional tags set in the code. This will require time-series databases and is planned for DevNets.

## Troubleshooting

To debug, activate the logging feature:

`--http-debug` or `--http-debug=full`

## File structure

```
load-testing
├── README.md
├── config
│   └── index.js
├── modules                              # reusable code for scenarios and tests
│   └── etheriumRequests.js
├── scenarios                            # different scenarios for tests
│   └── requestBlockNumberEtherium.js
└── tests                                # test scripts
    ├── anvilDirectEtherium.js
    └── appGateServerEtherium.js
```
