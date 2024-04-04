---
title: Send tokens
---

# Sending Tokens

This guide covers the process of sending tokens from one account to another on the poktrolld blockchain using the `poktrolld` command-line interface. Whether you're sending tokens to a friend or moving funds between your accounts, following these steps will ensure a smooth transaction.

:::note

Ensure your `poktrolld` client is correctly configured and that you have sufficient funds in your account for the transaction and fees.

:::

## Pre-requisites

- poktrolld installed on your system.
- Access to your wallet with sufficient tokens for the transaction and fees.
- The recipient's address.

## Step 1: Setting Up the Node Endpoint

Before initiating the transaction, you must specify the node endpoint you'll be interacting with. For testing purposes, you can use the provided testnet node, but in a live mainnet, ensure you're connected to a trusted full node, validator, or a client on the network.

```sh
--node=https://testnet-validated-validator-rpc.poktroll.com/
```

Replace the URL with the endpoint of your choice if you're not using the provided testnet node.

## Step 2: Sending Tokens

To send tokens, you'll use the poktrolld tx bank send command followed by the sender's address or key name, the recipient's address, and the amount to send.


```sh
poktrolld tx bank send [from_key_or_address] [to_address] [amount] --node=<node_endpoint> [additional_flags]
```

* Replace `[from_key_or_address]` with your wallet name or address.
* Replace `[to_address]` with the recipient's address.
* Replace `[amount]` with the amount you wish to send, including the denomination (e.g., 1000upokt).
* Replace `<node_endpoint>` with the node endpoint URL.

### Example:

```sh

poktrolld tx bank send myWallet pokt1recipientaddress123 1000upokt --node=https://testnet-validated-validator-rpc.poktroll.com/

```

This command sends `1000upokt` from `myWallet` to `pokt1recipientaddress123`.


## Step 3: Confirming the Transaction

After executing the send command, you'll receive a prompt to confirm the transaction details. Review the information carefully. If everything looks correct, proceed by confirming the transaction.

:::caution

Double-check the recipient's address and the amount being sent. Transactions on the blockchain are irreversible.

:::

## Step 4: Transaction Completion

Once confirmed, the transaction will be broadcast to the network. You'll receive a transaction hash which can be used to track the status of the transaction on a blockchain explorer.

Congratulations! You've successfully sent tokens on the poktrolld blockchain.

:::tip

For automated scripts or applications, you can use the --yes flag to skip the confirmation prompt.

:::


## Additional Flags

Refer to the command's help output for additional flags and options that can customize your transaction, such as setting custom gas prices, using a specific account number, or operating in offline mode for signing transactions.

```sh
poktrolld tx bank send --help
```