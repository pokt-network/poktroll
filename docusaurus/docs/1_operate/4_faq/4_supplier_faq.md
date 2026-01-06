---
sidebar_position: 4
title: Supplier FAQ
---

### What Supplier operations are available?

```bash
pocketd tx supplier -h
```

### What Supplier queries are available?

```bash
pocketd query supplier -h
```

### How do I query for all existing onchain Suppliers?

Then, you can query for all services like so:

```bash
pocketd query supplier list-suppliers --network=beta --output json | jq
```

See [Non-Custodial Staking](https://dev.poktroll.com/operate/configs/supplier_staking_config#non-custodial-staking) for more information about supplier owner vs operator and non-custodial staking.

### Secure vs Non-Secure `query_node_grpc_url`

In `/tmp/relayminer_config.yaml`, you'll see that we specify an endpoint
for `query_node_grpc_url` which is TLS terminated.

If `grpc-insecure=true` then it **MUST** be an HTTP port, no TLS. Once you have
an endpoint exposed, it can be validated like so:

```bash
grpcurl -plaintext <host>:<port> list
```

If `grpc-insecure=false`, then it **MUST** be an HTTPS port, with TLS.

The Grove team exposed one such endpoint on one of our validators for Beta Testnet
at `https://sauron-grpc.beta.infra.pocket.network/:443`.

It can be validated with:

```bash
grpcurl https://sauron-grpc.beta.infra.pocket.network:443 list
```

Note that no `-plaintext` flag is required when an endpoint is TLS terminated and
must be omitted if it is not.

:::tip

You can replace both `http` and `https` with `tcp` and it should work the same way.

:::

## What is the different between a RelayMiner & Supplier

## What happens if you go below the Min stake?

## What is the maximum stake?
