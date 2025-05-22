---
title: Migration E2E Testing (LocalNet)
sidebar_position: 11
---

**Goal of this document:** Manually test Supplier Claiming on LocalNet with clear, step-by-step, copy-pasteable commands.

:::warning Experience required

This document assumes you are a developer or operator and are familiar with concepts
like `pocket`, `pocketd`, `LocalNet`, etc...

:::

## Table of Contents <!-- omit in toc -->

- [Prerequisites](#prerequisites)
- [Morse Setup (`pocket`)](#morse-setup-pocket)
  - [Morse Account Preparation](#morse-account-preparation)
  - [Morse State Preparation](#morse-state-preparation)
- [State Upload](#state-upload)
- [Shannon Setup (`pocketd`)](#shannon-setup-pocketd)
  - [Shannon Account Preparation](#shannon-account-preparation)
  - [5. Shannon Supplier Claim](#5-shannon-supplier-claim)
    - [5.1 Claim WITHOUT Output Address](#51-claim-without-output-address)
    - [5.2 Claim WITH Output Address - Signed by Owner](#52-claim-with-output-address---signed-by-owner)
    - [5.3 Claim WITH Output Address - Signed by Operator](#53-claim-with-output-address---signed-by-operator)

## Prerequisites

- `pocket` installed
- `pocketd` installed
- You know how to run a shannon `LocalNet`

## Morse Setup (`pocket`)

### Morse Account Preparation

**Create 4 accounts:**

1. PNF & Validator (1)
2. Supplier Address/Operator (no output address)
3. Supplier Address/Operator (with output address)
4. Supplier Owner/Output (1)

```bash
for i in {1..4}; do pocket accounts create --datadir ./pocket_test; done
```

**List accounts:**

```bash
pocket accounts list --datadir ./pocket_test
```

**Example output:**

```text
(0) 6280986b72469fe3817d9382cf52ec310f1dddcc
(1) 997b833caceb0d5f139e3bcb6fe1f4e2a3f7d02d
(2) 9d7bc65655e9aa38122da324fc5c73ab417e09c6
(3) efad4318739151de95af4a0b3709291f387e8d66
```

**Assign addresses to variables**:

```bash
ADDR1="6280986b72469fe3817d9382cf52ec310f1dddcc"
ADDR2="997b833caceb0d5f139e3bcb6fe1f4e2a3f7d02d"
ADDR3="9d7bc65655e9aa38122da324fc5c73ab417e09c6"
ADDR4="efad4318739151de95af4a0b3709291f387e8d66"
```

**Export keys:**

```bash
pocket accounts export $ADDR2 --datadir ./pocket_test
pocket accounts export $ADDR3 --datadir ./pocket_test
pocket accounts export $ADDR4 --datadir ./pocket_test
```

Which will create a few `pocket-account-*.json` files in your current directory.

**Retrieve their public keys:**

```bash
pocket accounts show $ADDR2 --datadir ./pocket_test
pocket accounts show $ADDR3 --datadir ./pocket_test
pocket accounts show $ADDR4 --datadir ./pocket_test
```

Assign public keys to variables:

```bash
PUBKEY2="e0f82c9c1843b320e0436ee25abc67a536a452973f83030183a99bab5dc67f27"
PUBKEY3="ccc15d61fa80c707cb55ccd80b61720abbac13ca56f7896057e889521462052d"
PUBKEY4="32a60f6e5217ef1e6fa6cbbed376db3cce64277ab19947624e309483185bf92f"
```

### Morse State Preparation

Make a copy of `example_state_export.json` to `testing_state_export.json`.

```bash
cp docusaurus/docs/2_explore/4_morse_migration/example_state_export.json testing_state_export.json
```

Edit `testing_state_export.json` to match these addresses:

- `ADDR1`: PNF & Validator (not used)
- `ADDR2`: Supplier Address/Operator (no output address)
- `ADDR3`: Supplier Address/Operator (with output address)
- `ADDR4`: Supplier Owner/Output

By running the following command:

```bash
sed -i.bak -e "s/\"ADDR1\"/\"$ADDR1\"/g" \
           -e "s/\"ADDR2\"/\"$ADDR2\"/g" \
           -e "s/\"ADDR3\"/\"$ADDR3\"/g" \
           -e "s/\"ADDR4\"/\"$ADDR4\"/g" \
           -e "s/\"PUBKEY1\"/\"$PUBKEY1\"/g" \
           -e "s/\"PUBKEY2\"/\"$PUBKEY2\"/g" \
           -e "s/\"PUBKEY3\"/\"$PUBKEY3\"/g" \
           testing_state_export.json
```

**Generate import message:**

```bash
pocketd tx migration collect-morse-accounts \
  testing_state_export.json msg_import_morse_accounts.json \
  --home=./localnet/pocketd
```

And optionally inspect `msg_import_morse_accounts.json`.

## State Upload

**Start LocalNet:**

```bash
make localnet_up
make acc_initialize_pubkeys
```

**Upload Morse state:**

```bash
pocketd tx migration import-morse-accounts \
  ./msg_import_morse_accounts.json \
  --from=pnf \
  --grpc-addr=localhost:9090 \
  --home=./localnet/pocketd --keyring-backend=test \
  --chain-id=pocket \
  --gas=auto --gas-adjustment=1.5
```

**Check claimable accounts:**

```bash
pocketd query migration list-morse-claimable-account \
  -o json --node=tcp://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

**Example output**:

```json
...
{
  "morse_src_address": "9D7BC65655E9AA38122DA324FC5C73AB417E09C6",
  "unstaked_balance": {
    "denom": "upokt",
    "amount": "20000000000000"
  },
  "supplier_stake": {
    "denom": "upokt",
    "amount": "60000000000"
  },
  "application_stake": {
    "denom": "upokt",
    "amount": "0"
  },
  "morse_output_address": "EFAD4318739151DE95AF4A0B3709291F387E8D66"
}
...
```

## Shannon Setup (`pocketd`)

### Shannon Account Preparation

Create prefix variables:

```bash
ADDR2_PREFIX=${ADDR2:0:4}
ADDR3_PREFIX=${ADDR3:0:4}
ADDR4_PREFIX=${ADDR4:0:4}
```

**Create 3 new accounts:**

```bash
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${ADDR2_PREFIX}-claim-supplier-1
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${ADDR3_PREFIX}-claim-supplier-2
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${ADDR4_PREFIX}-claim-owner
```

**Export addresses:**

```bash
ADDR_SUPPLIER_1=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${ADDR2_PREFIX}-claim-supplier-1 -a)
ADDR_SUPPLIER_2=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${ADDR3_PREFIX}-claim-supplier-2 -a)
ADDR_OWNER=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${ADDR4_PREFIX}-claim-owner -a)
```

**Fund all accounts:**

```bash
pocketd tx bank send pnf $ADDR_SUPPLIER_1 1000000000000upokt --home=./localnet/pocketd --yes
sleep 2
pocketd tx bank send pnf $ADDR_SUPPLIER_2 1000000000000upokt --home=./localnet/pocketd --yes
sleep 2
pocketd tx bank send pnf $ADDR_OWNER 1000000000000upokt --home=./localnet/pocketd --yes
```

**Check balances:**

```bash
pocketd query bank balances $ADDR_SUPPLIER_1 --home ./localnet/pocketd
pocketd query bank balances $ADDR_SUPPLIER_2 --home ./localnet/pocketd
pocketd query bank balances $ADDR_OWNER --home ./localnet/pocketd
```

### 5. Shannon Supplier Claim

#### 5.1 Claim WITHOUT Output Address

**Create config:**

```bash
cat <<EOF > ${ADDR2_PREFIX}_claim_supplier_1_supplier_config.yaml
owner_address: ${ADDR_SUPPLIER_1}
operator_address: ${ADDR_SUPPLIER_1}
default_rev_share_percent:
  ${ADDR_SUPPLIER_1}: 100
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Claim:**

```bash
pocketd tx migration claim-supplier \
  ${ADDR2} pocket-account-${ADDR2}.json \
  ${ADDR2_PREFIX}_claim_supplier_1_supplier_config.yaml \
  --from=${ADDR2_PREFIX}-claim-supplier-1 \
  --node=http://localhost:26657 --chain-id=pocket \
  --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
  --gas=auto --gas-adjustment=1.5 --yes
```

**Verify onchain:**

```bash
pocketd query supplier show-supplier $ADDR_SUPPLIER_1 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd
```

**Check stake:**

```bash
pocketd query supplier show-supplier $ADDR_SUPPLIER_1 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

**Check unstaked balance:**

```bash
pocketd query bank balance $ADDR_SUPPLIER_1 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

#### 5.2 Claim WITH Output Address - Signed by Owner

**Create config:**

```bash
cat <<EOF > ${ADDR3_PREFIX}_claim_supplier_2_supplier_config.yaml
owner_address: ${ADDR_OWNER}
operator_address: ${ADDR_SUPPLIER_2}
default_rev_share_percent:
  ${ADDR_OWNER}: 20
  ${ADDR_SUPPLIER_2}: 80
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Claim:**

```bash
pocketd tx migration claim-supplier \
  ${ADDR3} pocket-account-${ADDR3}.json \
  ${ADDR3_PREFIX}_claim_supplier_2_supplier_config.yaml \
  --from=${ADDR3_PREFIX}-claim-supplier-2 \
  --node=http://localhost:26657 --chain-id=pocket \
  --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
  --gas=auto --gas-adjustment=1.5 --yes
```

**Verify onchain:**

```bash
pocketd query supplier show-supplier $ADDR_SUPPLIER_2 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd
```

**Check stake:**

```bash
pocketd query supplier show-supplier $ADDR_SUPPLIER_2 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

**Check owner's unstaked balance:**

```bash
pocketd query bank balance $ADDR_OWNER upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

#### 5.3 Claim WITH Output Address - Signed by Operator
