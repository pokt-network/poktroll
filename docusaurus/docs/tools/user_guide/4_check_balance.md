---
title: Balance check
sidebar_position: 4
---

# Checking Your Wallet Account Balance <!-- omit in toc -->

:::note Usage requirements

You will need access to your wallet address and the denomination of the token you
wish to query (i.e. `upokt`).

The default node is set to interact with a local instance. For network-specific
queries (i.e. accessing TestNet or MainNet), you will need an RPC endpoint.

:::

Knowing your account's balance is crucial for effective transaction management
on Pocket Network. This guide provides the necessary steps to check your wallet's
balance using the `pocketd` command-line interface (CLI).

- [Pre-requisites](#pre-requisites)
- [Step 1: Preparing the Query](#step-1-preparing-the-query)
- [Step 2: Viewing the Balance](#step-2-viewing-the-balance)
- [Accessing non-local environments](#accessing-non-local-environments)

## Pre-requisites

1. `pocketd` is installed on your system; see the [installation guide](./1_pocketd_cli.md) for more details
2. You have the address of the wallet you wish to check
3. You know the token denomination you wish to check; `upokt` for POKT tokens

:::info What is a upokt?

1 POKT = 1,000,000 upokt

1 upokt = 1 micro POKT = 10^-6 POKT = 0.000001 POKT

:::

## Step 1: Preparing the Query

You can check your wallet's balance by specifying the `address` in the following command:

```sh
pocketd query bank balance [address] upokt
```

Example:

```sh
pocketd query bank balance pokt1hdfggsqdy66awgvr4lclyupddz4n2dfrl9rjwv upokt
```

## Step 2: Viewing the Balance

Upon executing the command, you'll receive output similar to the following, showing your balance:

```plaintext
balance:
  amount: "8999"
  denom: upokt
```

This output indicates that the wallet address holds 8999 `upokt` tokens.

## Accessing non-local environments

You must provide the `--node` flag to access non LocalNet environments.

For example, to check a balance on TestNet, you would use the following command:

```sh
pocketd query bank balance [address] upokt \
  --node=https://testnet-validated-validator-rpc.poktroll.com
```
