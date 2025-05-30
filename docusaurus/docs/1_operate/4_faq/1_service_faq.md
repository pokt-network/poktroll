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

### How do I create or update a service?

Use the `setup-service` command which supports both creating new services and updating existing ones:

```bash
pocketd tx service setup-service --help
```

**Command format:**
```bash
pocketd tx service setup-service <service_id> <service_name> [compute_units_per_relay] [service_owner] --from <signer>
```

:::tip Key points

- For new services: The signer pays the service fee and can specify a different owner
- For updates: Only the current service owner can update the service (no additional fee)
- Service ownership can be transferred during updates

:::

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
