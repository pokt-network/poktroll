---
title: Claiming Morse Validators
sidebar_position: 9
---

## Table of Contents <!-- omit in toc -->

- [Can I claim my Morse Validator as a Shannon Validator?](#can-i-claim-my-morse-validator-as-a-shannon-validator)
- [Background Information on Morse \& Shannon Validators](#background-information-on-morse--shannon-validators)
- [Prerequisites](#prerequisites)
- ["Claiming" a Shannon Validator](#claiming-a-shannon-validator)
  - [1. Prepare your Morse and Shannon Keys and Accounts](#1-prepare-your-morse-and-shannon-keys-and-accounts)
  - [2. Options for Claiming a Shannon Validator](#2-options-for-claiming-a-shannon-validator)
    - [Option #1: Unstake and Restake as Validator](#option-1-unstake-and-restake-as-validator)
    - [Option #2: Claim a Supplier and Restake on Shannon](#option-2-claim-a-supplier-and-restake-on-shannon)
    - [Option #3: Special Cases Validator Migration](#option-3-special-cases-validator-migration)

## Can I claim my Morse Validator as a Shannon Validator?

**No. You cannot claim a Morse Validator directly as a Shannon Validator.**

## Background Information on Morse & Shannon Validators

In Morse, Validators and Suppliers are coupled.

Validators are "elected" to create blocks based on being in the top 1000 staked Validators.

In Shannon, Validators and Suppliers are decoupled as fundamentally different actors that require different stakes.

Validators in Shannon are also stake-weighted for election into the active pool. You can [read more about it in the Validator FAQ](../../1_operate/4_faq/3_validator_faq.md).

## Prerequisites

- You have read the [Claiming Introduction](./5_claiming_introduction.md)
- You have installed the Morse `pocket` CLI
- You have installed the Shannon `pocketd` CLI
- You have imported your Morse key into a keyring
- You have a valid RPC endpoint
- You are familiar with how to stake a native Shannon Supplier (see [supplier staking config](../../1_operate/3_configs/3_supplier_staking_config.md))

## "Claiming" a Shannon Validator

### 1. Prepare your Morse and Shannon Keys and Accounts

Follow steps 1-5 from [Claiming Morse Account](./6_claiming_account.md)

### 2. Options for Claiming a Shannon Validator

What do I do if I have a Morse Validator (a.k.a Node, Service) that I want to migrate to Shannon as a Validator?

#### Option #1: Unstake and Restake as Validator

1. Unstake Validator on Morse or Shannon
2. Wait for the Unbonding Period (e.g. 21 days)
3. [Claim your Account](./6_claiming_account.md) on Shannon
4. [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon

#### Option #2: Claim a Supplier and Restake on Shannon

1. [Claim your Supplier](./7_claiming_supplier.md) on Shannon
2. Unstake Supplier on Shannon
3. Wait for the Unbonding Period (e.g. 21 days)
4. [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon

#### Option #3: Special Cases Validator Migration

:::important Foundation Support

**Due to the importance of Validators, special cases can be covered in the Snapshot process of the Migration**

:::

1. Reach out to the [Pocket Network Foundation on Discord](https://discord.com/invite/pocket-network)
2. Share your address with the Pocket Network Foundation
3. PNF and Grove 🌿 will work together to update the Morse Snapshot to Automatically Unstake (Skip the Unbonding Period)
4. [Claim your Account](./6_claiming_account.md) on Shannon
5. [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon
