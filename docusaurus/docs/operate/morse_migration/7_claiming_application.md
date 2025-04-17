---
title: Claiming Morse Applications
sidebar_position: 7
---

### Claim a Morse Application (Staked, Actor)

Claiming a Morse Application account will:

1. **Mint Unstaked Balance**: The unstaked balance of the Morse Account being claimed will transfer to the unstaked balance of the Shannon account (i.e. signer of `MsgClaimMorseAccount`).
2. **Stake a new Application**: The staked balance of the Morse Application being claimed will stake the corresponding Shannon "destination" account as an Application.

:::note Same Balance, New Configurations

Note that even though the staked & unstaked balance map identically from Morse to Shannon, Shannon actors can modify the actual staking configurations (e.g. `service_id`).

:::

Recall that the unstaked balance and application stake amounts are retrieved from the corresponding onchain `MorseClaimableAccount` imported by the foundation.

For example, running the following command:

```bash
pocketd tx migration claim-application \
  ./pocket-account-8b257c7f4e884e49bafc540d874f33f91436e1dc.json \
  anvil \
  --from app1
```

Should prompt for a passphrase and produce output similar to the following:

```shell
Enter Decrypt Passphrase:
MsgClaimMorseApplication {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288",
  "morse_signature": "6kax1TKdvP1sIGrz8lW8jH/jQxv5OiPiFq0/BG5sEfLwVyVNVXihDhJNXd0cQtwDiMPB88PCkvWZOdY/WMY4Dg==",
  "service_config": {
    "service_id": "anvil"
  }
}
Confirm MsgClaimMorseApplication: y/[n]:
```

:::tip

See `pocketd tx migrationclaim-application --help` for more details.

:::
