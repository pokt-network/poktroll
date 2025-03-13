# Relay Spam Tool

A comprehensive tool for stress testing Pocket Network with relay requests.

## Overview

The Relay Spam tool is designed to test Pocket Network's relay capabilities by generating high volumes of relay requests from multiple accounts. It provides functionality for:

- Creating and managing accounts
- Funding accounts
- Staking applications
- Delegating applications to gateways
- Sending relay requests with configurable concurrency and rate limiting
- Collecting and reporting metrics

## Installation

```bash
# Clone the repository
git clone https://github.com/pokt-network/poktroll.git
cd poktroll

# Build the tool
go build -o relay-spam ./tools/relay-spam
```

## Configuration

Copy the example configuration file and modify it as needed:

```bash
cp tools/relay-spam/config.yml.example config.yml
```

The configuration file includes:

- Home directory for Pocket Network
- Transaction flags
- Chain ID for transactions
- Application defaults (stake amount, service ID, gateways)
- List of applications
- Gateway URLs mapping (maps gateway IDs to their URLs for relay requests)

## Configuration Example

```yaml
# Chain ID for transactions
chain_id: poktroll

# Map of gateway IDs to their URLs
gateway_urls:
  pokt1tgfhrtpxa4afeh70fk2aj6ca4mw84xqrkfgrdl: http://localhost:8081
  pokt15vzxjqklzjtlz7lahe8z2dfe9nm5vxwwmscne4: http://anvil.localhost:3000/v1

# Applications configuration
applications:
  - name: relay_spam_app_0
    address: pokt18wmctmhu49csyy6j0eyhmua63rvlwgc8hddg2c
    mnemonic: certain monitor elephant guard must vacant magnet present bacon scare social cattle enact average stairs orient disorder whisper frame banner version open spray brother
    serviceidgoal: svc1qjpxsjkz0kujcvdlxm2wkjv5m4g0p9k
    delegateesgoal:
      - pokt1tgfhrtpxa4afeh70fk2aj6ca4mw84xqrkfgrdl
```

## Usage

### Creating Accounts

```bash
./relay-spam populate --num-accounts 5
```

This command creates new accounts and adds them to the configuration file.

### Importing Accounts

```bash
./relay-spam import
```

This command imports accounts from the configuration file into the keyring.

### Funding Accounts

```bash
./relay-spam fund
```

This command generates commands to fund accounts in the configuration file.

### Staking Applications

```bash
./relay-spam stake
```

This command stakes applications in the configuration file.

### Delegating Applications to Gateways

```bash
./relay-spam stake --delegate
```

This command stakes applications and delegates them to gateways.

### Running Relay Spam

```bash
./relay-spam run --num-requests 100 --concurrency 20 --rate-limit 50
```

This command runs relay spam with the configured applications and gateways.

## Command Line Options

- `--config, -c`: Config file (default: config.yml)
- `--num-requests, -n`: Number of requests per application-gateway pair (default: 10)
- `--concurrency, -p`: Concurrent requests (default: 10)
- `--num-accounts, -a`: Number of accounts to create (default: 10)
- `--rate-limit, -r`: Rate limit in requests per second (0 for no limit) (default: 0)

## Example Workflow

1. Create a configuration file:
   ```bash
   cp tools/relay-spam/config.yml.example config.yml
   ```

2. Create accounts:
   ```bash
   ./relay-spam populate --num-accounts 5
   ```

3. Fund accounts (run the commands output by the previous step)

4. Stake applications and delegate to gateways:
   ```bash
   ./relay-spam stake --delegate
   ```

5. Run relay spam:
   ```bash
   ./relay-spam run --num-requests 100 --concurrency 20 --rate-limit 50
   ```

## Metrics

After running relay spam, the tool will output metrics including:

- Total requests
- Successful requests
- Failed requests
- Duration
- Requests per second 