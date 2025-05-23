---
title: Migration E2E Testing (LocalNet)
sidebar_position: 11
---

:::danger TODO(@bryanchriswhite) - Verify

1. Verify this flow E2E
2. Add one-liners on expected checks/outputs along the way

:::

:::warning Read Me First

- You must already know what `pocket`, `pocketd`, and `LocalNet` are.
- If you are not a developer/operator, stop here.

:::

**Goal of this document:**

- Manually test Supplier Claiming on LocalNet with clear, step-by-step, copy-pasteable commands.
- This guide is **idiot-proofed**: every step is explicit, warnings are surfaced, and copy-paste actions are clearly marked.

## Quick Navigation

- [Quick Navigation](#quick-navigation)
- [Prerequisites](#prerequisites)
- [Morse Setup (`pocket`)](#morse-setup-pocket)
  - [Morse Account Preparation](#morse-account-preparation)
  - [Morse State Preparation](#morse-state-preparation)
- [State Upload](#state-upload)
- [Shannon Setup (`pocketd`)](#shannon-setup-pocketd)
  - [Shannon Account Preparation](#shannon-account-preparation)
  - [Option 1: Supplier Claim WITH Output Address - Signed by Owner](#option-1-supplier-claim-with-output-address---signed-by-owner)
  - [Option 2: Supplier Claim WITH Output Address - Signed by Operator](#option-2-supplier-claim-with-output-address---signed-by-operator)
  - [Option 3: Supplier Claim WITHOUT Output Address](#option-3-supplier-claim-without-output-address)

---

## Prerequisites

- `pocket` **must be installed**
- `pocketd` **must be installed**
- You **must** know how to run a shannon `LocalNet`
- You need access to a shell with `jq`, `sed`, and `make`

## Morse Setup (`pocket`)

### Morse Account Preparation

Create 6 Accounts with these roles:

- (1) PNF & Validator (same address for both)
- (2-4) Operators (suppliers)
- (5-6) Owners (outputs)

```bash
for i in {1..6}; do pocket accounts create --datadir ./morse_pocket_datadir; done
```

_When prompted for a password, just press Enter (leave it empty)._

**Verify accounts were created:**

```bash
pocket accounts list --datadir ./morse_pocket_datadir
```

_You should see 6 addresses. Example:_

```text
(0) 1c9d96c0bd1a98c90151a804f18e9ba75dae12b4
(1) 86ff8ecdce4c93def67d018967fcbeebfed253bf
(2) 96b41ff38115b23d34e0201a16953a9088243bf3
(3) a1bc4dc57ca80a58953fea7438cedba2b4141abe
(4) dda8fe050d21511dd3b58bf5b6d81428573bc986
(5) f761c00d797baa4a3ac9b7d7248394c412d1e047
```

**Assign addresses to variables**

Assign addresses to variables using the actual values (example shown below; use your real output if different):

```bash
MORSE_ADDR_PNF="1c9d96c0bd1a98c90151a804f18e9ba75dae12b4"
MORSE_ADDR_SUPPLIER_1="86ff8ecdce4c93def67d018967fcbeebfed253bf"
MORSE_ADDR_SUPPLIER_2="96b41ff38115b23d34e0201a16953a9088243bf3"
MORSE_ADDR_SUPPLIER_3="a1bc4dc57ca80a58953fea7438cedba2b4141abe"
MORSE_ADDR_OWNER_1="dda8fe050d21511dd3b58bf5b6d81428573bc986"
MORSE_ADDR_OWNER_2="f761c00d797baa4a3ac9b7d7248394c412d1e047"
```

_⚠️ Double check you use the right address for each variable. Copy-paste with care!_

**Export keys:**

```bash
pocket accounts export $MORSE_ADDR_SUPPLIER_1 --datadir ./morse_pocket_datadir
pocket accounts export $MORSE_ADDR_SUPPLIER_2 --datadir ./morse_pocket_datadir
pocket accounts export $MORSE_ADDR_SUPPLIER_3 --datadir ./morse_pocket_datadir
pocket accounts export $MORSE_ADDR_OWNER_1 --datadir ./morse_pocket_datadir
pocket accounts export $MORSE_ADDR_OWNER_2 --datadir ./morse_pocket_datadir
```

_This creates several `pocket-account-*.json` files in your current directory._

**Check files exist:**

```bash
ls -la pocket-account*
```

**Retrieve their public keys:**

```bash
pocket accounts show $MORSE_ADDR_PNF --datadir ./morse_pocket_datadir
pocket accounts show $MORSE_ADDR_SUPPLIER_1 --datadir ./morse_pocket_datadir
pocket accounts show $MORSE_ADDR_SUPPLIER_2 --datadir ./morse_pocket_datadir
pocket accounts show $MORSE_ADDR_SUPPLIER_3 --datadir ./morse_pocket_datadir
pocket accounts show $MORSE_ADDR_OWNER_1 --datadir ./morse_pocket_datadir
pocket accounts show $MORSE_ADDR_OWNER_2 --datadir ./morse_pocket_datadir
```

Assign the public keys to variables using the actual values (example shown below; use your real output if different):

```bash
MORSE_PNF_PUBKEY="765c466ba9fdd182a0e4fb1c5968aaa0a76f00caea06d0cfbfd524366c85433a"
MORSE_SUPPLIER_PUBKEY_1="7a9d685014154504e302af75f36986e31ce7cd1b3e7fd6e27a13ce04c003b333"
MORSE_SUPPLIER_PUBKEY_2="6916b93ee96e8cee6774edf23c908f79a3372eba91ccd15e62c16c6658669056"
MORSE_SUPPLIER_PUBKEY_3="27f16c70ad256af90b8b35cd021d6e4b05dc6b770e3d862d45d8cda9e00b79d8"
MORSE_OWNER_PUBKEY_1="da23b83d40485c506a692804f6a50b11e4bffceb492e5e1dfda5829cabc7c1e2"
MORSE_OWNER_PUBKEY_2="7aa876179e5b2acd4c69dd359b075dfb9a614ac7567097fb324658f94b2563c6"
```

_⚠️ Double check all assignments!_

### Morse State Preparation

**Copy the example state file:**

```bash
cp \
  docusaurus/docs/2_explore/4_morse_migration/example_state_export.json \
  localnet_testing_state_export.json
```

**Replace placeholder variables in the new file:**

```bash
sed -i.bak -e "s/\"MORSE_ADDR_PNF\"/\"$MORSE_ADDR_PNF\"/g" \
           -e "s/\"MORSE_ADDR_SUPPLIER_1\"/\"$MORSE_ADDR_SUPPLIER_1\"/g" \
           -e "s/\"MORSE_ADDR_SUPPLIER_2\"/\"$MORSE_ADDR_SUPPLIER_2\"/g" \
           -e "s/\"MORSE_ADDR_SUPPLIER_3\"/\"$MORSE_ADDR_SUPPLIER_3\"/g" \
           -e "s/\"MORSE_ADDR_OWNER_1\"/\"$MORSE_ADDR_OWNER_1\"/g" \
           -e "s/\"MORSE_ADDR_OWNER_2\"/\"$MORSE_ADDR_OWNER_2\"/g" \
          -e "s/\"MORSE_PNF_PUBKEY\"/\"$MORSE_PNF_PUBKEY\"/g" \
           -e "s/\"MORSE_SUPPLIER_PUBKEY_1\"/\"$MORSE_SUPPLIER_PUBKEY_1\"/g" \
           -e "s/\"MORSE_SUPPLIER_PUBKEY_2\"/\"$MORSE_SUPPLIER_PUBKEY_2\"/g" \
           -e "s/\"MORSE_SUPPLIER_PUBKEY_3\"/\"$MORSE_SUPPLIER_PUBKEY_3\"/g" \
           -e "s/\"MORSE_OWNER_PUBKEY_1\"/\"$MORSE_OWNER_PUBKEY_1\"/g" \
           -e "s/\"MORSE_OWNER_PUBKEY_2\"/\"$MORSE_OWNER_PUBKEY_2\"/g" \
           localnet_testing_state_export.json
```

**Generate import message:**

```bash
pocketd tx migration collect-morse-accounts \
  localnet_testing_state_export.json localnet_testing_msg_import_morse_accounts.json \
  --home=./localnet/pocketd
```

- This creates `localnet_testing_msg_import_morse_accounts.json`.
- (Optional) Inspect the file if you want to verify contents.

## State Upload

**Start LocalNet:**

```bash
make localnet_up
make acc_initialize_pubkeys
```

_Wait for all services to be fully up before continuing._

**Upload Morse state:**

```bash
pocketd tx migration import-morse-accounts \
  localnet_testing_msg_import_morse_accounts.json \
  --from=pnf \
  --grpc-addr=localhost:9090 \
  --home=./localnet/pocketd --keyring-backend=test \
  --chain-id=pocket \
  --gas=auto --gas-adjustment=1.5
```

_⚠️ This command does not output anything. If it returns to prompt, it likely succeeded._

**Check claimable accounts:**

```bash
pocketd query migration list-morse-claimable-account \
  -o json --node=tcp://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

_⚠️ You should see **exactly 6** accounts in the output! If not, something is wrong._

**Example output:**

```json
...
  {
    "morse_src_address": "F761C00D797BAA4A3AC9B7D7248394C412D1E047",
    "unstaked_balance": {
      "denom": "upokt",
      "amount": "20000000000000"
    },
    "supplier_stake": {
      "denom": "upokt",
      "amount": "0"
    },
    "application_stake": {
      "denom": "upokt",
      "amount": "0"
    }
  }
...
```

## Shannon Setup (`pocketd`)

### Shannon Account Preparation

**Create prefix variables:**

```bash
MORSE_SUPPLIER_1_PREFIX=${MORSE_ADDR_SUPPLIER_1:0:4}
MORSE_SUPPLIER_2_PREFIX=${MORSE_ADDR_SUPPLIER_2:0:4}
MORSE_SUPPLIER_3_PREFIX=${MORSE_ADDR_SUPPLIER_3:0:4}
MORSE_OWNER_1_PREFIX=${MORSE_ADDR_OWNER_1:0:4}
MORSE_OWNER_2_PREFIX=${MORSE_ADDR_OWNER_2:0:4}
```

**Create 5 new Shannon accounts (3 suppliers, 2 owners):**

```bash
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_1_PREFIX}-claim-supplier-1
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_3_PREFIX}-claim-supplier-3
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_OWNER_1_PREFIX}-claim-owner-1
pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_OWNER_2_PREFIX}-claim-owner-2
```

**Export addresses:**

```bash
SHANNON_ADDR_SUPPLIER_1=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_1_PREFIX}-claim-supplier-1 -a)
SHANNON_ADDR_SUPPLIER_2=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2 -a)
SHANNON_ADDR_SUPPLIER_3=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_3_PREFIX}-claim-supplier-3 -a)
SHANNON_ADDR_OWNER_1=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_OWNER_1_PREFIX}-claim-owner-1 -a)
SHANNON_ADDR_OWNER_2=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_OWNER_2_PREFIX}-claim-owner-2 -a)
```

_These variables will be used in all subsequent steps._

**Fund all accounts:**

```bash
pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_1 1000000000000upokt --home=./localnet/pocketd --yes --unordered --timeout-duration=5s
pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_2 1000000000000upokt --home=./localnet/pocketd --yes --unordered --timeout-duration=5s
pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_3 1000000000000upokt --home=./localnet/pocketd --yes --unordered --timeout-duration=5s
pocketd tx bank send pnf $SHANNON_ADDR_OWNER_1 1000000000000upokt --home=./localnet/pocketd --yes --unordered --timeout-duration=5s
pocketd tx bank send pnf $SHANNON_ADDR_OWNER_2 1000000000000upokt --home=./localnet/pocketd --yes --unordered --timeout-duration=5s
```

_If any command fails, **stop and debug before continuing**._

**Check balances:**

```bash
pocketd query bank balances $SHANNON_ADDR_SUPPLIER_1 --home ./localnet/pocketd
pocketd query bank balances $SHANNON_ADDR_SUPPLIER_2 --home ./localnet/pocketd
pocketd query bank balances $SHANNON_ADDR_SUPPLIER_3 --home ./localnet/pocketd
pocketd query bank balances $SHANNON_ADDR_OWNER_1 --home ./localnet/pocketd
pocketd query bank balances $SHANNON_ADDR_OWNER_2 --home ./localnet/pocketd
```

_Each account should show a balance. If not, **fix before proceeding**._

---

### Option 1: Supplier Claim WITH Output Address - Signed by Owner

**Create config:**

```bash
cat <<EOF > ${MORSE_SUPPLIER_1_PREFIX}_claim_supplier_1_supplier_config.yaml
owner_address: ${SHANNON_ADDR_OWNER_1}
operator_address: ${SHANNON_ADDR_SUPPLIER_1}
default_rev_share_percent:
  ${SHANNON_ADDR_OWNER_1}: 80
  ${SHANNON_ADDR_SUPPLIER_1}: 20
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Check stake before claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_1 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

_This should error (supplier doesn't exist yet)._

**Check owner's unstaked balance before claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_OWNER_1 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Check supplier's unstaked balance before claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_1 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Submit the onchain claim:**

```bash
pocketd tx migration claim-supplier \
  ${MORSE_ADDR_SUPPLIER_1} pocket-account-${MORSE_ADDR_OWNER_1}.json \
  ${MORSE_SUPPLIER_1_PREFIX}_claim_supplier_1_supplier_config.yaml \
  --from=${MORSE_OWNER_1_PREFIX}-claim-owner-1 \
  --node=http://localhost:26657 --chain-id=pocket \
  --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
  --gas=auto --gas-adjustment=1.5 --yes
```

_If this fails, **do not continue** until resolved._

**Verify supplier exists onchain:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_1 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd
```

**Check stake after claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_1 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

**Check owner's unstaked balance after claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_OWNER_1 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Check supplier's unstaked balance after claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_1 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

---

### Option 2: Supplier Claim WITH Output Address - Signed by Operator

**Create config:**

```bash
cat <<EOF > ${MORSE_SUPPLIER_2_PREFIX}_claim_supplier_2_supplier_config.yaml
owner_address: ${SHANNON_ADDR_OWNER_2}
operator_address: ${SHANNON_ADDR_SUPPLIER_2}
default_rev_share_percent:
  ${SHANNON_ADDR_OWNER_2}: 80
  ${SHANNON_ADDR_SUPPLIER_2}: 20
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Check stake before claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_2 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

_This should error (supplier doesn't exist yet)._.

**Check owner's unstaked balance before claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_OWNER_2 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Check supplier's unstaked balance before claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_2 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Submit the onchain claim:**

```bash
pocketd tx migration claim-supplier \
  ${MORSE_ADDR_SUPPLIER_2} pocket-account-${MORSE_ADDR_SUPPLIER_2}.json \
  ${MORSE_SUPPLIER_2_PREFIX}_claim_supplier_2_supplier_config.yaml \
  --from=${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2 \
  --node=http://localhost:26657 --chain-id=pocket \
  --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
  --gas=auto --gas-adjustment=1.5 --yes
```

**Verify supplier exists onchain:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_2 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd
```

**Check stake after claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_2 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

**Check owner's unstaked balance after claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_OWNER_2 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Check supplier's unstaked balance after claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_2 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

---

### Option 3: Supplier Claim WITHOUT Output Address

**Create config:**

```bash
cat <<EOF > ${MORSE_SUPPLIER_3_PREFIX}_claim_supplier_3_supplier_config.yaml
owner_address: ${SHANNON_ADDR_SUPPLIER_3}
operator_address: ${SHANNON_ADDR_SUPPLIER_3}
default_rev_share_percent:
  ${SHANNON_ADDR_SUPPLIER_3}: 100
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Check stake before claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_3 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

_This should error (supplier doesn't exist yet)._.

**Check unstaked balance before claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_3 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```

**Submit the onchain claim:**

```bash
pocketd tx migration claim-supplier \
  ${MORSE_ADDR_SUPPLIER_3} pocket-account-${MORSE_ADDR_SUPPLIER_3}.json \
  ${MORSE_SUPPLIER_3_PREFIX}_claim_supplier_3_supplier_config.yaml \
  --from=${MORSE_SUPPLIER_3_PREFIX}-claim-supplier-3 \
  --node=http://localhost:26657 --chain-id=pocket \
  --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
  --gas=auto --gas-adjustment=1.5 --yes
```

**Verify supplier exists onchain:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_3 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd
```

**Check stake after claim:**

```bash
pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_3 -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

**Check unstaked balance after claim:**

```bash
pocketd query bank balance $SHANNON_ADDR_SUPPLIER_3 upokt -o json --node=http://127.0.0.1:26657 --home=./localnet/pocketd | jq '.balance.amount'
```
