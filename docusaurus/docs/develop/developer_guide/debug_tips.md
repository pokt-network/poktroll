---
sidebar_position: 2
title: Debugging Tips
---

## Debugging Tips <!-- omit in toc -->

:::note
This is a living document and will be updated as the ecosystem matures & grows.

If you have a tip you'd like to share with others, please open a PR to add it here!
:::

- [`itest` - Investigating Flaky Tests](#itest---investigating-flaky-tests)
  - [`itest` Usage](#itest-usage)
  - [`itest` Example](#itest-example)
- [`pocketd query tx` - Investigating Failed Transactions](#pocketd-query-tx---investigating-failed-transactions)
  - [`pocketd query tx` Example](#pocketd-query-tx-example)
  - [TODO_DOCUMENT: pprof](#todo_document-pprof)
  - [TODO_DOCUMENT: dlv](#todo_document-dlv)

## `itest` - Investigating Flaky Tests

We developed a tool called `itest` to help with debugging flaky tests. It runs
the same test iteratively

### `itest` Usage

Run the following command to see the usage of `itest`:

```bash
./tools/scripts/itest.sh
```

### `itest` Example

The following is an example of `itest` in action to run the `TxClient_SignAndBroadcast_Succeeds`
test in the `pkg/client/tx` 50 times in total (5 consecutive tests over 10 runs).

```bash
make itest 5 10 ./pkg/client/tx/... -- -run TxClient_SignAndBroadcast_Succeeds
```

## `pocketd query tx` - Investigating Failed Transactions

_tl;dr Submitted Transaction != Committed Transaction_

After a transaction (e.g. staking a new service) is successfully sent to an RPC node, we have to wait
until the next block, when a proposer will try to commit to the network's state, to see if its valid.
If the transaction's (TX) state transition is invalid, it will not be committed.

In other words, receiving a transaction (TX) hash from the `pocketd` CLI doesn't mean it was committed.
However, the transaction (TX) hash can be used to investigate the failed transaction.

### `pocketd query tx` Example

The following is an example of `pocketd query tx` in action to investigate a failed transaction.
In this example, the command to add a new service is executed as follows, returning the TX hash shown.
However, the service does not appear in the list of services when querying the full node.

```bash
pocketd tx service add-service "svc1" "service1" 1 --from $SUPPLIER_ADDRESS --chain-id=pocket
```

The TX hash is returned by the above command:

```bash
txhash: 9E4CA2B72FCD6F74C771A5B2289CEACED30C2717ABEA4330E12543D3714D322B
```

To investigate this issue, the following command is used to get the details of the transaction:

```bash
pocketd query tx \
--type=hash 9E4CA2B72FCD6F74C771A5B2289CEACED30C2717ABEA4330E12543D3714D322B \
--node https://shannon-testnet-grove-seed-rpc.poktroll.com
```

Which shows the following log entry:

```bash
info: ""
logs: []
raw_log: 'failed to execute message; message index: 0: account has 100000 uPOKT, but
  the service fee is 1000000000 uPOKT: not enough funds to add service'
```

The output above shows the cause of the transaction failure: `insufficient funds`. Fixing this by adding
more funds to the corresponding supplier account will allow the transaction to result in the expected
state transition.

:::note

If you are reading this and the `9E4CA...` hash is no longer valid, we may have done a re-genesis of
TestNet at this point. Please consider updating with a new one!

:::

:::tip

`pocketd query tx` supports an `--output` flag which can have the values text or json. This can be useful for programatic querying or in combination with tools like `jq`, e.g.:

```bash
pocketd query tx \
--type=hash 9E4CA2B72FCD6F74C771A5B2289CEACED30C2717ABEA4330E12543D3714D322B \
--node https://shannon-testnet-grove-seed-rpc.poktroll.com \
 --output json | jq .raw_log
```

The above command will produce the following output:

```bash
"failed to execute message; message index: 0: account has 100000 uPOKT, but the service fee is 1000000000 uPOKT: not enough funds to add service"
```

:::

### TODO_DOCUMENT: pprof

### TODO_DOCUMENT: dlv
