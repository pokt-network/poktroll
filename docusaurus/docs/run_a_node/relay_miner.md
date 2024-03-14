---
title: RelayMiner - Docker Compose
---

### What is a RelayMiner?

RelayMiner is a type of node that node runners can deploy to provide service via Pocket Network. Read more in [RelayMiner documentation](../actors/relay_miner.md).

### 0. Prerequisites

Ensure the following software is installed on your system:
- [git](https://github.com/git-guides/install-git);
- [Docker](https://docs.docker.com/engine/install/);
- [docker-compose](https://docs.docker.com/compose/install/#installation-scenarios);
<!-- - TODO(@okdas): provide a link to binaries -->
- poktrolld binary;
<!-- - TODO(@okdas): what's the correct amount to stake? -->
- POKT tokens to stake your relayminer and allow it to submit transactions;

### 1. Generate new key and fund the wallet

:::info

Some **content** with _Markdown_ `syntax`. Check [this `api`](#).

In this example, we use "test" keyring to showcase how to run RelayMiner. It is not advisable to use this method in production.
Please, consult with [this documentation page](https://docs.cosmos.network/v0.50/user/run-node/keyring) to
understand what other options are available.

:::

Your relayminer will need access to the keys to submit transactions on the network.
```
KEY_NAME="relayminer1" poktrolld --keyring-backend=test --home=./$KEY_NAME keys add $KEY_NAME
```

