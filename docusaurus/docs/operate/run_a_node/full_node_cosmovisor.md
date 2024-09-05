---
title: Full Node - Cosmovisor
sidebar_position: 2
---

## Run a Full Node using Cosmovisor <!-- omit in toc -->

This document provides instructions on using the official Cosmos SDK [Cosmosvisor](https://docs.cosmos.network/v0.45/run-node/cosmovisor.html) to run a full Pocket Network node.

- [What is a Full Node](#what-is-a-full-node)
- [What is Cosmovisor](#what-is-cosmovisor)
- [Installation Instructions](#installation-instructions)
  - [Prerequisites](#prerequisites)
  - [Installation Steps](#installation-steps)
- [Useful Command Cheat Sheet](#useful-command-cheat-sheet)
  - [Check the status of your node](#check-the-status-of-your-node)
  - [View the logs](#view-the-logs)
  - [Stop the node](#stop-the-node)
  - [Start the node](#start-the-node)
  - [Restart the node](#restart-the-node)

### What is a Full Node

In blockchain networks, a full node retains continuous synchs and updates the latest copy of the ledger. It may either be a pruned full node (the latest data only) or an archival full node (including complete and historical data).

You can visit the [Cosmos SDK documentation](https://docs.cosmos.network/main/user/run-node/run-node) for more information on Full Nodes.

### What is Cosmovisor

[Cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor) is a tool that automates the version management for our blockchain. It allows operators to automatically upgrade their full nodes and validators without downtime and reduce maintenance overhead.

### Installation Instructions

To install and set up a Poktroll Full Node using Cosmovisor, we provide a comprehensive installation script. This script will handle all the necessary steps, including user creation, dependency installation, Cosmovisor and Poktrolld setup, and system configuration.

#### Prerequisites

- A Linux-based system (Debian-based distributions are fully supported, others may work as well)
- Root or sudo access
- A dedicated server or a virtual machine (any provider should work, Vultr and Hetzner have been tested)

#### Installation Steps

1. Download the installation script:

   ```bash
   curl -O https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/installer/full-node.sh
   ```

2. Make the script executable:

   ```bash
   chmod +x full-node.sh
   ```

3. Run the script with sudo privileges:

   ```bash
   sudo ./full-node.sh
   ```

4. Follow the prompts to provide the necessary information:
   - Desired username to run poktrolld (`default: poktroll`)
   - Node moniker (`default: hostname`)
   - Seeds (`default: fetched` [from the official source](https://github.com/pokt-network/pocket-network-genesis/tree/master/poktrolld))
   - Chain ID (`default: poktroll-testnet`)

The script will then proceed with the installation and setup process.

### Useful Command Cheat Sheet

After the installation is complete, your Poktroll Full Node should be up and running.

:::tip
Remember to keep your system updated and monitor your node regularly to ensure its proper functioning and security.
:::

Here are some useful commands for managing your node:

#### Check the status of your node

```bash
 sudo systemctl status cosmovisor.service
```

#### View the logs

```bash
sudo journalctl -u cosmovisor.service -f
```

#### Stop the node

```bash
sudo systemctl stop cosmovisor.service
```

#### Start the node

```bash
sudo systemctl start cosmovisor.service
```

#### Restart the node

```bash
sudo systemctl restart cosmovisor.service
```
