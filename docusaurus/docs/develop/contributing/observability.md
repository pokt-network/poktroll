---
sidebar_position: 2
title: Observability guidelines
---

# Work in progress <!-- omit in toc -->

:::warning
We are still refining our observability guidelines. If in doubt - please reach out on `#protocol-public` channel on 
[Grove Discord](https://discord.gg/build-with-grove).
:::

- [Metrics](#metrics)
  - [Overview](#overview)
  - [Types of Metrics](#types-of-metrics)
  - [High Cardinality Considerations](#high-cardinality-considerations)
  - [Best Practices](#best-practices)
  - [\[TODO(@okdas)\] Examples](#todookdas-examples)
  - [Counter](#counter)
  - [Gauage](#gauage)
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

### [TODO(@okdas)] Examples

:::warning
This is a placeholder section so our team remembers to add it in the future.
:::

### Counter

TODO: Add a code example, link to usage, and screenshot of the output.

### Gauage

TODO: Add a code example, link to usage, and screenshot of the output.

### Histogram

TODO: Add a code example, link to usage, and screenshot of the output.

## Logs

Please refer to our own [polylog package](https://github.com/pokt-network/poktroll/blob/main/pkg/polylog/godoc.go#L1).