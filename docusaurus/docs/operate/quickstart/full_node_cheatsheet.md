---
sidebar_position: 3
title: Full Node Cheat Sheet
---

This cheat sheet provides quick instructions for installing a Full Node using an automated script.

- [Pre-Requisites](#pre-requisites)
- [Install a Full Node using Cosmovisor](#install-a-full-node-using-cosmovisor)
- [What Gets Installed](#what-gets-installed)
- [Useful Commands](#useful-commands)
  - [Check the status of your node](#check-the-status-of-your-node)
  - [View the logs](#view-the-logs)
  - [Stop the node](#stop-the-node)
  - [Start the node](#start-the-node)
  - [Restart the node](#restart-the-node)
  - [Advanced Operations](#advanced-operations)
- [Automatic Upgrades](#automatic-upgrades)

### Pre-Requisites

1. **Linux-based System**: Ensure you have a Debian-based Linux distribution (other distributions may work but are not fully supported).
2. **Root or Sudo Access**: You need administrative privileges to run the installation script.
3. **Dedicated Server or Virtual Machine**: Any provider should work (Vultr and Hetzner have been tested).

### Install a Full Node using Cosmovisor

To install and set up a Full Node, follow these steps:

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
   - **Set Node Moniker**: Input the node moniker (default: your hostname).
   - **Confirm Seeds and Genesis File**: The script fetches seeds and the genesis file automatically.
   - **External IP Address**: The script detects your external IP address. Confirm or input manually if incorrect.

The script will handle the installation of dependencies, user creation, environment variable setup, and configuration of Cosmovisor and `poktrolld`.

### What Gets Installed

When you run the installation script, the following components are set up:

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

### Useful Commands

After installation, you can manage your node using the following commands:

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

#### Advanced Operations

Check the current version:
```bash
sudo -u poktroll poktrolld version
```

View the Cosmovisor directory structure:
```bash
ls -la /home/poktroll/.poktroll/cosmovisor/
```

Check if an upgrade is available:
```bash
ls -la /home/poktroll/.poktroll/cosmovisor/upgrades/
```

View node configuration:
```bash
cat /home/poktroll/.poktroll/config/config.toml
```

### Automatic Upgrades

Your node is configured to handle chain upgrades automatically through Cosmovisor. When a chain upgrade is proposed and approved:

1. Cosmovisor will download the new binary
2. The node will stop at the designated upgrade height
3. Cosmovisor will switch to the new binary
4. The node will restart automatically

No manual intervention is required for standard upgrades.

<!-- 
## Becoming a Validator

TODO(@okdas, #754): Add instructions for becoming a validator.

-->
