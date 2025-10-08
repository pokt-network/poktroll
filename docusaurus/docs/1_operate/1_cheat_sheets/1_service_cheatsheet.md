---
sidebar_position: 1
title: Service Creation (~ 5 min)
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
- [How do I create a new service?](#how-do-i-create-a-new-service)
  - [1. Add a Service](#1-add-a-service)
  - [2. Query for the Service](#2-query-for-the-service)
  - [3. What do I do next?](#3-what-do-i-do-next)
- [How do I update an existing service's `compute_units_per_relay`?](#how-do-i-update-an-existing-services-compute_units_per_relay)

## Introduction

This page will walk you through creating an onchain Service.

To learn more about what a Service is, or how it works, see the [Protocol Documentation](../../protocol/).

## Prerequisites

1. Install the [pocketd CLI](../../2_explore/2_account_management/1_pocketd_cli.md).
2. [Create and fund a new account](../../2_explore/2_account_management/2_create_new_account_cli.md) before you begin.

## How do I create a new service?

:::info Service Limitations

Service IDs are limited to `42` chars and descriptions are limited to `169` chars.

:::

### 1. Add a Service

:::danger Grove Employees Service Creation

If you are a Grove Employee, you **ABSOLUTELY MUST** create all Mainnet Services using the Grove Master Gateway: `pokt1lf0kekv9zcv9v3wy4v6jx2wh7v4665s8e0sl9s` 

:::

Use the `add-service` command to create a new service like so:

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} \
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta \
    [--experimental--metadata-file ./openapi.json | --experimental--metadata-base64 $(base64 ./openapi.json)]

Attach an API specification to the service by providing **either** `--experimental--metadata-file` (raw bytes) **or**
`--experimental--metadata-base64` (pre-encoded). These flags are mutually exclusive and the payload must decode to 100 KiB or less.
```

For example, assuming you have an account with the name $USER (`pocketd keys show $USER -a`), you can run the following for Beta TestNet:

```bash
pocketd tx service add-service \
   "svc-$USER" "service description for $USER" 13 \
    --fees 300upokt --from $USER \
   --network=beta \
   --experimental--metadata-file ./svc-$USER-openapi.json
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

## How do I update an existing service's `compute_units_per_relay`?

Use the `add-service` command to modify the `compute_units_per_relay` for an existing service.

Provide the `SERVICE_ID` of the `Service` you want to update, but with a new value for `COMPUTE_UNITS_PER_RELAY`.

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${NEW_COMPUTE_UNITS_PER_RELAY} \
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta \
    [--experimental--metadata-file ./openapi.json | --experimental--metadata-base64 $(base64 ./openapi.json)]
```

For example:

```bash
pocketd tx service add-service \
   "svc-$USER" "service description for $USER" 20 \
    --fees 300upokt --from $USER \
   --network=beta \
   --experimental--metadata-base64 $(base64 -w0 ./svc-$USER-openapi.json)
```
