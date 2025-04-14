---
title: Failed upgrade contingency plan
sidebar_position: 8
---

## Contingency plans <!-- omit in toc -->

There's always a chance an upgrade will fail due to a variety of unknown unknowns.

This document is intended to help you recover with minimal downtime.

- [Option 0: The bug is discovered before the upgrade height is reached](#option-0-the-bug-is-discovered-before-the-upgrade-height-is-reached)
- [Option 1: The migration didn't start (i.e. migration halt)](#option-1-the-migration-didnt-start-ie-migration-halt)
- [Option 2: The migration is stuck (i.e. incomplete/partial migration)](#option-2-the-migration-is-stuck-ie-incompletepartial-migration)
- [Option 3: The migration succeed but the network is stuck (i.e. migration had a bug)](#option-3-the-migration-succeed-but-the-network-is-stuck-ie-migration-had-a-bug)
- [Failed Upgrade Checklist](#failed-upgrade-checklist)

### Option 0: The bug is discovered before the upgrade height is reached

**tl;dr cancel the upgrade plan!**

See the instructions of [how to do that here](3_upgrade_procedure.md#cancelling-the-upgrade-plan).

### Option 1: The migration didn't start (i.e. migration halt)

**tl;dr This is unlikely to happen.**

Possible reasons for this are if the name of the upgrade handler is different
from the one specified in the upgrade plan, or if the binary suggested by the
upgrade plan is wrong.

If the nodes on the network stopped at the upgrade height and the migration did not
start yet (i.e. there are no logs indicating the upgrade handler and store migrations are being executed),
we **MUST** gather social consensus to restart validators with the `--unsafe-skip-upgrade=$upgradeHeightNumber` flag.

This will skip the upgrade process, allowing the chain to continue and the protocol team to plan another release.

`--unsafe-skip-upgrade` simply skips the upgrade handler and store migrations.
The chain continues as if the upgrade plan was never set.
The upgrade needs to be fixed, and then a new plan needs to be submitted to the network.

:::caution

`--unsafe-skip-upgrade` needs to be documented in the list of upgrades and added
to the scripts so the next time somebody tries to sync the network from genesis,
they will automatically skip the failed upgrade.

**TODO_IMPROVE(@okdas): Provide more documentation here and details on how cosmovisor UX can simplify this.**

:::

### Option 2: The migration is stuck (i.e. incomplete/partial migration)

**tl;dr Requires social consensus and protocol team support to issue a new upgrade.**

If the migration is stuck, there's always a chance the upgrade handler was executed onchain as scheduled, but the migration didn't complete.

In such a case, we need:

1. **All full nodes and validators: Roll back validators to the backup.** A snapshot is taken by `cosmovisor` automatically prior to upgrade when `UNSAFE_SKIP_BACKUP` is set to `false` (the default recommended value; [more information](https://docs.cosmos.network/main/build/tooling/cosmovisor#command-line-arguments-and-environment-variables))

2. **All full nodes and validators: skip the upgrade.** Add the `--unsafe-skip-upgrade=$upgradeHeightNumber` argument to `pocket start` command like so:

   ```bash
   pocketd start --unsafe-skip-upgrade=$upgradeHeightNumber # ... the rest of the arguments
   ```

3. **Protocol team: Resolve the issue with an upgrade and schedule a new plan.** The upgrade needs to be fixed, and then a new plan needs to be submitted to the network.

4. **Protocol team: Document the failed upgrade.**

   - Document and add `--unsafe-skip-upgrade=$upgradeHeightNumber` to the scripts (such as docker-compose and cosmovisor installer)
   - The next time somebody tries to sync the network from genesis they will automatically skip the failed upgrade

<!-- TODO_IMPROVE(@okdas): new cosmovisor UX can simplify this -->

### Option 3: The migration succeed but the network is stuck (i.e. migration had a bug)

**tl;dr This should be treated as a consensus or non-determinism bug that is unrelated to the upgrade.**

See [Recovery From Chain Halt](9_recovery_from_chain_halt.md) for more information on how to handle such issues.

### Failed Upgrade Checklist

The following is a list of documentation & scripts that need to be updated on a failed upgrade:

- [ ] The [upgrade list](4_upgrade_list.md) should reflect a failed upgrade and provide a range of heights that served by each version.
- [ ] Systemd service should include`--unsafe-skip-upgrade=$upgradeHeightNumber` argument in its start command [here](https://github.com/pokt-network/poktroll/blob/main/tools/installer/full-node.sh).
- [ ] The [Helm chart](https://github.com/pokt-network/helm-charts/blob/main/charts/pocketd/templates/StatefulSet.yaml) should point to the latest version;consider exposing via a `values.yaml` file
- [ ] The [docker-compose](https://github.com/pokt-network/poktroll-docker-compose-example/tree/main/scripts) examples should point to the latest version
