---
title: Full Node Cheat Sheet
sidebar_position: 3
---

## Full Node Cheat Sheet Using Systemd & Cosmovisor <!-- omit in toc -->

This cheat sheet provides quick copy-pasta like instructions for installing and
running a Full Node using an automated script.

:::tip

If you're interested in understanding the underlying details, or having full control over every
step of the process, check out the [Full Node Walkthrough](../run_a_node/full_node_walkthrough.md).

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

#### Test that installation was successful
Query the latesrt block (i.e. check the node height)

```bash
curl -X GET http://localhost:26657/block | jq
```

If you set it up correctly, you should see a response in this form:

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
          {
            "block_id_flag": 2,
            "validator_address": "80A08A0BE0916161F4D6375B300C6F9ADA7A2EA3",
            "timestamp": "2024-11-25T21:33:54.785576474Z",
            "signature": "qppmmdFchx8RKT4qnNQJGXV8/ukHW7PgI2fg9hrMU55HKosX0nTaz0yFIrDDL3TmHScKhdRYess777xT9T6VBA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "8AC6613D760B9B5D9C44F97A5893DB4AC9AF1ACB",
            "timestamp": "2024-11-25T21:33:54.82911715Z",
            "signature": "nlxxO5n0ueLRNjJXqk7afjPPsD4ltJLzPbC/VA3x9VygFZYPM6ngjBoNTIO1BQjnanfHGQrU39R4hGZwNLRnAA=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "9B1E1BA4443F8962F56FB46745A7FBFDD6694B4F",
            "timestamp": "2024-11-25T21:33:54.828959724Z",
            "signature": "soip0IoEaCwIl2zHB69h5ehCifAw/WhMSO1lE74YCJIxWDSSPFQKQYAMiEBQ5K3+iaONHj7g+yIs4Sk9eCAWAQ=="
          },
          {
            "block_id_flag": 2,
            "validator_address": "F9854E3CB13E69501B5C596C1FB7A44DB231B38E",
            "timestamp": "2024-11-25T21:33:54.770123943Z",
            "signature": "NA4Y8Yv+z2wUprtT/mV+/lUBRyqF0BmBVFIpfHHjmq3ATsU2SmX5ur1nmukzWx5NiKPCW+fhowRfNH1u+79zCg=="
          }
        ]
      }
    }
  }
}
```

### FAQ & Troubleshooting

See the [FAQ & Troubleshooting section in the Full Node Walkthrough](../run_a_node/full_node_walkthrough.md#faq--troubleshooting)
for examples of useful commands, common debugging instructions and other advanced usage.

### [OPTIONAL] Do you care to know what just happened?

:::info
This section is optional and for informational purposes only.
:::

If you're interested in understanding what just got installed, keep reading...

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
