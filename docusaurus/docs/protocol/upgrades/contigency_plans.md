---
title: Failed upgrade contingency plan
sidebar_position: 5
---

# Contingency plans <!-- omit in toc -->

There's always a chance the upgrade will fail. We have prepared some contingency plans, so we can recover without significant downtime.

:::tip

This documentation covers failed upgrade contingency for `poktroll` - a `cosmos-sdk` based chain. While this can be helpful for other blockchain networks, it is not guaranteed to work for other chains.

:::

- [Option 0: The bug is discovered before the upgrade height is reached](#option-0-the-bug-is-discovered-before-the-upgrade-height-is-reached)
- [Option 1: The upgrade height is reached and the migration didn't start](#option-1-the-upgrade-height-is-reached-and-the-migration-didnt-start)
- [Option 2: The migration is stuck](#option-2-the-migration-is-stuck)
- [Option 3: The network is stuck at the future height after the upgrade](#option-3-the-network-is-stuck-at-the-future-height-after-the-upgrade)


## Option 0: The bug is discovered before the upgrade height is reached

Cancel the upgrade plan: [how](./upgrade_procedure.md#cancelling-the-upgrade-plan).

## Option 1: The upgrade height is reached and the migration didn't start

If the nodes on the network stopped at the upgrade height and the migration did not start yet (there are no logs indicating the upgrade handler and store migrations are being executed), we should gather a social consensus to restart validators with the `--unsafe-skip-upgrade=$upgradeHeightNumber` flag. This will skip the upgrade process, allowing the chain to continue and the protocol team to plan another release.

`--unsafe-skip-upgrade` simply skips the upgrade handler and store migrations, and the chain continues as if the upgrade plan was never set. The upgrade needs to be fixed, and then a new plan needs to be submitted to the network.

:::caution
`--unsafe-skip-upgrade` needs to be documented and added to the scripts so the next time somebody tries to sync the network from genesis - they will automatically skip the failed upgrade.

<!-- TODO: new cosmovisor UX can simplify this -->
:::

## Option 2: The migration is stuck

If the migration is stuck, there's always a chance the state has been mutated for the upgrade but the migration didn't complete. In such a case, we need to:

- Roll back validators to the backup (a snapshot is taken by `cosmovisor` automatically prior to upgrade, if `UNSAFE_SKIP_BACKUP` is set to `false`). 
- Skip the upgrade handler and store migrations with `--unsafe-skip-upgrade=$upgradeHeightNumber`.
- Document and add `--unsafe-skip-upgrade=$upgradeHeightNumber` to the scripts so the next time somebody tries to sync the network from genesis - they will automatically skip the failed upgrade.
- Resolve the issue with an upgrade and schedule another plan.

## Option 3: The network is stuck at the future height after the upgrade

This should be treated as a consensus or non-determinism bug that is unrelated to the upgrade. See [Recovery From Chain Halt](../../develop/developer_guide/recovery_from_chain_halt.md) for more information on how to handle such issues.