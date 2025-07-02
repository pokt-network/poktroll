---
title: Community Tooling
sidebar_position: 1
---

A collection of community-maintained tools and resources for Pocket Network. All links and instructions are copy-pasta ready.

## Table of Contents <!-- omit in toc -->

- [pocketd: Easy Full Node Deployment (with Snapshot)](#pocketd-easy-full-node-deployment-with-snapshot)
- [shannon-keyring-loader: Bulk Key Import Tool](#shannon-keyring-loader-bulk-key-import-tool)
- [Helm Chart for RelayMiner \& FullNode](#helm-chart-for-relayminer--fullnode)

---

## [pocketd: Easy Full Node Deployment (with Snapshot)](https://github.com/stakenodes-unchained/pocketd)

Useful for anyone spinning up a full node quickly

**Features**:

- Easy full-node deployment
- Snapshot download included

[GitHub Repo](https://github.com/stakenodes-unchained/pocketd)

---

## [shannon-keyring-loader: Bulk Key Import Tool](https://github.com/pokt-shannon/shannon-keyring-loader)

Bulk import Shannon wallet keys into your keyring from a JSON file.

**Features**:

- Loads Shannon wallet keys into your keyring from a JSON file
- Securely adds keys to your configured keyring
- Avoids manual, one-by-one key entry
- Optionally adds named keys to the RelayMiner `config.yaml`

[Full instructions & details](https://github.com/pokt-shannon/shannon-keyring-loader/blob/main/README.md)

---

## [Helm Chart for RelayMiner & FullNode](https://github.com/eddyzags/pocket-network-helm-chart)

Streamlines deployment on Kubernetes clusters

**Features**:

- Deploys both `RelayMiner` and `FullNode`
- Sensible default values for both
- Custom configuration provisioning
- Key provisioning via Kubernetes Secret
- Persistent storage for `FullNode` (local or PVC)
- Prometheus integration with `ServiceMonitor`
- Ingress resource definitions for services/endpoints
- Resources & limits presets (`small`, `medium`, `large`)
- Input validators for values
- Delve development environment for `relayminer`
- Liveness probes for both
- HPA resource definition for relayminer (autoscaling)

[GitHub Repo](https://github.com/eddyzags/pocket-network-helm-chart)

## [Pocket Knife](https://github.com/buildwithgrove/pocket-knife/)

A python syntactic sugar wrapper for `pocketd` — streamlined for bulk operations.

**Features**:

- Quickly find and export all node operator addresses owned by a wallet, with smart filtering, deduplication, and file output.
- Analyze total holdings across liquid balances, app stakes, and node stakes from a single structured JSON input.
- Batch unstake hundreds of operators with automatic gas estimation, transaction tracking, and error reporting.
- and many more features!

[GitHub Repo](https://github.com/buildwithgrove/pocket-knife)
