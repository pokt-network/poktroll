---
sidebar_position: 7
title: Third-Party Custodians FAQ
---

:::important

This document was last updated on 05/19/2025.

:::

:::info Who is this for?
**Target Audience:**

- Centralized exchanges and custodians integrating with Pocket Network
- Anyone responsible for $POKT custody, integration, or liquidity management
- Technical and operational teams preparing for the Morse → Shannon migration

:::

## Table of Contents <!-- omit in toc -->

- Migration Red0ne's docs
- Add keplr details + example
- Update validator docs
- Add keplr wallet (single or multi)
- Show how to claim MACT or Beta Tokens
- Add link to recoverable accounts
- Add link to accounts we'll auto liquidate
- Tell people to request getting auto unclaimed
- Explain unbonding
- Update docs with `pocketd relayminer relay`



I was trying to answer some other questions I had, here are some of them:

Q: Is there any on-chain concept of regions?
A: No. There is no regionality defined on-chain in Shannon.

Q: How are providers selected across regions?
A: Selection is handled by PATH QoS logic. Gateways select the best and fastest node in-session. Region is not a factor.

Q: What are the current Grove gateway regions?
A: Same regions as before: USE (US East), SGP (Singapore), EUC (EU Central)

Q: Is there a limit to how many services a supplier can stake? — -> Awaiting confirmation About this one !!!
??: No known protocol-level hard or soft limit ?
— There are some test suppliers staked onchain (beta testate) with ~10k service endpoints behind them.

cc @fred | Grove

Hope it helps. Let me know if any answer is incorrrect and I will edit it here also in order to avoid any confusion. 






- [Terminology](#terminology)
- [Background and Action Items for Exchanges](#background-and-action-items-for-exchanges)
- [What **IS NOT** Changing?](#what-is-not-changing)
- [What **IS** Changing?](#what-is-changing)
  - [Accounts/Wallets](#accountswallets)
  - [Token Minting on Morse](#token-minting-on-morse)
  - [Full Node Integration](#full-node-integration)
- [Actions Exchanges Must Take (Pre-Migration)](#actions-exchanges-must-take-pre-migration)
- [Timeline](#timeline)
  - [14-Day Migration Window](#14-day-migration-window)
- [Communication](#communication)

## Terminology

| Term           | Meaning                                                              |
| -------------- | -------------------------------------------------------------------- |
| **Morse**      | _Current_ version of Pocket Network your exchange is integrated with |
| **Shannon**    | _Upcoming_ version of Pocket Network (will replace Morse)            |
| **Foundation** | Pocket Network Foundation (operational migration lead)               |
| **Grove**      | Official labs team (technical migration lead)                        |

---

## Background and Action Items for Exchanges

[Pocket Network](https://pocket.network) will hard-fork from Morse MainNet to Shannon MainNet on June 3, 2025 at 10 a.m. PDT. This is consensus-breaking, non-backwards-compatible upgrade. Activities for June 3, 2025 can be found [here](https://medium.com/decentralized-infrastructure/pocket-network-shannon-state-shift-day-b8c06122cb76).

**Exchanges must:**

- Update their integrations to the new version (hard fork).
- Maintain a full-node to the new network _OR_ use a Foundation-vendored RPC endpoint.
- Claim current `$POKT` token liquidity on the new network.

---

## What **IS NOT** Changing?

| Category          | Details                                                                                                                                                                                                                                                               |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Network Name      | **Pocket Network**                                                                                                                                                                                                                                                    |
| Chain Ticker      | **$POKT**                                                                                                                                                                                                                                                             |
| Liquidity & State | - Snapshot of Morse state during a 14-day migration window <br/> - 1:1 migration to Shannon                                                                                                                                                                           |
| Tokenomics        | - [Mint = Burn model launched April 2025](https://forum.pokt.network/t/protocol-economics-parameters-for-the-shannon-upgrade/5490) <br/> - Weekly manual burning (track at [pokt.money](https://pokt.money)) <br/> - After migration, burning will occur in real time |

---

## What **IS** Changing?

### Accounts/Wallets

- The Shannon upgrade uses the [Cosmos Cryptographic Key Scheme](https://docs.cosmos.network/main/learn/beginner/accounts).
- A new Cosmos-based wallet is required to claim tokens.
- Tokens will need to be claimed manually.

### Token Minting on Morse

- $POKT token minting will stop within the same block that the snapshot is taken.
- Morse will stop functioning as there will be no incentives for validators.

### Full Node Integration

- Integrate with new client:
  - [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet)
  - or get a free Foundation-sponsored RPC endpoint from [Grove](https://portal.grove.city)

---

## Actions Exchanges Must Take (Pre-Migration)

| Action                | How-To / Links                                                                                                                                                                                  |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Create New Wallet** | - [pocketd CLI tool](https://dev.poktroll.com/explore/account_management/create_new_account_cli) <br/> - [Keplr Wallet](https://www.keplr.app/) <br/> - [Soothe Vault](https://trustsoothe.io/) |
| **Integrate Node**    | - [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet) <br/> - or get a free Foundation-sponsored RPC endpoint from [Grove](https://portal.grove.city)    |
| **Claim Tokens**      | - CLI: [pocketd CLI tool](https://dev.poktroll.com/explore/morse_migration/claiming_account) <br/> - UI: Website with offline signing support (Coming Soon)                                     |

:::warning TODO(Grove): Coming Soon

- Pocket specific tutorial on using the Keplr wallet
- Pocket specific tutorial on using the Soothe wallet
- Pocket specific tutorial on using the Cosmos multisig wallets
- Tutorial for token claiming website for both online and offline use cases
- Video tutorial on claiming tokens

Have ideas for other improvements? Please open an issue [here](https://github.com/pokt-network/poktroll/issues/new?template=issue.md).

:::

---

## Timeline

### 14-Day Migration Window

**Before Migration Window:**

- Disable deposits/withdrawals until migration is complete.
- Integrate with the pocket client (see above).
- Notify the Foundation in the shared Telegram group when these actions have been taken.

**After Migration:**

- Claim the tokens on the new network (see above).
- Re-enable deposits/withdrawals.
- Notify the Foundation in the shared Telegram group when these actions have been taken.

---

## Communication

- Each custodian should already have a Telegram group with Foundation & Grove personnel.
- For reference, these personnel are located in the US-East timezone, and may therefore have a delayed response time.
- Use your group to ask questions or update on progress.

:::note Where to reach out?

Our assumption is that if you're reading this, you're already in touch with the Foundation team.

If not, please reach out to the community via discord at [https://discord.gg/pocket-network](https://discord.gg/pocket-network).

:::
