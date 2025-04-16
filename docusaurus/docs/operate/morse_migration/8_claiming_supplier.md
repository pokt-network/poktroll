---
title: Claiming Morse Suppliers
sidebar_position: 8
---

#### Critical Analogy: Morse Servicer to Shannon Supplier

_tl;dr Claiming a Morse Servicer as a Shannon Supplier is like staking a new Shannon Supplier without explicitly specifying the `stake_amount`_

Claiming a Morse supplier requires a supplier config that is identical to that used by the Shannon CLI when staking a supplier (as opposed to claiming). See the [supplier staking config](../configs/supplier_staking_config.md).

The only difference between staking a new supplier on Shannon and **claiming** an existing supplier from Morse to Shannon is that the `stake_amount` is determined by the corresponding onchain `MorseClaimableAccount`'s supplier stake amount.

:::important Omit `stake_amount`

Omit the `stake_amount` field in a supplier config; providing it in when claiming a Morse supplier is an error.

:::

:::important (optional) Non-custodial staking

If you would like to claim a Morse supplier with distinct `owner` and `operator` addresses,
you MAY do so by specifying both in the [supplier staking config](../configs/supplier_staking_config.md).

This delegates signing claims and proofs to the `operator` address, while the `owner` address retains sole ownership over the supplier stake and rewards.

See the ["Supplier staking config" > "Staking types"](../configs/supplier_staking_config.md#staking-types).

:::

#### Instructions to Claim A Morse Servicer as a Shannon Supplier

Claiming a Morse supplier account will:

1. **Mint** the unstaked balance of the Morse account being claimed to the Shannon account; the signer of the `MsgClaimMorseAccount` is the Shannon "destination" account.
2. **Stake** the corresponding Shannon "destination" account as a supplier (on Shannon) with the given services configuration and same stake amount as the Morse application being claimed had (on Morse).

Both the unstaked balance and supplier stake amounts are retrieved from the corresponding onchain `MorseClaimableAccount` imported by the foundation.

For example, running the following command:

```bash
pocketd tx migration claim-supplier \
Enter Decrypt Passphrase:
MsgClaimMorseSupplier {
  "shannon_owner_address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
  "shannon_operator_address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
  "morse_src_address": "44892C8AB52396BA016ADDD0221783E3BD29A400",
  "morse_signature": "rYyN2mnjyMMrYdDhuw+Hrs98b/svn38ixdSWye3Gr66aAJ9CQhdiaYB8Lta8tiwWIteIM8rmWYUh0QkNdpkNDQ==",
  "services": [
    {
      "service_id": "anvil",
      "endpoints": [
        {
          "url": "http://relayminer1:8545",
          "rpc_type": 3
        }
      ],
      "rev_share": [
        {
          "address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
          "rev_share_percentage": 100
        }
      ]
    }
  ]
}

Confirm MsgClaimMorseSupplier: y/[n]: y

```

Should prompt for a passphrase and produce output similar to the following:

```shell
Enter Decrypt Passphrase:
MsgClaimMorseAccount {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "8B257C7F4E884E49BAFC540D874F33F91436E1DC",
  "morse_signature": "hLGhLRjP6jgP6wgOIaYFxIxT3z4jb4IBDKovMkX5AqUsOqdF+rEIO5aofOKnmYW9BkqL0v2DfUfE3nj25FNhBA=="
}
Confirm MsgClaimMorseAccount: y/[n]:
```

:::tip

See: `pocketd tx migrate claim-supplier --help` for more details.

:::
