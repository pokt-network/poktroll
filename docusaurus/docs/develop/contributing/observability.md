---
sidebar_position: 2
title: Observability guidelines
---

:::warning
We are still refining our observability guidelines. If in doubt - please reach out on `#protocol-public` channel on
[Grove Discord](https://discord.gg/build-with-grove).
:::

- [Metrics](#metrics)
  - [Overview](#overview)
  - [Types of Metrics](#types-of-metrics)
  - [High Cardinality Considerations](#high-cardinality-considerations)
  - [Best Practices](#best-practices)
  - [Examples](#examples)
  - [Counter](#counter)
    - [x/proof/keeper/msg_server_create_claim.go:](#xproofkeepermsg_server_create_claimgo)
  - [Gauage](#gauage)
    - [x/tokenomics/module/abci.go:](#xtokenomicsmoduleabcigo)
  - [Histogram](#histogram)
- [Logs](#logs)

## Metrics

### Overview

In our system, metrics are exposed using the Prometheus exporter. This approach aligns with tools like Rollkit, and we
leverage the [go-kit metrics package](https://pkg.go.dev/github.com/go-kit/kit/metrics) for custom metrics
implementation. For practical examples of metric definitions, refer to
[AppGate Metrics](https://github.com/pokt-network/poktroll/blob/main/pkg/appgateserver/metrics.go).

### Types of Metrics

1. **Counter:** A cumulative metric that represents a single numerical value that only ever goes up. Ideal for counting
   requests, tasks completed, errors, etc.

2. **Gauge:** Represents a single numerical value that can arbitrarily go up and down. Suitable for measuring values
   like memory usage, number of active goroutines, etc.

3. **Histogram:** Captures a distribution of values. It divides the range of possible values into buckets and counts how
   many values fall into each bucket. Useful for tracking request durations, response sizes, etc.

### High Cardinality Considerations

Developers should be cautious about the high cardinality of labels. High cardinality labels can significantly increase
the memory usage and reduce the performance of the Prometheus server. To mitigate this:

- Limit the use of labels that have a large number of potential values (e.g., user IDs, email addresses).
- Prefer using labels with low cardinality (e.g., status codes, environment names).
- Regularly review and clean up unused or less useful metrics.

### Best Practices

- **Clarity and Relevance:** Ensure that each metric provides clear and relevant information for observability.
- **Documentation:** Document each custom metric, including its purpose and any labels used.
- **Consistency:** Follow the Prometheus Metric and Label Naming Guide for consistent naming and labeling. See more at [Prometheus Naming Guide](https://prometheus.io/docs/practices/naming/).
- **Defer:** When the code being metered includes conditional branches, defer calls to metrics methods to ensure that any referenced variables are in their final state prior to reporting.
- **Sufficient Variable Scope:** Ensure any variables which are passed to metrics methods are declared in a scope which is sufficient for reference by such calls.
  - Ensure that these variables **are not shadowed** by usage of a subsequent walrus operator `:=` (redeclaration) within the same scope.
  - The above might requrie declaring previously undeclared variables which are part of a multiple return.

### Examples

### Counter

#### [x/proof/keeper/msg_server_create_claim.go](https://github.com/pokt-network/poktroll/blob/main/x/proof/keeper/msg_server_create_claim.go):

```go
// Declare a named `error` return argument.
func (k msgServer) CreateClaim(...) (_ *types.MsgCreateClaimResponse, err error) {
	// Declare claim to reference in telemetry.
	var (
		claim           types.Claim
		isExistingClaim bool
		numRelays       uint64
		numComputeUnits uint64
	)

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingClaim {
			telemetry.ClaimCounter(types.ClaimProofStage_CLAIMED, 1, err)
			telemetry.ClaimRelaysCounter(types.ClaimProofStage_CLAIMED, numRelays, err)
			telemetry.ClaimComputeUnitsCounter(types.ClaimProofStage_CLAIMED, numComputeUnits, err)
		}
	}()


    // Ensure `err` is not shadowed by avoiding `:=` operator.
    var result any
    result, err = doSomething()
    if err != nil {
        return nil, err
    }
```

### Gauage

#### [x/tokenomics/module/abci.go](https://github.com/pokt-network/poktroll/blob/main/x/tokenomics/module/abci.go):

```go
	// Emit telemetry for each service's relay mining difficulty.
	for serviceId, newDifficulty := range difficultyPerServiceMap {
		miningDifficultyNumBits := keeper.RelayMiningTargetHashToDifficulty(newDifficulty.TargetHash)
		telemetry.RelayMiningDifficultyGauge(miningDifficultyNumBits, serviceId)
		telemetry.RelayEMAGauge(newDifficulty.NumRelaysEma, serviceId)
	}
```

### Histogram

TODO: Add a code example, link to usage, and screenshot of the output.

## Logs

Please refer to our own [polylog package](https://github.com/pokt-network/poktroll/blob/main/pkg/polylog/godoc.go#L1).
