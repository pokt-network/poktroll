---
title: Full Node (~10 min)
sidebar_position: 2
---

## Full Node Cheat Sheet <!-- omit in toc -->

**🖨 🍝 instructions to get you up and running with a `Full Node` on Pocket Network using `Systemd` and `Cosmovisor` ✅**

:::warning There is lots of scripting and some details are abstracted away

See the [Full Node Walkthrough](../walkthroughs/full_node_walkthrough.md) if you want to understand what's happening under the hood.

:::

---

## Table of Contents <!-- omit in toc -->

- [Pre-Requisites \& Requirements](#pre-requisites--requirements)
- [Install and Run a Full Node using Cosmovisor](#install-and-run-a-full-node-using-cosmovisor)
  - [Verify successful installation using `curl`](#verify-successful-installation-using-curl)
  - [How are automatic upgrades handled out of the box?](#how-are-automatic-upgrades-handled-out-of-the-box)
- [Do you care to know what just happened?](#do-you-care-to-know-what-just-happened)

## Pre-Requisites & Requirements

1. **Linux-based System**: Preferably Debian-based distributions (Ubuntu, Debian).
2. **Hardware Requirements**: See the [hardware requirements doc](../configs/hardware_requirements.md)
3. **Architecture Support**: Both x86_64 (amd64) and ARM64 architectures are supported.
4. **Root or Sudo Access**: Administrative privileges are required.
5. **Dedicated Server or Virtual Machine**: Any provider is acceptable.

:::tip Vultr Playbook

If you are using [Vultr](https://www.vultr.com/) for your deployment, you can following the [CLI Playbook we put together here](../../tools/playbooks/vultr.md) to speed things up.

:::

## Install and Run a Full Node using Cosmovisor

:::info
This section's script will handle the installation of dependencies, user creation,
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

   1. **Choose the Network**: Select `testnet-alpha`, `testnet-beta`, or `mainnet`.
   2. **Set Username**: Input the desired username to run `poktrolld` (default: `poktroll`).
   3. **Set Node Moniker**: Input the node moniker (default: your `hostname`).
   4. **Confirm Seeds and Genesis File**: The script fetches seeds and the genesis file automatically.
   5. **External IP Address**: The script detects your external IP address. Confirm or input manually if incorrect.

### Verify successful installation using `curl`

We are going to use `curl` to query the latest block to verify the installation was successful.

Running the following command will return the latest synched block height:

```bash
curl -X GET http://localhost:26657/block | jq '.result.block.header.height'
```

Or the following command to get the entire block:

```bash
curl -X GET http://localhost:26657/block | jq
```

Which should return a response similar to the following format:

```json
{
  "jsonrpc": "2.0",
  "id": -1,
  "result": {
    "block_id": {
      "hash": "924904A2FB97327D2D91EB18225041B3DF82D1DBA5BA988AB79CD3EAC4A4960C",
      "parts": {
        "total": 1,
        "hash": "90E8EDC6841779CF4BADE35CDB53AA1276153BD26690999C5E87EB0E49E91AC8"
      }
    },
    "block": {
      "header": {
        "version": {
          "block": "11"
        },
        "chain_id": "pocket-beta",
        "height": "4971",
        "time": "2024-11-25T21:33:54.785576474Z",
        "last_block_id": {
          "hash": "E1D9F26882FD28447063CC11D326331C4B7C4A6417B2B2E5E38C5484C6D98168",
          "parts": {
            "total": 1,
            "hash": "85847883D9A34F345A2C3E610E1EC524B3C12F41DD2BDC49B36824D9A12EAB32"
          }
        },
        "last_commit_hash": "D49C2BF69F43658D63EF78487258DCA05F7239554E668CF9AE2502A5C6DB104E",
        "data_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
        "validators_hash": "5DC32F6AF7A2B6BAF1738FC5ADC8760E3A1A33A98839071D6A6FE503AD3BD52E",
        "next_validators_hash": "5DC32F6AF7A2B6BAF1738FC5ADC8760E3A1A33A98839071D6A6FE503AD3BD52E",
        "consensus_hash": "048091BC7DDC283F77BFBF91D73C44DA58C3DF8A9CBC867405D8B7F3DAADA22F",
        "app_hash": "DEACCBB96F23B7B58CADAFBE7894DDC2C5ACA0F29A68EA1C67407FA06C8D617C",
        "last_results_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
        "evidence_hash": "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
        "proposer_address": "21FABB12F80DAFF6CB83BA0958B2509FC127C3BD"
      },
      "data": {
        "txs": []
      },
      "evidence": {
        "evidence": []
      },
      "last_commit": {
        "height": "4970",
        "round": 0,
        "block_id": {
          "hash": "E1D9F26882FD28447063CC11D326331C4B7C4A6417B2B2E5E38C5484C6D98168",
          "parts": {
            "total": 1,
            "hash": "85847883D9A34F345A2C3E610E1EC524B3C12F41DD2BDC49B36824D9A12EAB32"
          }
        },
        "signatures": [
          {
            "block_id_flag": 2,
            "validator_address": "21FABB12F80DAFF6CB83BA0958B2509FC127C3BD",
            "timestamp": "2024-11-25T21:33:54.770507235Z",
            "signature": "zQb3QPt032nIRTUc7kk4cSxgVF4hpMZycE6ZvpSSZM4Bj1XlOEcdFtHWiLsileVX9RkZHqChzGBstCnfCfK8Bg=="
          },
          ...
        ]
      }
    }
  }
}
```

### How are automatic upgrades handled out of the box?

Your node is configured to handle chain upgrades automatically through Cosmovisor. No manual intervention is required for standard upgrades.

When a chain upgrade is proposed and approved:

1. Cosmovisor will download the new binary
2. The node will stop at the designated upgrade height
3. Cosmovisor will switch to the new binary
4. The node will restart automatically

## Do you care to know what just happened?

:::info Optional reading for the curious

This section is optional and for informational purposes only.

:::

If you're interested in understanding what just got installed, keep reading...

1. **System User**: A dedicated user (default: `poktroll`) is created to run the node securely.

2. **Cosmovisor**: A binary manager that handles chain upgrades automatically:

   - **Location**: `/home/poktroll/bin/cosmovisor`
   - **Purpose**: Manages different versions of `poktrolld` and handles chain upgrades
   - **Configuration**: Set up to automatically download and switch to new binaries during upgrades

3. **Poktrolld**: The core node software:

   - **Location**: `/home/poktroll/.poktroll/cosmovisor/genesis/bin/poktrolld`
   - **Configuration**: `/home/poktroll/.poktroll/config/`
   - **Data**: `/home/poktroll/.poktroll/data/`

4. **Systemd Service**: A service that manages the node:
   - **Name**: `cosmovisor.service`
   - **Status**: Enabled and started automatically
   - **Configured** for automatic restarts and upgrades
