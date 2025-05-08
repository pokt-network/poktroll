---
sidebar_position: 1
title: Service Creation (~ 5 min)
---

- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
- [Service Help](#service-help)
- [How do I create a new service?](#how-do-i-create-a-new-service)
  - [1. Add a Service](#1-add-a-service)
  - [2. Query for the Service](#2-query-for-the-service)

## Introduction

This page will walk you through creating an onchain Service.

To learn more about what a Service is, or how it works, see the [Protocol Documentation](../../4_protocol/protocol.md).

<!--

TODO(@olshansky): Link to a dedicated page to learn and understand how services work.

Explain things like:
- Why create and/or maintain a service?
- What are the earnings & rewards?
- How to tracking the service usage & earnings onchain?
- The process of maintaining the service APIs
- Limitations on service id or description
-->

## Prerequisites

1. Install the [pocketd CLI](../../2_explore/user_guide/pocketd_cli.md).
2. [Create and fund a new account](../../2_explore/user_guide/create-new-wallet.md) before you begin.

## Service Help

**Service queries**:

**Service queries**:

```bash
pocketd query service --help
```

**Service transactions**:

```bash
pocketd tx service --help
```

Visit the [Service FAQ](../../faq/1_service_faq.md) for more information about interacting with Services.

## How do I create a new service?

:::info Service Limitations

Service IDs are limited to `42` chars and descriptions are limited to `169` chars.

:::

### 1. Add a Service

Use the `add-service` command to create a new service like so:

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} \
    --fees 300upokt --from ${SERVICE_OWNER}
```

For example, assuming you have an account with the name $USER (`pocketd keys show $USER -a`), you can run the following for Beta TestNet:

```bash
pocketd tx service add-service \
   "svc-$USER" "service description for $USER" 13 \
   --fees 300upokt --from $USER \
   --chain-id pocket-beta --node https://shannon-testnet-grove-rpc.beta.poktroll.com
```

### 2. Query for the Service

Query for your service on the next block:

```bash
pocketd query service show-service ${SERVICE_ID}
```

For example:

```bash
pocketd query service show-service "svc-$USER" \
 --node https://shannon-testnet-grove-rpc.beta.poktroll.com --output json | jq
```
