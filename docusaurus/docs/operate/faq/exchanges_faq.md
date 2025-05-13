---
sidebar_position: 1
title: Exchanges FAQ
---
## Terminology

- **Morse**: The current version of Pocket Network with which your exchange has an integration.
- **Shannon**: The next version of Pocket Network that your exchange is integrating.
- **Foundation**: The Pocket Network Foundation, which is responsible for the migration.

## Background
[Pocket Network](https://pocket.network) will undergo a major, consensus-breaking, non-backwards-compatible migration from Morse to Shannon before the end of Q2. 

This migration will require exchanges to: 
 - Update their integrations to the new version of Pocket Network, which is billed in an upgrade, but technologically, it is a hard fork.
 - Maintain a full-node to the new network or use a Foundation-vendored RPC endpoint.
 - Claim their current liquidiy of $POKT tokens on the new network.
 
## What **IS NOT** changing?
- The name: **Pocket Network**
- The ticker: **$POKT**
- Liquidity and State: 
    - A snapshot of the current state of Pocket Network will occur within a 14-day migration window. 
    - A 1:1 migration of liquidity and state from the snapshot will migrate over to the new network.

## What **IS** changing? (And what you must do to prepare)
- **Accounts/Wallets**
  - A new wallet must be created as Pocket Network is switching to using the [Cosmos Cryptogrpahic Key Scheme](https://docs.cosmos.network/main/learn/beginner/accounts).
  - The [poktrolld CLI tool](https://dev.poktroll.com/explore/account_management/create_new_account_cli) can create a new wallet.
   Two third-party wallet integrations are also supported; [Keplr Wallet](https://www.keplr.app/) and [Soothe Vault](https://trustsoothe.io/).
- **Token Minting on Morse**
    - In the same block a snapshot is taken of liquidity and state on Morse, a governance transaction will be run to turn off minting of $POKT on Morse. This is to ensure accurate state preservation between Morse and Shannon.
    - Due to the lack of new token minting, the incentive to keep validators up to process Morse requests will be reduced and Morse will stop functioning.
- **Full Node Integration**
  - Integrate with the new Pocket Network client by using: 
      - The [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet)
      - Obtaining a Foundation-sponsored RPC endpoint to the new network from [Grove](https://grove.city).

## Timeline
- There will be a 14-day window prior to the migration where exchanges will need to do a handful of items.
  - Before the Migration Window
    - Disable deposits and withdrawals until the migration is complete.
    - Integrate with the new Pocket Network client (see above.).
    - Notify the Foundation that you are prepared to migrate.
  - After the Migration
    - Claim tokens on the new network  
    - Enable deposits and withdrawals
    Notify the Foundation that you have completed the migration.

## Migration