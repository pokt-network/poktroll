---
title: Send tokens
sidebar_position: 5
---

## Sending Tokens Between Accounts <!-- omit in toc -->

This guide covers the process of sending tokens from one account to another on
Pocket Network using the `poktrolld` command-line interface (CLI).

- [Pre-requisites](#pre-requisites)
- [Step 1: Preparing a Node Endpoint](#step-1-preparing-a-node-endpoint)
- [Step 2: Sending Tokens](#step-2-sending-tokens)
- [Step 3: Confirming the Transaction](#step-3-confirming-the-transaction)
- [Step 4: Transaction Completion](#step-4-transaction-completion)
- [Additional Flags](#additional-flags)

## Pre-requisites

1. `poktrolld` is installed on your system; see the [installation guide](./poktrolld_cli.md) for more details
2. You have access to your wallet with sufficient tokens for the transaction and fees
3. You have the recipient's address

## Step 1: Preparing a Node Endpoint

Before initiating the transaction, you must specify the node endpoint you'll be interacting with.

For testing purposes, you can use the provided TestNet node:

```bash
--node=https://testnet-validated-validator-rpc.poktroll.com/
```

On MainNet, ensure you're connected to a trusted full node, validator, or other client on the network.

## Step 2: Sending Tokens

To send tokens, you'll use the `poktrolld tx bank send` command followed by the
sender's address or key name, the recipient's address, and the amount to send.

```sh
poktrolld tx bank send [from_key_or_address] [to_address] [amount] \
    --node=<node_endpoint> [additional_flags]
```

- Replace `[from_key_or_address]` with your wallet name or address
- Replace `[to_address]` with the recipient's address
- Replace `[amount]` with the amount you wish to send, including the denomination (e.g., 1000upokt)
- Replace `<node_endpoint>` with the node endpoint URL

For example, the following command sends `1000upokt` from `myWallet` to `pokt1recipientaddress420`:

```bash
poktrolld tx bank send myWallet pokt1recipientaddress420 1000upokt \
    --node=https://testnet-validated-validator-rpc.poktroll.com/
```

## Step 3: Confirming the Transaction

After executing the send command, you'll receive a prompt to confirm the transaction details.
Review the information carefully. If everything looks correct, proceed by confirming the transaction.

:::caution Check Recipient

Double-check the recipient's address and the amount being sent.
Transactions on the blockchain are irreversible.

:::

## Step 4: Transaction Completion

Once confirmed, the transaction will be broadcast to the network.
You'll receive a transaction hash which can be used to track the status of the transaction on a blockchain explorer.

**Congratulations!** You've successfully sent tokens on the poktrolld blockchain.

:::tip

For automated scripts or applications, you can use the `--yes` flag to skip the confirmation prompt.

:::

## Additional Flags

Refer to the command's help output for additional flags and options that can customize
your transaction. For example, you can set custom gas prices, use a specific account number,
or operate in offline mode for signing transactions.

```sh
poktrolld tx bank send --help
```
