# Submit upgrade
poktrolld tx authz exec tools/scripts/upgrades/authz_upgrade_tx.json --from pnf --yes

sleep 3

poktrolld query upgrade plan

# If changing consensus module parameters (such as block size), execute after an upgrade to verify the block size
# has been changed.
# 
# params:
#   abci: {}
#   block:
#     max_bytes: "22020096"
# 
# Should be increased to:
# 
# params:
#   abci: {}
#   block:
#     max_bytes: "44040192"
#     max_gas: "-1"

poktrolld query consensus params