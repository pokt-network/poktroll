---
sidebar_position: 7
title: Pocketdex Indexer
---

- [Usage](#usage)
  - [GraphQL](#graphql)
  - [Postgres - CLI](#postgres---cli)
- [Debugging](#debugging)
  - [Port already in use](#port-already-in-use)
  - [Yarn dependency installation failure](#yarn-dependency-installation-failure)

:::warning

This document is a living WIP and assumes you are familiar with the LocalNet environment.

:::

## Pocketdex <!-- omit in toc -->

[Pocketdex](https://github.com/pokt-network/pocketdex/), the pocket indexer starts up as part of the default LocalNet.

### Usage

#### GraphQL

The localnet graphiql playground is available at [http://localhost:3000](http://localhost:3000), by default.

![GraphiQL Playground](../../../static/img/pocketdex_graphiql_screenshot.png)

A link is accessible from the ["GraphQL API" tab in tilt](http://localhost:10350/r/GraphQL%20API/overview):

![LocalNet Dashboard](../../../static/img/pocketdex_graphiql_link.png)

See the [pocketdex docs](https://github.com/pokt-network/pocketdex?tab=readme-ov-file#usage--query-docs) for more details.

#### Postgres - CLI

You can connect using a tool of your choice or with the `psql` CLI via:

```bash
psql -h localhost -p 5432 -U postgres -d postgres
```

After you've connected, you MUST update your schema to `localnet` and start exploring the data:

```sql
set schema 'localnet';
\dt
select * from accounts limit 10; # Example query
```

### Debugging

#### Port already in use

If you go to [http://localhost:10350/r/Postgres/overview](http://localhost:10350/r/Postgres/overview) and see the following error:

```bash
Reconnecting... Error port-forwarding Postgres (5432 -> 5432): Unable to listen on port 5432: Listeners failed to create with the following errors: [unable to create listener: Error listen tcp4 127.0.0.1:5432: bind: address already in use unable to create listener: Error listen tcp6 [::1]:5432: bind: address already in use]
```

You likely have another local Postgres instance running. You can identify it by running

```bash
lsof -i:5432
```

On macOS, if installed via `brew`, it can be stopped with:

```bash
brew services stop postgresql
```

#### Yarn dependency installation failure

If you see the pocketdex indexer failing to build in Tilt with yarn dependency errors like this:

![Pocketdex Tilt Error](./img/pocketdex_tilt_error.png)

This typically indicates issues with corrupted dependencies or improperly initialized git submodules. To resolve:

1. **Change to the local pocketdex directory**: Navigate to the pocketdex directory:

   ```bash
   cd ../pocketdex
   ```

2. **Clean all dependencies**: Run the cleanup script to remove all node_modules and dist directories:

   ```bash
   yarn run clean:all
   ```

3. **Update git submodules**: Ensure all submodules are properly cloned and checked out:

   ```bash
   git submodule update --init
   ```

4. **Restart the indexer in Tilt**: After cleaning, trigger a rebuild of the indexer resource in the Tilt UI

:::note Why does this happen?

This error often occurs when git submodules aren't properly initialized or when there are stale/corrupted dependencies from previous builds.

:::
