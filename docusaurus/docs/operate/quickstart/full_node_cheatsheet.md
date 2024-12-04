---
title: Full Node Cheat Sheet
sidebar_position: 3
---

## Full Node Cheat Sheet Using Systemd & Cosmovisor <!-- omit in toc -->

This cheat sheet provides quick copy-pasta like instructions for installing and
running a Full Node using an automated scripts.

:::tip

If you're interesting in understanding everything, or having full control of every
step, check out the [Full Node Walkthrough](../run_a_node/full_node_walkthrough.md).

:::

- [Introduction](#introduction)
- [Pre-Requisites](#pre-requisites)
- [Install and Run a Full Node using Cosmovisor](#install-and-run-a-full-node-using-cosmovisor)
  - [Automatic Upgrades Out of the Box](#automatic-upgrades-out-of-the-box)
- [FAQ \& Troubleshooting](#faq--troubleshooting)
- [\[OPTIONAL\] Do you care to know what just happened?](#optional-do-you-care-to-know-what-just-happened)

### Introduction

This guide will help you install a Full Node for Pocket Network,
**using helper that abstract out some of the underlying complexity.**

Running a Full Node is the first step toward becoming a Validator, Supplier, or Gateway.

### Pre-Requisites

1. **Linux-based System**: Ensure you have a Debian-based Linux distribution (other distributions may work but are not fully supported).
2. **Root or Sudo Access**: You need administrative privileges to run the installation script.
3. **Dedicated Server or Virtual Machine**: Any provider should work (Vultr and Hetzner have been tested).

### Install and Run a Full Node using Cosmovisor

:::info
This section script will handle the installation of dependencies, user creation,
environment variable setup, and configuration of Cosmovisor and `poktrolld`.
:::

Follow the instructions below to **quickly** install and set up a Full Node:

1. **Download the Installation Script**:

   ```bash
   curl -O https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/installer/full-node.sh
   ```

2. **Run the Script with Sudo Privileges**:

   ```bash
   sudo bash full-node.sh
   ```

3. **Follow the Prompts**:

   - **Choose the Network**: Select `testnet-alpha`, `testnet-beta`, or `mainnet`.
   - **Set Username**: Input the desired username to run `poktrolld` (default: `poktroll`).
   - **Set Node Moniker**: Input the node moniker (default: your `hostname`).
   - **Confirm Seeds and Genesis File**: The script fetches seeds and the genesis file automatically.
   - **External IP Address**: The script detects your external IP address. Confirm or input manually if incorrect.

#### Automatic Upgrades Out of the Box

Your node is configured to handle chain upgrades automatically through Cosmovisor. No manual intervention is required for standard upgrades.

When a chain upgrade is proposed and approved:

1. Cosmovisor will download the new binary
2. The node will stop at the designated upgrade height
3. Cosmovisor will switch to the new binary
4. The node will restart automatically

### FAQ & Troubleshooting

See the [FAQ & Troubleshooting section in the Full Node Walkthrough](../run_a_node/full_node_walkthrough.md#faq--troubleshooting)
for examples of useful commands, common debugging instructions and other advanced usage.

### [OPTIONAL] Do you care to know what just happened?

:::info
This section is optional and for informational purposes only.
:::

If you're interest in understand what just got installed, keep reading...

1. **System User**: A dedicated user (default: `poktroll`) is created to run the node securely.

2. **Cosmovisor**: A binary manager that handles chain upgrades automatically:

   - Location: `/home/poktroll/bin/cosmovisor`
   - Purpose: Manages different versions of `poktrolld` and handles chain upgrades
   - Configuration: Set up to automatically download and switch to new binaries during upgrades

3. **Poktrolld**: The core node software:

   - Location: `/home/poktroll/.poktroll/cosmovisor/genesis/bin/poktrolld`
   - Configuration: `/home/poktroll/.poktroll/config/`
   - Data: `/home/poktroll/.poktroll/data/`

4. **Systemd Service**: A service that manages the node:
   - Name: `cosmovisor.service`
   - Status: Enabled and started automatically
   - Configured for automatic restarts and upgrades
