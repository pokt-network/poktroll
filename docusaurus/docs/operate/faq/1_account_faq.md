---
sidebar_position: 2
title: Account FAQ
---

## Why am I getting an `account sequence mismatch` error when trying to submit a `bank send` transaction?

When trying to submit a `bank send` transaction like so:

```bash
pocketd tx bank send src dest 42upokt
```

If you get the following error:

```bash
code: 32
codespace: sdk
data: ""
events: []
gas_used: "0"
gas_wanted: "0"
height: "0"
info: ""
logs: []
raw_log: 'account sequence mismatch, expected 12, got 11: incorrect account sequence'
timestamp: ""
tx: null
txhash: REDACTED
```

You'll need to set the `account_number` and `sequence` manually.

```bash
pocketd query auth account pokt1fpxstscnqtzc9tq0u0nlvg69hqt6zhaxyka2mj --output json |
jq '
  .account.value |
  {
    account_number: (.account_number // "0"),
    next_sequence: ((.sequence | tonumber) + 1 | tostring)
  }
'
```

You can read more about the details [here](https://ctrl-felix.medium.com/how-do-i-get-the-cosmos-account-number-and-sequence-3f1643af285a)
