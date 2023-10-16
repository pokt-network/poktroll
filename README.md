# poktroll <!-- omit in toc -->

**poktroll** is a rollup built using [Rollkit](https://rollkit.dev/), [Cosmos SDK](https://docs.cosmos.network) and [CometBFT](https://cometbft.com/), created with [Ignite CLI](https://ignite.com/cli) for the Shannon upgrade of the [Pocket Network](https://pokt.network) blockchain.

- [Getting Started](#getting-started)
  - [Makefile](#makefile)
  - [Development](#development)
  - [LocalNet](#localnet)

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
