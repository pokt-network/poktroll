---
title: Send tokens
sidebar_position: 6
---

## Sending Tokens Between Accounts <!-- omit in toc -->

This guide covers the process of sending tokens from one account to another on
Pocket Network using the `pocketd` command-line interface (CLI).

- [Prerequisites](#prerequisites)
- [Step 1: Preparing a Node Endpoint](#step-1-preparing-a-node-endpoint)
- [Step 2: Sending Tokens](#step-2-sending-tokens)
- [Step 3: Confirming the Transaction](#step-3-confirming-the-transaction)
- [Step 4: Transaction Completion](#step-4-transaction-completion)
- [Additional Flags](#additional-flags)

## Prerequisites

1. `pocketd` is installed on your system; see the [installation guide](1_pocketd_cli.md) for more details
2. You have access to your wallet with sufficient tokens for the transaction and fees
3. You have the recipient's address

## Step 1: Preparing a Node Endpoint

Before initiating the transaction, you must specify the node endpoint you'll be interacting with.

For testing purposes, you can use the provided Beta TestNet node:

```bash
--network=beta
```

On MainNet, ensure you're connected to a trusted full node, validator, or other client on the network.

## Step 2: Sending Tokens

To send tokens, you'll use the `pocketd tx bank send` command followed by the
sender's address or key name, the recipient's address, and the amount to send.

```sh
pocketd tx bank send [from_key_or_address] [to_address] [amount] \
    --network=<network> [additional_flags]
```

- Replace `[from_key_or_address]` with your wallet name or address
- Replace `[to_address]` with the recipient's address
- Replace `[amount]` with the amount you wish to send, including the denomination (e.g., 1000upokt)
- Replace `<network>` with the network name (e.g., `local`, `alpha`, `beta`, `main`)

For example, the following command sends `1000upokt` from `myWallet` to `pokt1recipientaddress420`:

```bash
pocketd tx bank send myWallet pokt1recipientaddress420 1000upokt \
    --network=<network>
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

**Congratulations!** You've successfully sent tokens on the pocketd blockchain.

:::tip

For automated scripts or applications, you can use the `--yes` flag to skip the confirmation prompt.

:::

## Additional Flags

Refer to the command's help output for additional flags and options that can customize
your transaction. For example, you can set custom gas prices, use a specific account number,
or operate in offline mode for signing transactions.

```sh
pocketd tx bank send --help
```
