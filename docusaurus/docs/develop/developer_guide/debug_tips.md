---
sidebar_position: 2
title: Debugging Tips
---

# Debugging Tips <!-- omit in toc -->

:::note
This is a living document and will be updated as the ecosystem matures & grows.

If you have a tip you'd like to share with others, please open a PR to add it here!
:::

- [`itest` - Investigating Flaky Tests](#itest---investigating-flaky-tests)
  - [`itest` Usage](#itest-usage)
  - [`itest` Example](#itest-example)
  - [TODO: pprof](#todo-pprof)
  - [TODO: dlv](#todo-dlv)
- [`poktrolld query tx` - Investigating Failed Transactions](#poktrolld-query-tx---investigating-failed-transactions)
  - [`poktrolld query tx` Example](#poktrolld-query-tx-example)
   
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

## `poktrolld query tx` - Investigating Failed Transactions

If a transaction, e.g. staking a new service, is successfully posted but does not seem to have taken effect,
it could be due to any error that prevented the corresponding state transition from taking place.
In other words, receiving a transaction (TX) hash doesn't mean it was committed.
But the transaction (TX) hash can be used to investigate the failed transaction.

### `poktrolld query tx` Example

The following is an example of `poktrolld query tx` in action to investigate a failed transaction.
In this example, the command to add a new service is executed as follows, returning the TX hash shown.
However, the service does not appear in the list of services when querying the full node.

```bash
poktrolld tx service add-service "svc1" "service1" 1 --from $SUPPLIER_ADDRESS --chain-id=poktroll
```

The TX hash is returned by the above command:
```bash
txhash: 9E4CA2B72FCD6F74C771A5B2289CEACED30C2717ABEA4330E12543D3714D322B
```

To investigate this issue, the following command is used to get the details of the transaction:

```bash
poktrolld query tx --type=hash 9E4CA2B72FCD6F74C771A5B2289CEACED30C2717ABEA4330E12543D3714D322B
```

Which shows the following log entry:

```bash
info: ""
logs: []
raw_log: 'failed to execute message; message index: 0: account has 100000 uPOKT, but
  the service fee is 1000000000 uPOKT: not enough funds to add service'
```

The output above shows the cause of the transaction failure: insufficient funds. Fixing this by adding
more funds to the corresponding supplier account will allow the transaction to result in the expected
state transition.

### TODO: pprof

### TODO: dlv
