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
- [Experimental: How do I attach a card to a service?](#experimental-how-do-i-attach-a-card-to-a-service)
  - [Validate Before You Broadcast](#validate-before-you-broadcast)
  - [Using a File](#using-a-file)
  - [Using Base64-Encoded Data](#using-base64-encoded-data)
  - [Updating a Service Card](#updating-a-service-card)
  - [Reading a Card Back](#reading-a-card-back)
  - [Important Notes](#important-notes)
- [Experimental: How do I attach a card to a gateway?](#experimental-how-do-i-attach-a-card-to-a-gateway)

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
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta
```

For example, assuming you have an account with the name $USER (`pocketd keys show $USER -a`), you can run the following for Beta TestNet:

```bash
pocketd tx service add-service \
   "svc-$USER" "service description for $USER" 13 \
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

#### Query without metadata (dehydrated)

If you want to query a service without its metadata (API specifications) to reduce payload size:

```bash
pocketd query service show-service ${SERVICE_ID} --dehydrated
```

For example:

```bash
pocketd query service show-service "svc-$USER" \
 --dehydrated --network=beta --output json | jq
```

This is useful when you only need basic service information (ID, name, compute units, owner) without the full API specification.

#### Query all services

To list all services:

```bash
pocketd query service all-services
```

By default, this excludes metadata to reduce payload size. To include metadata for all services:

```bash
pocketd query service all-services --dehydrated=false
```

### 3. What do I do next?

_TODO(@olshansk): Coming soon..._

## How do I update an existing service's `compute_units_per_relay`?

Use the `add-service` command to modify the `compute_units_per_relay` for an existing service.

Provide the `SERVICE_ID` of the `Service` you want to update, but with a new value for `COMPUTE_UNITS_PER_RELAY`.

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${NEW_COMPUTE_UNITS_PER_RELAY} \
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta
```

For example:

```bash
pocketd tx service add-service \
   "svc-$USER" "service description for $USER" 20 \
    --fees 300upokt --from $USER \
   --network=beta
```

## Experimental: How do I attach a card to a service?

:::warning Experimental Feature

The onchain metadata feature is experimental and subject to change.
The payload is limited to 256 KiB when decoded as of [#1825](https://github.com/pokt-network/poktroll/pull/1825).

:::

A **service card** is a small, self-describing JSON document attached to a service, stored onchain
and readable by applications, gateways, and suppliers. It describes what the service is, which
transports it expects, and what a node runner needs to serve it — and it points at a full API
specification ([OpenAPI](https://www.openapis.org/), [OpenRPC](https://open-rpc.org/), etc.) via
its `specs[]` field rather than inlining one.

**Full reference:** [`docs/pocket_cards.md`](https://github.com/pokt-network/poktroll/blob/main/docs/pocket_cards.md)
covers both service cards and gateway cards. The canonical schema is
`pkg/cards/service_card.schema.json`.

### Validate Before You Broadcast

The chain never parses the card, so client-side validation is the only place a malformed one is
caught before it costs gas:

```bash
pocketd tx service validate-card ./card.json
```

`add-service` and `edit-service` run the same check automatically. Pass `--skip-card-validation`
to publish a payload that is deliberately not a card.

### Using a File

To attach a card from a file:

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} \
    --card-file ./card.json \
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta
```

For example, to create a service for the Pocket network with its card:

```bash
pocketd tx service add-service \
   "pocket" "Pocket Network RPC" 1 \
    --card-file ./card.json \
    --fees 300upokt --from $USER \
   --network=beta
```

### Using Base64-Encoded Data

Alternatively, you can provide the card as base64-encoded data:

```bash
pocketd tx service add-service \
    ${SERVICE_ID} "${SERVICE_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} \
    --card-base64 $(base64 -w0 ./card.json) \
    --fees 300upokt --from ${SERVICE_OWNER} --network=beta
```

### Updating a Service Card

To update the card of an existing service, use the same `add-service` command with the new card:

```bash
pocketd tx service add-service \
   "pocket" "Pocket Network RPC" 1 \
    --card-file ./card-v2.json \
    --fees 300upokt --from $USER \
   --network=beta
```

To update several services at once, use `edit-service` with a YAML config instead. All updates go
out as a single batched transaction by default:

```bash
pocketd tx service edit-service --config ./services.yaml --from $USER --fees 300upokt
```

```yaml
# services.yaml
services:
  - service_id: svc1
    compute_units_per_relay: 15
  - service_id: svc2
    compute_units_per_relay: 25
    card_file: ./cards/svc2.json
```

An entry with no `card_file` leaves that service's stored card untouched. Card comparison is
byte-exact, so reformatting a card file counts as a change even when the JSON is identical
semantically.

### Reading a Card Back

There is no dedicated decoding command for services, so decode the base64 payload yourself:

```bash
pocketd query service show-service ${SERVICE_ID} -o json \
  | jq -r '.service.metadata.card' | base64 -d
```

### Important Notes

- The `--card-file` and `--card-base64` flags are mutually exclusive.
- The decoded payload must be 256 KiB or less. Target 4 KiB — point at full API specs with the
  card's `specs[]` field instead of inlining them.
- The card is stored onchain as raw bytes and base64-encoded in JSON representations.
- Only the service owner can update the service card.
- Updating replaces the entire previous card (not a partial update).
- **Omitting both card flags preserves the stored card — it does not remove it.** As of v0.1.35, a
  message with no card means "leave the stored card alone", so an unrelated
  `compute_units_per_relay` update can no longer wipe it. There is currently no way to clear a card
  back to nil; publishing a minimal payload such as `{}` is the closest available action.

## Experimental: How do I attach a card to a gateway?

Gateways carry a card too, using the same container, the same 256 KiB cap, and the same rules —
only the field set differs, since a gateway is a reachable endpoint rather than an API class. See
[`docs/pocket_cards.md`](https://github.com/pokt-network/poktroll/blob/main/docs/pocket_cards.md#gateway-cards)
for the fields and a worked example.

```bash
# Validate first
pocketd tx gateway validate-card ./gateway-card.json

# Publish. --from IS the gateway being updated, so there is no address argument.
pocketd tx gateway update-gateway-metadata \
    --card-file ./gateway-card.json \
    --fees 300upokt --from ${GATEWAY} --network=beta
```

Unlike `add-service`, one of `--card-file` / `--card-base64` is **required** here — there is no
no-op form of this command.

Cards are never set via `stake-gateway`. That is deliberate: `stake-gateway` requires escrowing
additional POKT on every call, so folding the card into it would mean paying stake to fix a typo.
Updating a card costs gas only. The gateway must already be staked, but may be unbonding.

Read a gateway card back with the dedicated decoding query:

```bash
pocketd query gateway card ${GATEWAY_ADDRESS}

# Exact stored bytes, no re-indenting — for hashing, diffing, or feeding
# straight back into --card-file
pocketd query gateway card ${GATEWAY_ADDRESS} --raw > gateway-card.json
```
