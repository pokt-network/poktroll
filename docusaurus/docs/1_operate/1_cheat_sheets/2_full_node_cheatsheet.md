---
title: Full Node Script (~10 min)
sidebar_position: 2
---

import ReactPlayer from "react-player";

**Quick copy-paste guide to deploy a `Full Node` on Pocket Network using `Systemd` + `Cosmovisor`.**

:::warning Scripting abstracted

Some steps are scripted/automated. For full details, see the [Full Node Walkthrough](../2_walkthroughs/1_full_node_binary.md).

:::

---

## Table of Contents <!-- omit in toc -->

- [Prerequisites \& Requirements](#prerequisites--requirements)
- [10-Minute Video Walkthrough](#10-minute-video-walkthrough)
- [Install \& Run Full Node (Cosmovisor)](#install--run-full-node-cosmovisor)
  - [Verify install with `curl`](#verify-install-with-curl)
  - [Automatic upgrades?](#automatic-upgrades)
- [Want to know what just happened?](#want-to-know-what-just-happened)

## Prerequisites & Requirements

:::tip Vultr Playbook

Using [Vultr](https://www.vultr.com/)? Use our [CLI Playbook](../5_playbooks/1_vultr.md) for faster setup.

:::

- **Linux** (Debian/Ubuntu preferred)
- **Hardware**: [See requirements](../4_faq/6_hardware_requirements.md)
- **CPU**: x86_64 (amd64) or ARM64
- **Root/Sudo access**
- **Dedicated server/VM** (any provider)

## 10-Minute Video Walkthrough

Watch this quick video walkthrough of the process:

<ReactPlayer
  playing={false}
  controls
  url="https://github.com/user-attachments/assets/745cc1a4-28ee-4c02-8b22-858263e1f018"
/>

## Install & Run Full Node (Cosmovisor)

:::info

Script installs dependencies, creates user, sets env vars, configures Cosmovisor + `pocketd`.

:::

**Quick install:**

1. **Download script:**

   ```bash
   curl -O https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/full-node.sh
   ```

2. **Run as sudo:**

   ```bash
   sudo bash full-node.sh
   ```

3. **Follow prompts:**
   - Choose network: `testnet-alpha`, `testnet-beta`, or `mainnet`
   - Set username (default: `pocket`)
   - Set node moniker (default: hostname)
   - Seeds/genesis auto-fetched
   - Confirm external IP (auto-detected, or enter manually)

### Verify install with `curl`

Check node status:

- **Block height:**

  ```bash
  curl -X GET http://localhost:26657/block | jq '.result.block.header.height'
  ```

- **Full block:**

  ```bash
  curl -X GET http://localhost:26657/block | jq
  ```

You should see a response like this:

<details>
<summary>Block response</summary>

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
        "chain_id": "pocket-lego-testnet",
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

</details>

### Automatic upgrades?

- Cosmovisor handles upgrades automatically.
- No manual action needed for standard upgrades.
- What happens:
  1. Cosmovisor downloads new binary
  2. Node stops at upgrade height
  3. Cosmovisor switches to new binary
  4. Node restarts automatically

## Want to know what just happened?

:::info Optional reading

- This section is for info only.
  :::

What the script did:

- **System user**: Created (default: `pocket`)
- **Cosmovisor**:
  - Location: `/home/pocket/bin/cosmovisor`
  - Manages `pocketd` versions, handles upgrades
- **pocketd**:
  - Location: `/home/pocket/.pocket/cosmovisor/genesis/bin/pocketd`
  - Config: `/home/pocket/.pocket/config/`
  - Data: `/home/pocket/.pocket/data/`
- **Systemd service**:
  - Name: `cosmovisor.service`
  - Enabled, auto-starts, auto-restarts, auto-upgrades
