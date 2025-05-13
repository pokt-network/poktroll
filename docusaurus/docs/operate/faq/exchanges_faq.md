---
sidebar_position: 1
title: Exchanges FAQ
---
## Terminology

- Morse: The current version of Pocket Network with which your exchange has an integration.
- Shannon: The next version of Pocket Network that your exchange is integrating

## Background
[Pocket Network](https://pocket.network) will undergo a major, consensus-breaking, non-backwards-compatible migration from Morse to Shannon before the end of Q2. This migration will require exchanges to update their integrations to the new version of Pocket Network, which is billed in an upgrade, but technologically speaking, it is a hard fork.

## What **IS NOT** changing?
- The name: **Pocket Network**
- The ticker symbol: **$POKT**
- Liquidity and State: 
    - A snapshot of the current state of Pocket Network will occur within the 14-day window. 
    - A 1:1 migration of liquidity and state from the snapshot will migrate over to the new network. 

## What **IS** changing?
- The new Pocket Network is switching to using the [Cosmos Cryptogrpahic Key Scheme](https://docs.cosmos.network/main/learn/beginner/accounts), and therefore, a new wallet must be created.


## Timeline
- There will be a 14-day window prior to the migration where exchanges will need to:
    - Freeze deposits and withdrawals until the migration is complete.
    - Integrate with the new Pocket Network client using the [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet) or obtain a Foundation-sponsored RPC endpoint to the new network from [Grove](https://grove.city). 

