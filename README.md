# poktroll <!-- omit in toc -->

**poktroll** is a rollup built using [Rollkit](https://rollkit.dev/), [Cosmos SDK](https://docs.cosmos.network) and [CometBFT](https://cometbft.com/), created with [Ignite CLI](https://ignite.com/cli) for the Shannon upgrade of the [Pocket Network](https://pokt.network) blockchain.

- [Where are the docs?](#where-are-the-docs)
  - [Roadmap](#roadmap)
  - [Godoc](#godoc)
  - [Pocket V1 (Shannon) Docs](#pocket-v1-shannon-docs)
- [Getting Started](#getting-started)
  - [Makefile](#makefile)
  - [Development](#development)
  - [LocalNet](#localnet)

## Where are the docs?

_This repository is still young & early. We're focusing on development right now._

### Roadmap

You can find our Roadmap Changelog [here](https://github.com/pokt-network/poktroll/blob/main/docs/roadmap_changelog.md).

### Godoc

The godocs for this repository can be found at [pkg.go.dev/github.com/pokt-network/poktroll](https://pkg.go.dev/github.com/pokt-network/poktroll).

### Pocket V1 (Shannon) Docs

It is the result of a research spike conducted by the Core [Pocket Network](https://pokt.network/) Protocol Team at [GROVE](https://grove.city/) documented [here](https://www.pokt.network/why-pokt-network-is-rolling-with-rollkit-a-technical-deep-dive/) (deep dive) and [here](https://www.pokt.network/blog/pokt-network-rolling-into-the-modular-future-of-the-protocol-a-technical-deep-dive) (summary).

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
