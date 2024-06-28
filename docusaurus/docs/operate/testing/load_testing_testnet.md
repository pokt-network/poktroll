---
sidebar_position: 2
title: Load Testing - TestNet
---

# Load Testing on TestNets <!-- omit in toc -->

A guide on how to perform load testing on TestNets.

- [Overview](#overview)
- [Load Testing Steps](#load-testing-steps)
- [Examples](#examples)

## Overview

Load Testing on TestNets is very similar to testing on DevNets. The main difference is that the test might run against software you don't control. Gateways and Suppliers can be hosted by other teams or organizations, so you need to ensure that your tests do not adversely affect their operations. As another side effect, you won't be able to collect metrics and logs on the software you don't run.

## Load Testing Steps

Please refer to the generic [load testing documentation](./load_testing.md#non-ephemeral-networks-testnets-mainnet-etc) for non-ephemeral networks (TestNets, MainNet, etc.). The steps are very similar to those for DevNets, which are thoroughly documented on the dedicated [Load Testing on DevNets](./load_testing_devnet.md) page.

Key points to consider:
1. Identify the TestNet endpoints and actors (gateways, suppliers) you'll be testing against.
2. Ensure you have the necessary permissions and have communicated your intent to run load tests.
3. Configure your load testing manifest to target the TestNet resources.
4. Start with lower load and gradually increase to avoid overwhelming the network.
5. Monitor the TestNet's performance metrics during the test.

## Examples

:::info
TODO_DOCUMENT(@okdas): add a few examples on how to perform load testing on TestNets:
- How to run the test against gateways and suppliers that are not under your control.
- How to run the test against gateways and suppliers deployed in a separate service for isolation.
:::