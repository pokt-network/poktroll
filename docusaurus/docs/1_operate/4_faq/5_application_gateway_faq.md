---
sidebar_position: 5
title: Application & Gateway FAQ
---

## Application FAQs

### What onchain operations are available for Applications?

Application Query Help:

```bash
pocketd query application --help
```

Application Transaction Help:

```bash
pocketd tx application --help
```

### How do I update the services staked on my application? 
You can reuse the staking command like so to update your services:

```bash
# Set up config file
cat <<🚀 > /tmp/stake_app_config.yaml
stake_amount: 100000000upokt
service_ids:
  - anvil
🚀

# Upstake the application
pocketd tx application stake-application --config=/tmp/stake_app_config.yaml --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS

```

NOTE: The staked amount in the config file must exceed the current staked amount.

## Gateway FAQs

### What onchain operations are available for Gateways?

Gateway Query Help:

```bash
pocketd query gateway --help
```

Gateway Transaction Help:

```bash
pocketd tx gateway --help
```
