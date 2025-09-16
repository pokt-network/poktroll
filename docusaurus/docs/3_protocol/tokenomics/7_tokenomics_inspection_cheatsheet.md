---
title: Tokenomics Inspection Cheatsheet
sidebar_position: 7
---

- [Bash RC helpers](#bash-rc-helpers)
- [`pocketd` CLI queries](#pocketd-cli-queries)
- [Available Onchain Events](#available-onchain-events)
- [Viewing Account Balance over time](#viewing-account-balance-over-time)
- [Inspecting Claim settlement in a specific block](#inspecting-claim-settlement-in-a-specific-block)

## Bash RC helpers

There are a handful of useful bash helpers in the `tools/rc_helpers` directory.

```bash
source ./tools/rc_helpers/queries.sh
```

```text
Available commands:
  shannon_query_unique_tx_msgs_and_events  - Get unique message and event types
  shannon_query_unique_block_events        - Get unique block events
  ...
```

## `pocketd` CLI queries

To inspect various tokenomics params, you can query the `tokenomics` module

```bash
pocketd q tokenomics --help
```

To inspect onchain claims & proofs, you can query the `proofs` module

```bash
pocketd q proofs --help
```

## Available Onchain Events

You can find all available events by running

```bash
grep -r "message Event" ./proto/pocket/tokenomics
```

And filtering for the events you're interested in inspecting either onchain `txs`:

```bash
pocketd q txs --help
```

Or onchain `blocks`:

```bash
pocketd q block-results --help
```

## Viewing Account Balance over time

The following is an example of how to view the balance of `pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q` grow from height `205950` to `210000` in 30 height increments.

```bash
for ((h=205950; h<=210000; h+=30)); do echo -n "Height $h: "; curl -s -H "x-cosmos-block-height: $h" https://shannon-grove-api.mainnet.poktroll.com/cosmos/bank/v1beta1/balances/pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q | jq -r '.balances[0].amount // "0"'; done
```

## Inspecting Claim settlement in a specific block

If some block was used for claim settlemend (e.g. `210033`), you can download it like so:

```bash
pocketd query block-results 210033 --network=main -o json >> block_210033.json
```

And identify all the events related to token transfers associated with a particular account (e.g. `pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q`)

```bash
cat block_210033.json | jq -r '.finalize_block_events[]
  | select(.type == "transfer")
  | select(.attributes[]?
      | select(.key == "recipient" and .value == "pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q"))
  | .attributes[]
  | select(.key == "amount")
  | .value'
```
