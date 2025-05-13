# LocalNet Supplier Claim <!-- omit in toc -->

These are manual copy-pasta instructions to test, experiment and showcase Supplier Claiming on a localnet.

## Table of Contents <!-- omit in toc -->

- [Morse (`pocketd`)](#morse-pocketd)
  - [Morse Account Preparation](#morse-account-preparation)
  - [Morse State Upload Preparation](#morse-state-upload-preparation)
  - [State Upload](#state-upload)
- [Shannon (`pocketd`)](#shannon-pocketd)
  - [Shannon Account Preparation](#shannon-account-preparation)
  - [Shannon Claim Suppliers](#shannon-claim-suppliers)
    - [Claim Shannon Supplier WITHOUT an output address](#claim-shannon-supplier-without-an-output-address)
    - [\[NOT IMPLEMENTED YET\] Option 1 - Claim Shannon Supplier WITH an output address - as an owner](#not-implemented-yet-option-1---claim-shannon-supplier-with-an-output-address---as-an-owner)
    - [\[NOT IMPLEMENTED YET\] Option 2 - Claim Shannon Supplier WITH an output address - as an operator](#not-implemented-yet-option-2---claim-shannon-supplier-with-an-output-address---as-an-operator)

## Morse (`pocketd`)

### Morse Account Preparation

**Create four accounts**:

```bash
# PNF & Validator (exactly one)
pocket accounts create --datadir ./pocket_test
# Supplier Address/Operator (one of two WITHOUT output address)
pocket accounts create --datadir ./pocket_test
# Supplier Address/Operator (two of two WITH output address)
pocket accounts create --datadir ./pocket_test
# Supplier Owner/Output (exactly one)
pocket accounts create --datadir ./pocket_test
```

**List accounts**:

```bash
pocket accounts list --datadir ./pocket_test
```

```bash
(0) 028026796df1d8450410eab29c710a5944eef8dd
(1) 2e2624762bcfee4a44001543adddce0e4f4cc823
(2) 80e3058d66ee75578b07472650483da0035febe6
(3) f9f5335adfe2f7c4e49ef5cf5856eded1c5d3c58
```

**Grab the account address and export the key**:

```bash
pocket accounts export 2e2624762bcfee4a44001543adddce0e4f4cc823 --datadir ./pocket_test
pocket accounts export 80e3058d66ee75578b07472650483da0035febe6 --datadir ./pocket_test
pocket accounts export f9f5335adfe2f7c4e49ef5cf5856eded1c5d3c58 --datadir ./pocket_test
```

### Morse State Upload Preparation

Manually update `temp_state_export.json` with the following to reflect the following configs

- `028026796df1d8450410eab29c710a5944eef8dd`: PNF & Validator (not used)
- `2e2624762bcfee4a44001543adddce0e4f4cc823`: Supplier Address/Operator (one of two WITHOUT output address)
- `80e3058d66ee75578b07472650483da0035febe6`: Supplier Address/Operator (two of two WITH output address)
- `f9f5335adfe2f7c4e49ef5cf5856eded1c5d3c58`: Supplier Owner/Output (exactly one)

**Prepare the import account message**:

```bash
pocketd tx migration collect-morse-accounts temp_state_export.json msg_import_morse_accounts.json
```

**Upload the state**:

```bash
pocketd tx migration import-morse-accounts \
  "./msg_import_morse_accounts.json" \
  --from=pnf \
  --grpc-addr=localhost:9090 \
  --home=./localnet/pocketd --keyring-backend="test"\
  --chain-id=pocket \
  --gas=auto --gas-adjustment=1.5
```

**And validate the list of claimable accounts**:

```bash
pocketd query migration list-morse-claimable-account \
  -o json --node=tcp://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

For example, you will see output like so showing that a supplier has a staked and unstaked balance:

```json
{
  "morse_src_address": "2E2624762BCFEE4A44001543ADDDCE0E4F4CC823",
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
  }
}
```

### State Upload

**Run LocalNet**:

```bash
make localnet_up
```

**Upload State**:

```bash
pocketd tx migration import-morse-state \
  "./temp_state_export.json" \
  --from=pnf \
  --grpc-addr=localhost:9090 \
  --home=./localnet/pocketd --keyring-backend="test"\
  --chain-id=pocket \
  --gas=auto --gas-adjustment=1.5 \
```

```bash
pocketd tx migration collect-morse-accounts temp_state_export.json msg_import_morse_accounts.json
```

## Shannon (`pocketd`)

### Shannon Account Preparation

**Create three new `pocketd` accounts**:

```bash
pocketd --keyring-backend="test" --home=./localnet/pocketd  keys add 2e26-claim-supplier-1
pocketd --keyring-backend="test" --home=./localnet/pocketd  keys add 80e3-claim-supplier-2
pocketd --keyring-backend="test" --home=./localnet/pocketd  keys add f9f5-claim-owner
```

**Export their addresses**:

```bash
ADDR_SUPPLIER_1=$(pocketd --keyring-backend="test" --home=./localnet/pocketd  keys show 2e26-claim-supplier-1 -a)
ADDR_SUPPLIER_2=$(pocketd --keyring-backend="test" --home=./localnet/pocketd  keys show 80e3-claim-supplier-2 -a)
ADDR_OWNER=$(pocketd --keyring-backend="test" --home=./localnet/pocketd  keys show f9f5-claim-owner -a)
```

**And fund them**:

```bash
pocketd tx bank send pnf $ADDR_SUPPLIER_1 1000000000000upokt --home ./localnet/pocketd
# Wait for it to be funded (~ 10 seconds until the tx is processed)
pocketd tx bank send pnf $ADDR_SUPPLIER_2 1000000000000upokt --home ./localnet/pocketd
# Wait for it to be funded (~ 10 seconds until the tx is processed)
pocketd tx bank send pnf $ADDR_OWNER 1000000000000upokt --home ./localnet/pocketd
```

**And ensure they're funded**:

```bash
pocketd query bank balances ${ADDR_SUPPLIER_1} --home ./localnet/pocketd
pocketd query bank balances ${ADDR_SUPPLIER_2} --home ./localnet/pocketd
pocketd query bank balances ${ADDR_OWNER} --home ./localnet/pocketd
```

### Shannon Claim Suppliers

| Morse / Shannon-sign Description    | Morse (`address`, `output_address`) | Shannon (`owner_address`, `operator_address`) | Claim Signer | Supported | Details / Notes / Explanation                                                                    | Pre-conditions                                                                                  |
| ----------------------------------- | ----------------------------------- | --------------------------------------------- | ------------ | --------- | ------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------- |
| custodial / owner-operator-sign     | (`M`, `M`)                          | (`S`, `S`)                                    | `S`          | ✅        | Custodial flow #1                                                                                | `S` owns `M`                                                                                    |
| custodial / owner-sign              | (`M1`, null)                        | (`S1`, null)                                  | `S1`         | ✅        | Custodial flow #1                                                                                | `S1` owns `M1`                                                                                  |
| non-custodial / owner-sign          | (`M1`, `M2`)                        | (`S1`, null)                                  | `S1`         | ❌        | MUST have `operator_address` if `output_address` exists for backwards-simplification             | NA                                                                                              |
| non-custodial / owner-sign          | (`M1`, `M2`)                        | (`S1`, `S2`)                                  | `S1`         | ✅        | Non-custodial flow executed by owner                                                             | (`S1` owns `M1`) && (`S2` owns `M2`) && (`M2` gives `S2` shannon staking instructions offchain) |
| non-custodial / operator-sign       | (`M1`, `M2`)                        | (`S1`, `S2`)                                  | `S2`         | ✅        | Non-custodial flow executed by operator                                                          | (`S1` owns `M1`) && (`S2` owns `M2`) && (`S2` gives `M2` shannon address offline)               |
| non-custodial / owner-sign          | (`M1`, null)                        | (`S1`, `S2`)                                  | `S2`         | ❌        | MUST NOT have `operator_address` if `output_address` does not exist for backwards-simplification | NA                                                                                              |
| missing operator / NA               | (null, `M2`)                        | NA                                            | NA           | ❌        | Not supported because `M1` cannot be null                                                        | NA                                                                                              |
| NA / missing owner                  | NA                                  | (null, `S2`)                                  | NA           | ❌        | Not supported because `S1` cannot be null                                                        | NA                                                                                              |
| non-custodial / owner-operator-sign | (`M1`, `M2`)                        | (`S`, `S`)                                    | `S`          | ❌        | `operator_address` must differ from `owner_address` for backwards-simplification                 | NA                                                                                              |

#### Claim Shannon Supplier WITHOUT an output address

Conditions whereby:

- **In Morse**: `operator_address` != null `output_address` = `null`
- **In Shannon**: `owner_address` = `operator_address`
- Claiming as a `owner_address` on behalf of `operator_address`

**Prepare a supplier stake config**:

```bash
cat <<EOF > 2e26_claim_supplier_1_supplier_config.yaml
owner_address: ${ADDR_SUPPLIER_1}
operator_address: ${ADDR_SUPPLIER_1}
default_rev_share_percent:
  ${ADDR_OWNER}: 100
services:
  - service_id: anvil
    endpoints:
      - publicly_exposed_url: http://relayminer1:8545
        rpc_type: JSON_RPC
EOF
```

**Claim the supplier**:

```bash
pocketd tx migration claim-supplier \
 pocket-account-2e2624762bcfee4a44001543adddce0e4f4cc823.json \
 2e26_claim_supplier_1_supplier_config.yaml \
 --from=2e26-claim-supplier-1 \
 --node=http://localhost:26657 --chain-id=pocket \
 --home=./localnet/pocketd --keyring-backend=test --no-passphrase
```

**And verify it is onchain**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_1} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

You can check its **stake**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_1} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

You can check its **unstaked balance**:

```bash
pocketd query bank balance ${ADDR_SUPPLIER_1} upokt \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.balance.amount'
```

#### [NOT IMPLEMENTED YET] Option 1 - Claim Shannon Supplier WITH an output address - as an owner

**Conditions whereby**:

- **In Morse**: `output_address` != `null` & `operator_address` != null
- **In Shannon**: `owner_address` != `operator_address`
- Claiming as a `owner_address` on behalf of `output_address`

**Prepare a supplier stake config**:

```bash
cat <<EOF > 80e3_claim_supplier_2_supplier_config.yaml
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

**Claim the supplier**:

```bash
pocketd tx migration claim-supplier \
 pocket-account-80e3058d66ee75578b07472650483da0035febe6.json \
 80e3_claim_supplier_2_supplier_config.yaml \
 --from=80e3-claim-supplier-2 \
 --node=http://localhost:26657 --chain-id=pocket \
 --home=./localnet/pocketd --keyring-backend=test --no-passphrase
```

**And verify it is onchain**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_2} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

You can check its **stake**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_2} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

You can check the unstaked balance transfer of the owner's **unstaked balance**:

```bash
pocketd query bank balance ${ADDR_OWNER} upokt \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.balance.amount'
```

#### [NOT IMPLEMENTED YET] Option 2 - Claim Shannon Supplier WITH an output address - as an operator

**Conditions whereby**:

- **In Morse**: `output_address` != `null` & `operator_address` != null
- **In Shannon**: `owner_address` = `operator_address`
- Claiming as a `operator_address` on behalf of `output_address`

**Prepare a supplier stake config**:

```bash
cat <<EOF > 80e3_claim_supplier_2_supplier_config.yaml
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

**Claim the supplier**:

```bash
pocketd tx migration claim-supplier \
 pocket-account-80e3058d66ee75578b07472650483da0035febe6.json \
 80e3_claim_supplier_2_supplier_config.yaml \
 --from=80e3-claim-supplier-2 \
 --node=http://localhost:26657 --chain-id=pocket \
 --home=./localnet/pocketd --keyring-backend=test --no-passphrase
```

**And verify it is onchain**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_2} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd
```

You can check its **stake**:

```bash
pocketd query supplier show-supplier ${ADDR_SUPPLIER_2} \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.supplier.stake.amount'
```

You can check the unstaked balance transfer of the owner's **unstaked balance**:

```bash
pocketd query bank balance ${ADDR_OWNER} upokt \
  -o json --node=http://127.0.0.1:26657 \
  --home=./localnet/pocketd | jq '.balance.amount'
```
