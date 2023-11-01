# poktroll <!-- omit in toc -->

**poktroll** is a rollup built using [Rollkit](https://rollkit.dev/), [Cosmos SDK](https://docs.cosmos.network) and [CometBFT](https://cometbft.com/), created with [Ignite CLI](https://ignite.com/cli) for the Shannon upgrade of the [Pocket Network](https://pokt.network) blockchain.

- [Getting Started](#getting-started)
  - [Makefile](#makefile)
  - [Development](#development)
  - [LocalNet](#localnet)

## Where are the docs?

_This repository is still young & early._

It is the result of a research spike conducted by the Core [Pocket Network](https://pokt.network/) Protocol Team at [GROVE](https://grove.city/) documented [here](https://www.pokt.network/why-pokt-network-is-rolling-with-rollkit-a-technical-deep-dive/) (deep dive) and [here](https://www.pokt.network/a-sovereign-rollup-and-a-modular-future/) (summary).

For now, we recommend visiting the links in [pokt-network/pocket/README.md](https://github.com/pokt-network/pocket/blob/main/README.md) as a starting point.

If you want to contribute to this repository at this stage, you know where to find us.

## Getting Started

### Makefile

Run `make` to see all the available commands

### Development

```bash
# Build local files & binaries
make go_develop

# Run all the unit tests
make go_test
```

### LocalNet

Please check out the [LocalNet documentation](./localnet/README.md).
