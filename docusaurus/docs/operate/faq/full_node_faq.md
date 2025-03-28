---
sidebar_position: 1
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
pocketd query block --type=height --node http://localhost:26657
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
