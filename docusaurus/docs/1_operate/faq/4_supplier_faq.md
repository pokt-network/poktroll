---
sidebar_position: 4
title: Supplier FAQ
---

Get diagrams from here:

https://medium.com/decentralized-infrastructure/exploring-paths-capabilities-with-pocket-network-s-shannon-upgrade-and-deepseek-part-one-2a3ee6032dfb

https://medium.com/decentralized-infrastructure/exploring-paths-capabilities-with-pocket-network-s-shannon-upgrade-and-deepseek-part-two-a5e38766fd71

## What is the different between a RelayMiner & Supplier

TODO: Add onchain / offchain diagram

## What happens if you go below the Min stake?

## What is the maximum stake?

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
pocketd query supplier list-suppliers --node https://shannon-testnet-grove-rpc.beta.poktroll.com --output json | jq
```

See [Non-Custodial Staking](https://dev.poktroll.com/operate/configs/supplier_staking_config#non-custodial-staking) for more information about supplier owner vs operator and non-custodial staking.
