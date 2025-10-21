---
sidebar_position: 1
title: Service FAQ
---

### What Service Queries are available?

```bash
pocketd query service --help
```

### What Service Transactions are available?

```bash
pocketd tx service --help
```

### How do I query for all existing onchain Services?

To query for all services on Beta TestNet, you would run:

```bash
pocketd query service all-services --network=beta --output json | jq
```

And expect a response of the following format:

```json
{
  "service": [
    {
      "id": "svc_8ymf38",
      "name": "name for svc_8ymf38",
      "compute_units_per_relay": "7",
      "owner_address": "pokt1aqsr8ejvwwnjwx3ppp234l586kl06cvas7ag6w"
    },
    {
      "id": "svc_drce83",
      "name": "name for svc_drce83",
      "compute_units_per_relay": "7",
      "owner_address": "pokt1mgtf9k4k3pze57gwp3qsne88jmvqkc37t7vd9g"
    },
    {
      "id": "svc_jk07qh",
      "name": "name for svc_jk07qh",
      "compute_units_per_relay": "7",
      "owner_address": "pokt1mwynfsnzesc38f98zrk08pttjn48tu7crc2p09"
    }
  ],
  "pagination": {
    "total": "3"
  }
}
```

## I'm running `out of gas` when adding a service with a large metadata file

If you're adding a service with a very large metadata file like so:

```bash
pocketd tx service add-service pocket "Pocket Network" 1000 --from pokt1lf0kekv9zcv9v3wy4v6jx2wh7v4665s8e0sl9s --keyring-backend test --network=beta -y --gas auto --gas-prices 1upokt --gas-adjustment 1.5 --experimental-metadata-file=docs/static/openapi.json  --home=~/.pocket_prod
```

You might hit a gas limit error like this:

```bash
Error: out of gas in location: txSize; gasWanted: 200000, gasUsed: 1910452: out
```

You need to replace `--fees=100upokt` with `--gas auto --gas-prices 1upokt --gas-adjustment 1.5` to work around `'out of gas in location: txSize; gasWanted: 200000, gasUsed: 1910452: out`.
