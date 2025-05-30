---
sidebar_position: 1
title: Service Management (~ 5 min)
---

<!-- TODO(@olshansky):

- Add details about maintaining a service
- Add details about deleting a service
- Add details about updating the service API
- Add details about updating the service description
- Add details about updating the service compute units per relay -->

:::tip Services FAQ

Visit the [Service FAQ](../4_faq/1_service_faq.md) for more information about interacting with Services.

:::

## Table of Contents <!-- omit in toc -->

- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
- [How do I create or update a service?](#how-do-i-create-or-update-a-service)
  - [1. Setup a Service](#1-setup-a-service)
  - [2. Query for the Service](#2-query-for-the-service)
  - [3. What do I do next?](#3-what-do-i-do-next)

## Introduction

This page will walk you through creating and updating onchain Services.

To learn more about what a Service is, or how it works, see the [Protocol Documentation](../../protocol/).

## Prerequisites

1. Install the [pocketd CLI](../../2_explore/2_account_management/1_pocketd_cli.md).
2. [Create and fund a new account](../../2_explore/2_account_management/2_create_new_account_cli.md) before you begin.

## How do I create or update a service?

:::info Service Limitations

Service IDs are limited to `42` chars and descriptions are limited to `169` chars.

:::

### 1. Setup a Service

Use the `setup-service` command to create a new service or update an existing one like so:

```bash
pocketd tx service setup-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} ${OWNER_ADDRESS} \
    --fees 300upokt --from ${SIGNER} --network=beta
```

:::tip Service Creation vs Update

- **Creating a new service**: Requires payment of the service fee from the signer's account
- **Updating an existing service**: No additional fee required, but only the current service owner can update it
- **Ownership transfer**: The service owner can transfer ownership to another address during an update

:::

For example, assuming you have an account with the name $USER (`pocketd keys show $USER -a`), you can run the following for Beta TestNet:

**Create a new service:**
```bash
pocketd tx service setup-service \
   "svc-$USER" "service description for $USER" 13 \
   --fees 300upokt --from $USER \
   --network=beta
```

**Update an existing service:**
```bash
pocketd tx service setup-service \
   "svc-$USER" "updated service description" 20 \
   --fees 300upokt --from $USER \
   --network=beta
```

**Create a service for another owner:**
```bash
# Create a service where the signer ($USER) pays the fee but a different address owns the service
pocketd tx service setup-service \
   "svc-$USER" "service description" 13 $(TARGET_OWNER_ADDRESS) \
   --fees 300upokt --from $USER \
   --network=beta
```

### 2. Query for the Service

Query for your service on the next block:

```bash
pocketd query service show-service ${SERVICE_ID}
```

For example:

```bash
pocketd query service show-service "svc-$USER" \
 --network=beta --output json | jq
```

### 3. What do I do next?

_TODO(@olshansk): Coming soon..._
