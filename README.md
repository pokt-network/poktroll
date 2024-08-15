---
sidebar_position: 1
title: Poktroll
id: home-doc
slug: /
---

<div align="center">
  <a href="https://www.pokt.network">
    <img src="https://user-images.githubusercontent.com/2219004/151564884-212c0e40-3bfa-412e-a341-edb54b5f1498.jpeg" alt="Pocket Network logo" width="340"/>
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
  <a href="https://golang.org"><img  src="https://img.shields.io/badge/golang-v1.20-green.svg"/></a>
  <a href="https://github.com/tools/godep" ><img src="https://img.shields.io/badge/godep-dependency-71a3d9.svg"/></a>
</div>

# poktroll <!-- omit in toc -->

**poktroll** is built using the [Cosmos SDK](https://docs.cosmos.network) and
[CometBFT](https://cometbft.com/), created with [Ignite CLI](https://ignite.com/cli)
for the Shannon upgrade of the [Pocket Network](https://pokt.network) blockchain.

- [Learn about Pocket Network](#learn-about-pocket-network)
- [Developer Documentation](#developer-documentation)
- [Roadmap](#roadmap)
- [Quickstart](#quickstart)
- [Godoc](#godoc)
- [Have questions? Ask An PNYC](#have-questions-ask-an-pnyc)
- [License](#license)

## Learn about Pocket Network

User friendly documentation of the Shannon upgrade is still a WIP, but there are
a handful of (potentially outdated) resources you can reference in the meantime
to build a better understanding of Pocket Network:

- [Pocket Network official documentation](https://docs.pokt.network)
- [[Live] Pocket Network Morse; aka v0](https://github.com/pokt-network/pocket-core)
- [[Outdated] Pocket Network Protocol](https://github.com/pokt-network/pocket-network-protocol)
- [[Deprecated]Pocket Network V1](https://github.com/pokt-network/pocket)

## Developer Documentation

The developer documentation is available at [dev.poktroll.com](https://dev.poktroll.com).

## Roadmap

You can view the Shannon Roadmap on [Github](https://github.com/orgs/pokt-network/projects/144?query=is%3Aopen+sort%3Aupdated-desc)

## Quickstart

The best way to get involved is by following the [quickstart instructions](https://dev.poktroll.com/develop/developer_guide/quickstart).

## Godoc

The Godoc for the source code in this can be found at [pkg.go.dev/github.com/pokt-network/poktroll](https://pkg.go.dev/github.com/pokt-network/poktroll).

## Have questions? Ask An PNYC

You can use [PNYX](https://pnyxai.com/), an AI-powered search engine that has been
trained and indexed on the Pocket Network documentation, community calls, forums
and much more!

---

## License

This project is licensed under the MIT License; see the [LICENSE](https://github.com/pokt-network/poktroll/blob/main/LICENSE) file for details.

| Module | Field Type | Field Name | Comment |
| ------ | ---------- | ---------- | ------- |

| proof | bytes | relay_difficulty_target_hash | Params defines the parameters for the module. TODO_FOLLOWUP(@olshansk, #690): Either delete this or change it to be named "minimum" relay_difficulty_target_hash is the maximum value a relay hash must be less than to be volume/reward applicable. |
| proof | float | proof_request_probability | proof_request_probability is the probability of a session requiring a proof if it's cost (i.e. compute unit consumption) is below the ProofRequirementThreshold. |
| proof | uint64 | proof_requirement_threshold | proof_requirement_threshold is the session cost (i.e. compute unit consumption) threshold which asserts that a session MUST have a corresponding proof when its cost is equal to or above the threshold. This is in contrast to the this requirement being determined probabilistically via ProofRequestProbability. TODO_MAINNET: Consider renaming this to `proof_requirement_threshold_compute_units`. |
| shared | uint64 | num_blocks_per_session | Params defines the parameters for the module. num_blocks_per_session is the number of blocks between the session start & end heights. |
| shared | uint64 | grace_period_end_offset_blocks | grace_period_end_offset_blocks is the number of blocks, after the session end height, during which the supplier can still service payable relays. Suppliers will need to recreate a claim for the previous session (if already created) to get paid for the additional relays. |
| shared | uint64 | claim_window_open_offset_blocks | claim_window_open_offset_blocks is the number of blocks after the session grace period height, at which the claim window opens. |
| shared | uint64 | claim_window_close_offset_blocks | claim_window_close_offset_blocks is the number of blocks after the claim window open height, at which the claim window closes. |
| shared | uint64 | proof_window_open_offset_blocks | proof_window_open_offset_blocks is the number of blocks after the claim window close height, at which the proof window opens. |
| shared | uint64 | proof_window_close_offset_blocks | proof_window_close_offset_blocks is the number of blocks after the proof window open height, at which the proof window closes. |
| shared | uint64 | supplier_unbonding_period_sessions | supplier_unbonding_period_sessions is the number of sessions that a supplier must wait after unstaking before their staked assets are moved to their account balance. On-chain business logic requires, and ensures, that the corresponding block count of the unbonding period will exceed the end of any active claim & proof lifecycles. |
| application | uint64 | max_delegated_gateways | Params defines the parameters for the module. |
| service | uint64 | add_service_fee | Params defines the parameters for the module. The amount of uPOKT required to add a new service. This will be deducted from the signer's account balance, and transferred to the pocket network foundation. |
| tokenomics | uint64 | compute_units_to_tokens_multiplier | Params defines the parameters for the tokenomics module. The amount of upokt that a compute unit should translate to when settling a session. |
