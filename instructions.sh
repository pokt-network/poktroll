# pocket accounts export $MORSE_ADDR_SUPPLIER_1 --datadir ./morse_pocket_datadir
# pocket accounts export $MORSE_ADDR_SUPPLIER_2 --datadir ./morse_pocket_datadir
# pocket accounts export $MORSE_ADDR_SUPPLIER_3 --datadir ./morse_pocket_datadir
# pocket accounts export $MORSE_ADDR_OWNER_1 --datadir ./morse_pocket_datadir
# pocket accounts export $MORSE_ADDR_OWNER_2 --datadir ./morse_pocket_datadir

# pocket accounts show $MORSE_ADDR_PNF --datadir ./morse_pocket_datadir
# pocket accounts show $MORSE_ADDR_SUPPLIER_1 --datadir ./morse_pocket_datadir
# pocket accounts show $MORSE_ADDR_SUPPLIER_2 --datadir ./morse_pocket_datadir
# pocket accounts show $MORSE_ADDR_SUPPLIER_3 --datadir ./morse_pocket_datadir
# pocket accounts show $MORSE_ADDR_OWNER_1 --datadir ./morse_pocket_datadir
# pocket accounts show $MORSE_ADDR_OWNER_2 --datadir ./morse_pocket_datadir

MORSE_ADDR_PNF="16f6327acdd442f35cb4501d77174bb55eb90969"
MORSE_ADDR_SUPPLIER_1="2c3a78c6ddf74eb836c245d6b68cf368e0f3c2c7"
MORSE_ADDR_SUPPLIER_2="638e179624fac27bdcea0a9436c5db2372cb4ae0"
MORSE_ADDR_SUPPLIER_3="6f27fbe6849bb90a8dfd8966dc01f655d384f202"
MORSE_ADDR_OWNER_1="df0963bf1ac9c8f5c42b47a40688fee6e20903b7"
MORSE_ADDR_OWNER_2="ef4588ea42dcddaab6b68f25bd436c6333855501"

MORSE_PNF_PUBKEY="25fb32fb5172d85cadc568083116683ccedc7e3e8ab741cfe8127c2f98b18f7b"
MORSE_SUPPLIER_PUBKEY_1="4320e27c845e2bc580794854d2a8067c15e4c7e3a7b54e1d362396950df43aae"
MORSE_SUPPLIER_PUBKEY_2="7e8081907d8b201ae1ec7983bdc0a8063bb19e26ef6bb29bb0bbecd43059abfa"
MORSE_SUPPLIER_PUBKEY_3="fac1eb4019dc7dc5b74250fc5e92db805491b743fef431c034a2e692e961f001"
MORSE_OWNER_PUBKEY_1="a00f6e28e3b6847e504974758f9a3174ee1c64e107cdc72622c4a533decae6c0"
MORSE_OWNER_PUBKEY_2="dd2ec5ba5d5352cc19e6140f511a9e03d85140c0f2c79ba99ec4f7e384c6c3af"

MORSE_SUPPLIER_1_PREFIX=${MORSE_ADDR_SUPPLIER_1:0:4}
MORSE_SUPPLIER_2_PREFIX=${MORSE_ADDR_SUPPLIER_2:0:4}
MORSE_SUPPLIER_3_PREFIX=${MORSE_ADDR_SUPPLIER_3:0:4}
MORSE_OWNER_1_PREFIX=${MORSE_ADDR_OWNER_1:0:4}
MORSE_OWNER_2_PREFIX=${MORSE_ADDR_OWNER_2:0:4}

SHANNON_ADDR_SUPPLIER_1=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_1_PREFIX}-claim-supplier-1 -a)
SHANNON_ADDR_SUPPLIER_2=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2 -a)
SHANNON_ADDR_SUPPLIER_3=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_SUPPLIER_3_PREFIX}-claim-supplier-3 -a)
SHANNON_ADDR_OWNER_1=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_OWNER_1_PREFIX}-claim-owner-1 -a)
SHANNON_ADDR_OWNER_2=$(pocketd --keyring-backend=test --home=./localnet/pocketd keys show ${MORSE_OWNER_2_PREFIX}-claim-owner-2 -a)

pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_1 1mact --home=./localnet/pocketd --yes --unordered --timeout-duration=5s --fees=1upokt
pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_2 1mact --home=./localnet/pocketd --yes --unordered --timeout-duration=5s --fees=1upokt
pocketd tx bank send pnf $SHANNON_ADDR_SUPPLIER_3 1mact --home=./localnet/pocketd --yes --unordered --timeout-duration=5s --fees=1upokt
pocketd tx bank send pnf $SHANNON_ADDR_OWNER_1 1mact --home=./localnet/pocketd --yes --unordered --timeout-duration=5s --fees=1upokt
pocketd tx bank send pnf $SHANNON_ADDR_OWNER_2 1mact --home=./localnet/pocketd --yes --unordered --timeout-duration=5s --fees=1upokt

# pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_1_PREFIX}-claim-supplier-1
# pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2
# pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_SUPPLIER_3_PREFIX}-claim-supplier-3
# pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_OWNER_1_PREFIX}-claim-owner-1
# pocketd --keyring-backend=test --home=./localnet/pocketd keys add ${MORSE_OWNER_2_PREFIX}-claim-owner-2

pocketd tx migration collect-morse-accounts \
    localnet_testing_state_export.json localnet_testing_msg_import_morse_accounts.json \
    --home=./localnet/pocketd

pocketd tx migration import-morse-accounts \
    localnet_testing_msg_import_morse_accounts.json \
    --from=pnf \
    --home=./localnet/pocketd --keyring-backend=test \
    --network=local \
    --gas=auto --gas-adjustment=1.5 --gas-prices=0.000000001upokt

pocketd tx migration claim-account \
    pocket-account-${MORSE_ADDR_OWNER_2}.json \
    --from=${MORSE_OWNER_2_PREFIX}-claim-owner-2 \
    --network=local \
    --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
    --gas=auto --gas-adjustment=1.5 --yes --gas-prices=100000000000000000000000000upokt

pocketd tx migration claim-supplier \
    ${MORSE_ADDR_SUPPLIER_2} pocket-account-${MORSE_ADDR_SUPPLIER_2}.json \
    ${MORSE_SUPPLIER_2_PREFIX}_claim_supplier_2_supplier_config.yaml \
    --from=${MORSE_SUPPLIER_2_PREFIX}-claim-supplier-2 \
    --network=local \
    --home=./localnet/pocketd --keyring-backend=test --no-passphrase \
    --gas=auto --gas-adjustment=1.5 --yes --gas-prices=100000000000000000000000000upokt

pocketd query supplier show-supplier $SHANNON_ADDR_SUPPLIER_2 -o json --network=local --home=./localnet/pocketd

pocketd query bank balance $SHANNON_ADDR_SUPPLIER_2 upokt -o json --network=local --home=./localnet/pocketd | jq '.balance.amount'
