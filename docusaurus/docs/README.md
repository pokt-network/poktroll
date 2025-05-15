---
sidebar_position: 1
title: Pocket
id: home-doc
slug: /
---

<!-- markdownlint-disable MD033 -->
<!-- markdownlint-disable MD045 -->

<div align="center">
  <a href="https://www.pokt.network">
    <img src="https://github.com/user-attachments/assets/01ddfcac-3b64-42ab-8e83-e87a5e9b36a6" alt="Pocket Network logo" width="340"/>
  </a>
</div>

<div>
  <a href="https://discord.gg/pokt"><img src="https://img.shields.io/discord/553741558869131266"/></a>
  <a  href="https://github.com/pokt-network/poktroll/releases"><img src="https://img.shields.io/github/release-pre/pokt-network/pocket.svg"/></a>
  <a  href="https://github.com/pokt-network/poktroll/pulse"><img src="https://img.shields.io/github/contributors/pokt-network/pocket.svg"/></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/pulse"><img src="https://img.shields.io/github/last-commit/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/pulls"><img src="https://img.shields.io/github/issues-pr/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/releases"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos-pink.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/issues"><img src="https://img.shields.io/github/issues/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/issues"><img src="https://img.shields.io/github/issues-closed/pokt-network/pocket.svg"/></a>
  <a href="https://godoc.org/github.com/pokt-network/pocket"><img src="https://img.shields.io/badge/godoc-reference-blue.svg"/></a>
  <a href="https://goreportcard.com/report/github.com/pokt-network/pocket"><img src="https://goreportcard.com/badge/github.com/pokt-network/pocket"/></a>
  <a href="https://golang.org"><img  src="https://img.shields.io/badge/golang-v1.24-green.svg"/></a>
  <a href="https://github.com/tools/godep" ><img src="https://img.shields.io/badge/godep-dependency-71a3d9.svg"/></a>
</div>

## The Crypto-Native API Layer

Welcome to Web3's missing Crypto-Native API layer.

Building upon the learnings of a _Decentralized RPC_ network that launched on MainNet in 2020,
this upgrade to Pocket Network is a major evolution of the protocol.

### What does all of this enable?

Leveraging the nature of distributed ledger, we enable anyone (developers, enterprises, agents) to be able
to access any public canonical data source (e.g. geospatial data, blockchains) or any open-source-service (e.g. LLMS, data feeds, etc.).

This is achieved by:

1. **Creating** a permissionless **registry** of public APIs for any open-source-service or data-source
2. **Incentivizing** anyone to become an **operator** supporting the APIs above.
3. **Using** a cryptographically verifiable **API counter** (i.e. rate limiter) to reward and penalize actors appropriately
4. **Providing** a Gateway Framework (PATH) that ensures enterprise-grade **Quality-of-Service (QoS) and Service Level Agreements (SLAs)** atop of a set of permissionless operators and penalizes bad actors.

![PATH USP](../static/img/pokt-path-usp.png)

### What is it built on top of?

Pocket network is built on top of [Cosmos SDK](https://docs.cosmos.network), [CometBFT](https://cometbft.com/), and [Ignite CLI](https://ignite.com/cli).

## Where do I get Started?

### Most Common Starting Points

- **Affected by the Migration**: Check out [this page](./category/morse---shannon-migration) to start getting acquainted with the migration process.
- **Technical User**: Visit [this page](./explore/account_management/create_new_account_cli) to install the CLI.
- **Operator**: Visit one of the following pages to deploy a [Full Node](./operate/cheat_sheets/full_node_cheatsheet), [Validator](./operate/cheat_sheets/validator_cheatsheet), [Supplier](./operate/cheat_sheets/supplier_cheatsheet), or [Gateway](./operate/cheat_sheets/gateway_cheatsheet).
- [Coming soon]: **Casual User**: Check [this page](./explore/account_management/create_new_account_wallet) to create a new wallet.

### How is this documentation organized?

- ⚙️ **[Infrastructure Operators](./operate):** Cheat sheets, guides and configs for operators, node runners and infrastructure operators.
- 🗺️ **[Users & Explorers](./explore):** Explorers, wallets, faucets, CLIs and other resources to interact with the network.
- 🧑 **[Core Developers](./develop):** Guides and onboarding docs for contributing to the core protocol or SDK.
- 🧠 **[Protocol Specifications](./protocol):** Learn more about tokenomics design and protocol architecture.

## What is PATH?

[PATH](https://path.grove.city/) (Path API & Toolkit Harness) is an open source Gateway framework that streamlines access to the permissionless API operators on Pocket Network without sacrificing enterprise-grade SLAs.

You can think of Pocket Network as the directory of API providers, and PATH is the toolkit for building Gateways that ensure high quality of service on top of Pocket using Smart QoS.

---

## Need Help?

- Join our [Discord](https://discord.gg/pokt) for real-time support and community discussion.
- Open an issue on [GitHub](https://github.com/pokt-network/poktroll/issues) if you spot a bug or need help.

<!-- TODO(@olshansky): Add other ways to reach out -->

---

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/pokt-network/poktroll/blob/main/LICENSE) file for details.
