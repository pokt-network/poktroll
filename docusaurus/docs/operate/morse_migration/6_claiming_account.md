---
title: Claiming Morse Accounts
sidebar_position: 6
---

### Claim a Basic Morse Account (Unstaked, Non-Actor)

:::note

If the Morse account is not staked as an Application or Supplier, it is considered an "unstaked" or "non-actor" account.

:::

Claiming an unstaked account will mint the unstaked balance of the Morse account being claimed to the Shannon account which the signer of the `MsgClaimMorseAccount`.

This unstaked balance amount is retrieved from the corresponding onchain `MorseClaimableAccount` which was imported by the foundation.

For example, running the following command:

```bash
pocketd migrate claim-account \
  ./pocket-account-8b257c7f4e884e49bafc540d874f33f91436e1dc.json \
  --from app1
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
