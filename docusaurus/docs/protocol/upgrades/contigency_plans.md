---
title: Failed upgrade contingency plan
sidebar_position: 5
---

:::tip

This documentation covers failed upgrade contingency for `poktroll` - a `cosmos-sdk` based chain.

While this can be helpful for other blockchain networks, it is not guaranteed to work for other chains.

:::

## Contingency plans <!-- omit in toc -->

There's always a chance the upgrade will fail.

This document is intended to help you recover without significant downtime.

- [Option 0: The bug is discovered before the upgrade height is reached](#option-0-the-bug-is-discovered-before-the-upgrade-height-is-reached)
- [Option 1: The upgrade height is reached and the migration didn't start](#option-1-the-upgrade-height-is-reached-and-the-migration-didnt-start)
- [Option 2: The migration is stuck](#option-2-the-migration-is-stuck)
- [Option 3: The network is stuck at the future height after the upgrade](#option-3-the-network-is-stuck-at-the-future-height-after-the-upgrade)
- [Documentation and scripts to update](#documentation-and-scripts-to-update)

### Option 0: The bug is discovered before the upgrade height is reached

**Cancel the upgrade plan!**

See the instructions of [how to do that here](./upgrade_procedure.md#cancelling-the-upgrade-plan).

### Option 1: The upgrade height is reached and the migration didn't start

If the nodes on the network stopped at the upgrade height and the migration did not
start yet (i.e. there are no logs indicating the upgrade handler and store migrations are being executed),
we MUST gather social consensus to restart validators with the `--unsafe-skip-upgrade=$upgradeHeightNumber` flag.

This will skip the upgrade process, allowing the chain to continue and the protocol team to plan another release.

`--unsafe-skip-upgrade` simply skips the upgrade handler and store migrations.
The chain continues as if the upgrade plan was never set.
The upgrade needs to be fixed, and then a new plan needs to be submitted to the network.

:::caution

`--unsafe-skip-upgrade` needs to be documented in the list of upgrades and added to the scripts so the next time somebody tries to sync the network from genesis - they will automatically skip the failed upgrade. [Documentation and scripts to update](#documentation-and-scripts-to-update)

<!-- TODO_IMPROVE(@okdas): new cosmovisor UX can simplify this -->

:::

### Option 2: The migration is stuck

If the migration is stuck, there's always a chance the upgrade handler was executed on-chain as scheduled, but the migration didn't complete.

In such a case, we need to:

- Roll back validators to the backup. A snapshot is taken by `cosmovisor` automatically prior to upgrade when`UNSAFE_SKIP_BACKUP` is set to `false` (which is a default and recommended value -
  [more information](https://docs.cosmos.network/main/build/tooling/cosmovisor#command-line-arguments-and-environment-variables)).
- **All full nodes and validators**: skip the upgrade by adding `--unsafe-skip-upgrade=$upgradeHeightNumber`
  argument to your `poktroll start` command. Like this:
  ```bash
  poktrolld start --unsafe-skip-upgrade=$upgradeHeightNumber # ... the rest of the arguments
  ```
- **Protocol team**: document and add `--unsafe-skip-upgrade=$upgradeHeightNumber` to the scripts (such as docker-compose and cosmovisor installer) so the next time somebody
  tries to sync the network from genesis they will automatically skip the failed upgrade. [Documentation and scripts to update](#documentation-and-scripts-to-update)
- Resolve the issue with an upgrade and schedule another plan.

<!-- TODO_IMPROVE(@okdas): new cosmovisor UX can simplify this -->

### Option 3: The network is stuck at the future height after the upgrade

This should be treated as a consensus or non-determinism bug that is unrelated to the upgrade. See [Recovery From Chain Halt](../../develop/developer_guide/recovery_from_chain_halt.md) for more information on how to handle such issues.

### Documentation and scripts to update

- The [upgrade list](./upgrade_list.md) should reflect a failed upgrade and provide a range of heights that served by each version.
- Systemd service should include`--unsafe-skip-upgrade=$upgradeHeightNumber` argument in its start command [here](https://github.com/pokt-network/poktroll/blob/main/tools/installer/full-node.sh).
- [Helm chart](https://github.com/pokt-network/helm-charts/blob/main/charts/poktrolld/templates/StatefulSet.yaml) (consider exposing via a `values.yaml` file)
- [docker-compose](https://github.com/pokt-network/poktroll-docker-compose-example/tree/main/scripts) example 