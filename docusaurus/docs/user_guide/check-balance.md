---
title: Check balance
---

# Checking Your Wallet Balance

Knowing your account's balance is crucial for effective transaction management on the poktrolld blockchain. This guide provides the necessary steps to check your wallet's balance using the `poktrolld` command-line interface.

:::note

You'll need access to your wallet address and the denomination of the token you wish to query. The default node is set to interact with a local instance. For network-specific queries, ensure you specify the correct node endpoint.

:::

## Pre-requisites

- poktrolld installed on your system.
- The address of the wallet you wish to check.
- Token denomination - `upokt` for POKT tokens.

## Step 1: Preparing the Query

You can check your wallet's balance by specifying the address and the token denomination. 

```sh
poktrolld query bank balance [address] upokt
```

### Example:

```sh
poktrolld query bank balance pokt1hdfggsqdy66awgvr4lclyupddz4n2dfrl9rjwv upokt
```

This command queries the balance of upokt tokens for the specified address.

## Step 2: Viewing the Balance

Upon executing the command, you'll receive output similar to the following, showing your balance:

```plaintext
balance:
  amount: "8999"
  denom: upokt
```

This output indicates that the wallet address holds 8999 upokt tokens.

:::tip

For network-specific balance queries or when accessing a remote node, use the --node flag to specify the node endpoint. For example, for a testnet node, you could use --node=https://testnet-validated-validator-rpc.poktroll.com/. This flag is crucial for accurate and up-to-date balance information.

:::