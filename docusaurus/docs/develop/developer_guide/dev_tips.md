---
sidebar_position: 3
title: Developer Tips
---

# Developer Tips <!-- omit in toc -->

:::note
This is a living document and will be updated as the ecosystem matures & grows.

If you have a tip you'd like to share with others, please open a PR to add it here!
:::

- [`itest` - Investigating Flaky Tests](#itest---investigating-flaky-tests)
  - [`itest` Usage](#itest-usage)
  - [`itest` Example](#itest-example)

## `itest` - Investigating Flaky Tests

We developed a tool called `itest` to help with debugging flaky tests. It runs
the same test iteratively

### `itest` Usage

Run the following command to see the usage of `itest`:

```bash
./tools/scripts/itest.sh
```

### `itest` Example

The following is an example of `itest` in action to run the `TxClient_SignAndBroadcast_Succeeds`
test in the `pkg/client/tx` 50 times in total (5 consecutive tests over 10 runs).

```bash
make itest 5 10 ./pkg/client/tx/... -- -run TxClient_SignAndBroadcast_Succeeds
```
