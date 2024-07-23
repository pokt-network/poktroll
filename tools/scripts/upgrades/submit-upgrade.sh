# Submit upgrade
poktrolld tx authz exec tools/scripts/upgrades/authz_upgrade_tx.json --from pnf --yes

sleep 3

poktrolld query upgrade plan