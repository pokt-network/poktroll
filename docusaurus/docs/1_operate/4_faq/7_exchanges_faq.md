---
sidebar_position: 7
title: Exchanges FAQ
---

:::important

This document was last updated on 05/14/2025.

:::

:::info Who is this for?
**Target Audience:**

- Centralized exchanges and custodians integrating with Pocket Network
- Anyone responsible for $POKT custody, integration, or liquidity management
- Technical and operational teams preparing for the Morse → Shannon migration

:::

## Table of Contents <!-- omit in toc -->

- [Terminology](#terminology)
- [Background and Action Items for Exchanges](#background-and-action-items-for-exchanges)
- [What **IS NOT** Changing?](#what-is-not-changing)
- [What **IS** Changing? (What You Must Do)](#what-is-changing-what-you-must-do)
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

[Pocket Network](https://pocket.network) is migrating from Morse to Shannon (consensus-breaking, non-backwards-compatible upgrade) before end of Q2.

**Exchanges must:**

- Update integrations to the new version (hard fork).
- Maintain a full-node to the new network _OR_ use a Foundation-vendored RPC endpoint.
- Claim current `$POKT` token liquidity on the new network.

---

## What **IS NOT** Changing?

| Category          | Details                                                                                                                                                                                                                                                      |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Network Name      | **Pocket Network**                                                                                                                                                                                                                                           |
| Chain Ticker      | **$POKT**                                                                                                                                                                                                                                                    |
| Liquidity & State | - Snapshot of Morse state during a 14-day migration window<br>- 1:1 migration to Shannon                                                                                                                                                                     |
| Tokenomics        | - [Mint = Burn model launched April 2025](https://forum.pokt.network/t/protocol-economics-parameters-for-the-shannon-upgrade/5490)<br>- Weekly manual burning (track at [pokt.money](https://pokt.money))<br>- After migration, burning will be in real time |

---

## What **IS** Changing? (What You Must Do)

### Accounts/Wallets

- Shannon uses [Cosmos Cryptographic Key Scheme](https://docs.cosmos.network/main/learn/beginner/accounts).
- New wallet required; manual claiming of tokens needed.

### Token Minting on Morse

- Snapshot block disables $POKT minting on Morse via governance transaction.
- Morse will stop functioning (no incentive for validators).

### Full Node Integration

- Integrate with new client:
  - [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet)
  - Or get a Foundation-sponsored RPC endpoint from [Grove](https://portal.grove.city)

---

## Actions Exchanges Must Take (Pre-Migration)

| Action                | How-To / Links                                                                                                                                                                                        |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Create New Wallet** | - [pocketd CLI tool](https://dev.poktroll.com/explore/account_management/create_new_account_cli)<br>- [Keplr Wallet](https://www.keplr.app/)<br>- [Soothe Vault](https://trustsoothe.io/)             |
| **Integrate Node**    | - [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet)<br>- Or get Foundation RPC endpoint from [Grove](https://portal.grove.city) (DM Jinx/Arthur in Telegram) |
| **Claim Tokens**      | - CLI: [pocketd CLI tool](https://dev.poktroll.com/explore/morse_migration/claiming_account)<br>- UI: Website (Coming Soon)                                                                           |

:::warning TODO(Grove): Coming Soon

- Pocket specific tutorial on using the Keplr wallet
- Pocket specific tutorial on using the Cosmos multisig wallets
- Video tutorial on claiming tokens

Got ideas for other improvements? Please open an issue [here](https://github.com/pokt-network/poktroll/issues/new?template=issue.md).

:::

---

## Timeline

### 14-Day Migration Window

**Before Migration Window:**

- Disable deposits/withdrawals until migration is complete.
- Integrate with new client (see above).
- Notify Foundation when ready.

**After Migration:**

- Claim tokens on new network (see above).
- Re-enable deposits/withdrawals.
- Notify Foundation when complete.

---

## Communication

- Each exchange has a Telegram group with Foundation & Grove personnel (US-East timezone).
- Use your group to ask questions or update on progress.

:::note Where to reach out?

Our assumption is that if you're reading this, you're already in touch with the Foundation team.

If not, please reach out to the community via discord at [discord.gg/pokt](https://discord.gg/pokt).

:::
