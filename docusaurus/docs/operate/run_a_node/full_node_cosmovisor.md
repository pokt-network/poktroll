---
title: Full Node - Cosmovisor
sidebar_position: 1
---

<<<<<<< HEAD
## Table of Contents  <!-- omit in toc -->
=======
# Run a Full Node using Cosmovisor <!-- omit in toc -->

>>>>>>> origin/main
- [What is a Full Node](#what-is-a-full-node)
- [What is Cosmovisor](#what-is-cosmovisor)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Installation Steps](#installation-steps)
- [Post-Installation](#post-installation)

## What is a Full Node

In blockchain networks, a full node retains continuous synchs and updates the latest copy of the ledger. It may either be a pruned full node (the latest data only) or an archival full node (including complete and historical data).

You can visit the [Cosmos SDK documentation](https://docs.cosmos.network/main/user/run-node/run-node) for more information on Full Nodes.

## What is Cosmovisor

[Cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor) is a tool that automates the version management for our blockchain. It allows operators to automatically upgrade their full nodes and validators without downtime and reduce maintenance overhead.

## Installation

To install and set up a Poktroll Full Node using Cosmovisor, we provide a comprehensive installation script. This script will handle all the necessary steps, including user creation, dependency installation, Cosmovisor and Poktrolld setup, and system configuration.

### Prerequisites

- A Linux-based system (Debian-based distributions are fully supported, others may work as well)
- Root or sudo access
- A dedicated server or a virtual machine (any provider should work, Vultr and Hetzner have been tested)

### Installation Steps

<<<<<<< HEAD
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
   - Desired username to run poktrolld (default: poktroll)
   - Node moniker (default: hostname)
   - Seeds (default: fetched [from the official source](https://github.com/pokt-network/pocket-network-genesis/tree/master/poktrolld))
   - Chain ID (default: poktroll-testnet)

The script will then proceed with the installation and setup process.

## Post-Installation

After the installation is complete, your Poktroll Full Node should be up and running. Here are some useful commands for managing your node:

1. Check the status of your node:

```bash
sudo systemctl status cosmovisor.service
```

2. View the logs:

```bash
sudo journalctl -u cosmovisor.service -f
```

3. Stop the node:

```bash
sudo systemctl stop cosmovisor.service
```

4. Start the node:

```bash
sudo systemctl start cosmovisor.service
```

5. Restart the node:

```bash
sudo systemctl restart cosmovisor.service
```

Remember to keep your system updated and monitor your node regularly to ensure its proper functioning and security.
=======
[Content to be added]
>>>>>>> origin/main
