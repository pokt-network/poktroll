---
title: Claiming Morse Validators
sidebar_position: 9
---

## Table of Contents <!-- omit in toc -->

- [Can I claim my Morse Validator as a Shannon Validator](#can-i-claim-my-morse-validator-as-a-shannon-validator)
  - [0. More Information](#0-more-information)
- [How do I claim my Morse Validator as a Shannon Validator?](#how-do-i-claim-my-morse-validator-as-a-shannon-validator)
  - [0. Prerequisites](#0-prerequisites)
  - [1. Prepare your Morse and Shannon Keys and Accounts](#1-prepare-your-morse-and-shannon-keys-and-accounts)
  - [2. Options for Claiming and becoming a Shannon Validator](#2-options-for-claiming-and-becoming-a-shannon-validator)

## Can I claim my Morse Validator as a Shannon Validator?

**No, you cannot claim a Morse Validator directly as a Shannon Validator.**

### 0. More Information

In Morse, Validators and Suppliers are coupled. Validators are "elected" to create blocks based on being in the top 1000 staked Validators. 

In Shannon, Validators and Suppliers are fundamentally different actors that require different stakes. 

Like in Morse, Validators in Shannon are also stake-weighted for election into the active pool. You can [read more about it in the Validator FAQ](../../1_operate/4_faq/3_validator_faq.md).

## What are my options to claim my Morse Validator as a Shannon Validator?

### 0. Prerequisites

- You have read the [Claiming Introduction](./5_claiming_introduction.md)
- You have installed the Morse `pocket` CLI
- You have installed the Shannon `pocketd` CLI
- You have imported your Morse key into a keyring
- You have a valid RPC endpoint
- You are familiar with how to stake a native Shannon Supplier (see [supplier staking config](../../1_operate/3_configs/3_supplier_staking_config.md))

### 1. Prepare your Morse and Shannon Keys and Accounts

Follow steps 1-5 from [Claiming Morse Account](./6_claiming_account.md)

### 2. Options for Claiming and becoming a Shannon Validator

**Option #1:** Unstake and Restake as Validator
- Unstake on Morse or Shannon 
- Wait for the Unbonding Period (21 days) 
- [Claim your Account](./6_claiming_account.md) on Shannon 
- [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon

**Option #2:** Claim a Supplier and Restake on Shannon 
- [Claim your Supplier](./7_claiming_supplier.md) on Shannon 
- Unstake on Shannon
- Wait for the Unbonding Period (21 days)
- [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon

**Option #3:** Special Case Migration 
_Due to the importance of Validators, special cases can be covered in the Snapshot process of the Migration_
- Reach out to the [Pocket Network Foundation on Discord](https://discord.com/invite/pocket-network)
- Share your address with the Pocket Network Foundation
- Grove🌿 will update the Morse Snapshot to Automatically Unstake (Skip the Unbonding Period) 
- [Claim your Account](./6_claiming_account.md) on Shannon 
- [Stake as a Validator](../../1_operate/2_walkthroughs/3_validator_walkthrough.md) on Shannon
