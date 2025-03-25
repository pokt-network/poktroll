---
sidebar_position: 1
title: PoktrolPocketl
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

:::note Pocket Network Project Documentation

This is the living technical documentation for the protocol design, implementation,
and operation. If you're looking for general documentation related to Pocket Network,
please visit [docs.pokt.network](https://docs.pokt.network).

:::

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
  <a href="https://golang.org"><img  src="https://img.shields.io/badge/golang-v1.23-green.svg"/></a>
  <a href="https://github.com/tools/godep" ><img src="https://img.shields.io/badge/godep-dependency-71a3d9.svg"/></a>
</div>

## Pocket Network Shannon Technical Docs (aka pocket) <!-- omit in toc -->

**pocket** is the source code and core implementation of the [Shannon upgrade](https://docs.pokt.network/pokt-protocol/the-shannon-upgrade) for [Pocket Network](https://pokt.network/).

`pocket` is built using the [Cosmos SDK](https://docs.cosmos.network), [CometBFT](https://cometbft.com/) and [Ignite CLI](https://ignite.com/cli).

## What is Pocket Network? <!-- omit in toc -->

:::note 🚧 Under Construction 🚧

This documentation is not intended to answer this question as of 02/2025

Consider reading [this post from 02/2025](https://medium.com/decentralized-infrastructure/an-update-from-grove-on-shannon-beta-testnet-path-the-past-the-future-5bf7ec2a9acf) by @olshansk
to get some understanding of why you need Pocket & Grove.

:::

---

## Table of Contents <!-- omit in toc -->

- [Where do I start?](#where-do-i-start)
- [Shannon Roadmap](#shannon-roadmap)
- [PATH for Gateways](#path-for-gateways)
- [GoDoc Documentation](#godoc-documentation)
- [License](#license)

## Where do I start?

1. [Guides & Deployment](./operate/cheat_sheets/full_node_cheatsheet.md): Deployment cheat sheets and config overviews for node runners, infrastructure operators and CLI users.
2. [Tools & Explorers](./tools/user_guide/pocketd_cli.md): Explorers, wallets, faucets and other resources to interact with the network.
3. [Core Developers](./develop/developer_guide/walkthrough.md): Guides & walkthroughs for core or external developers looking to contribute to the core protocol or SDK.
4. [Protocol Design](./protocol/actors/actors.md): Learn more about tokenomics design & protocol architecture.

:::note 🚧 Under Construction 🚧

As of 02/2025, this documentation is under construction and does not have a clear
user journey. Different parts are intended to serve as references one can link to
or jump to/from when needed.

:::

## Shannon Roadmap

The Shannon Roadmap, along with all past, active and future work is tracked via [this Github project](https://github.com/orgs/pokt-network/projects/144).

## PATH for Gateways

[Grove](https://grove.city/) is developing [PATH](https://path.grove.city/) for
anyone who aims to deploy a Pocket Network gateway. Visit the docs to get started.

The PATH Roadmap, along with all past, active and future work is tracked via [this Github project](https://github.com/orgs/buildwithgrove/projects/1).

## GoDoc Documentation

The Godoc for the source code can be found at [pkg.go.dev/github.com/pokt-network/pocket](https://pkg.go.dev/github.com/pokt-network/pocket).

---

## License

This project is licensed under the MIT License; see the [LICENSE](https://github.com/pokt-network/poktroll/blob/main/LICENSE) file for details.
