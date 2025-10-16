---
sidebar_position: 2
title: Full Node FAQ
---

## How do I check whether my node is accessible from another machine?

```bash
nc -vz {EXTERNAL_IP} 26656
```

## How do I view my node status?

```bash
sudo systemctl status cosmovisor.service
```

## How do I view my node logs?

```bash
sudo journalctl -u cosmovisor.service -f
```

## How do I stop my node?

```bash
sudo systemctl stop cosmovisor.service
```

## How do I start my node?

```bash
sudo systemctl start cosmovisor.service
```

## How do I restart my node?

```bash
sudo systemctl restart cosmovisor.service
```

## How do I query the latest block (i.e. check the node height)?

Using pocketd:

```bash
pocketd query block --type=height --network=local
```

Or, using curl:

```bash
curl -X GET http://localhost:26657/block | jq
```

## How do I access my CometBFT endpoint externally?

The default CometBFT port is at `26657`.

To make it accessible externally, you'll need to port all the instructions from
port `26656` on this page to port `26657`. Specifically:

```bash
# Update your firewall
sudo ufw allow 26657/tcp

# Alternatively, if ufw is not available, update your iptables
sudo iptables -A INPUT -p tcp --dport 26657 -j ACCEPT

# Update your Cosmovisor config
sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' $HOME/.pocket/config/config.toml
sed -i 's|cors_allowed_origins = \[\]|cors_allowed_origins = ["*"]|' $HOME/.pocket/config/config.toml

# Restart the service
sudo systemctl restart cosmovisor.service

# Test the connection
nc -vz {EXTERNAL_IP} 26657
```

Learn more [here](https://docs.cometbft.com/main/rpc/).

:::warning

Be careful about making this public as adversarial actors may try to DDoS your node.

:::

## How do I check the node version?

```bash
pocketd version
```

## How do I check the Cosmosvisor directory structure?

```bash
ls -la /home/pocket/.pocket/cosmovisor/
```

## How do I check if an upgrade is available?

```bash
ls -la /home/pocket/.pocket/cosmovisor/upgrades/
```

## How do I view node configuration?

```bash
cat /home/pocket/.pocket/config/config.toml
```

## How do I check the version of a Full Node?

```bash
curl -s ${POCKET_RPC_ENDPONT}/abci_info | jq '.result.response.version'
```

For example, for Beta TestNet, we got the RPC endpoint from [here](../../2_explore/1_tools/2_shannon_beta.md) and ran:

```bash
curl -s https://shannon-testnet-grove-rpc.beta.poktroll.com/abci_info | jq '.result.response.version'
```

## How do I check for applied upgrades?

You can query on-chain upgrades using:

```bash
pocketd query upgrade applied v0.1.1
```

This will return the height of the upgrade `v0.1.1`.

## How do I find information about protocol upgrades and releases?

Protocol upgrades are tracked at [dev.poktroll.com/develop/upgrades/upgrade_list](https://dev.poktroll.com/develop/upgrades/upgrade_list#beta-testnet-protocol-upgrades).

The source of truth for releases is the [GitHub releases page](https://github.com/pokt-network/poktroll/releases/tag/v0.1.11).

## Should I use Grove's public RPC endpoint?

While the [beta RPC endpoint](https://shannon-testnet-grove-rpc.beta.poktroll.com) is available, it's recommended to set up your own RPC endpoint.

This is especially true if you're running a Supplier or Indexer, as there's no guarantee the public endpoints can handle all community traffic.

## What should I know about early Beta TestNet blocks?

- Early blocks on Beta TestNet may experience non-scalable validation times (e.g., 5 hours to validate) due to load tests
- This issue will be addressed in a future state shift
- Recommended workaround: Use a snapshot or wait for validation to complete

## Where can I find deployment solutions?

- [Full Node Cheatsheet](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet)
- [Community deployment solutions](https://dev.poktroll.com/explore/community/community)
- [Cosmovisor](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet#install--run-full-node-cosmovisor) for automated upgrades
