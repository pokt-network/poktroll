# Relay Spam Tool

A comprehensive tool for stress testing Pocket Network with relay requests.

## Overview

The Relay Spam tool is designed to test Pocket Network's relay capabilities by generating high volumes of relay requests from multiple accounts. It provides functionality for:

- Creating and managing accounts
- Funding accounts
- Staking applications, services, and suppliers (SUPPLIERS NOT TESTED - THEY WERE STAKED MANUALLY)
- Delegating applications to gateways
- Sending relay requests with configurable concurrency and rate limiting
- Collecting and reporting metrics

## Prerequisites

- Go 1.19+
- A running Pocket Network node
- Access to the node's RPC (default: http://localhost:26657) and GRPC (default: localhost:9090) endpoints

## Installation

Clone the repository and build the tool:

```bash
git clone https://github.com/pokt-network/poktroll.git
cd poktroll/tools/relay-spam

# Build the tool
go build -o relay-spam .
```

Alternatively, use the provided helper script:
```bash
chmod +x run-relay-spam.sh
./run-relay-spam.sh
```

## Quick Start

### 1. Create Configuration

Copy and edit the example configuration:
```bash
cp config.yml.example config.yml
```

Edit `config.yml` to set appropriate values for:
- Stake and fund amounts
- RPC and GRPC endpoints
- Gateway URLs

### 2. Create Accounts

Generate new accounts:
```bash
./relay-spam populate --num-accounts 5
```

This adds accounts to your config.yml file.

### 3. Fund Accounts

Create funding transactions:
```bash
./relay-spam fund
```

This outputs commands to fund your accounts. Run these commands in a terminal with access to a funded account.

### 4. Import Accounts

Import the accounts into your keyring:
```bash
./relay-spam import
```

### 5. Stake Applications

Stake your applications and delegate to gateways:
```bash
./relay-spam stake application --delegate
```

### 6. Run Relay Tests

Send relay requests:
```bash
./relay-spam run --num-requests 100 --concurrency 20 --rate-limit 50
```

## Configuration Details

The configuration file includes:

- Data directory for Pocket Network
- Transaction flags
- Chain ID for transactions
- RPC and GRPC endpoints
- Application, service, and supplier configuration
- Stake amounts for each entity type
- Gateway URLs mapping (maps gateway IDs to their URLs for relay requests)

### Example Configuration

```yaml
# Data directory for Pocket Network
datadir: ~/.poktroll

# Stake goals for different entity types
application_stake_goal: 1000000upokt
supplier_stake_goal: 1000000upokt

# Funding goals for entities
application_fund_goal: 2000000upokt

# Chain ID for transactions
chain_id: poktroll

# GRPC endpoint for querying balances
grpc_endpoint: localhost:9090

# RPC endpoint for broadcasting transactions
rpc_endpoint: http://localhost:26657

# Map of gateway IDs to their URLs
gateway_urls:
  pokt1tgfhrtpxa4afeh70fk2aj6ca4mw84xqrkfgrdl: http://localhost:8081
  pokt15vzxjqklzjtlz7lahe8z2dfe9nm5vxwwmscne4: http://anvil.localhost:3000/v1

txflagstemplate:
  chain-id: poktroll
  gas: auto
  gas-adjustment: 1.5
  gas-prices: 0.01upokt
  broadcast-mode: sync
  yes: true
  keyring-backend: test

# Applications configuration
applications:
  - name: relay_spam_app_0
    address: pokt18wmctmhu49csyy6j0eyhmua63rvlwgc8hddg2c
    mnemonic: certain monitor elephant guard must vacant magnet present bacon scare social cattle enact average stairs orient disorder whisper frame banner version open spray brother
    serviceidgoal: anvil
    delegateesgoal:
      - pokt1tgfhrtpxa4afeh70fk2aj6ca4mw84xqrkfgrdl

# Services configuration
services:
  - name: anvil_service
    address: pokt1abcdefghijklmnopqrstuvwxyz0123456789abcdef
    mnemonic: example mnemonic for service account goes here
    serviceid: anvil

# Suppliers configuration
suppliers:
  - name: anvil_supplier
    address: pokt1fedcba9876543210abcdefghijklmnopqrstuvwxyz
    mnemonic: example mnemonic for supplier account goes here
    services:
      - anvil
```

## Detailed Usage

### Creating Accounts

```bash
./relay-spam populate --num-accounts 5
```

This creates new accounts and adds them to the configuration file.

### Importing Accounts

```bash
./relay-spam import
```

This imports accounts from the configuration file into the keyring.

### Funding Accounts

```bash
./relay-spam fund
```

This generates commands to fund accounts in the configuration file.

### Staking Entities

Staking Applications:
```bash
./relay-spam stake application
```

Staking Services:
```bash
./relay-spam stake service
```

Staking Suppliers:
```bash
./relay-spam stake supplier
```

Use the `--dry-run` flag to preview transactions without sending them:
```bash
./relay-spam stake application --dry-run
```

### Delegating Applications to Gateways

```bash
./relay-spam stake application --delegate
```

This stakes applications and delegates them to gateways.

### Running Relay Spam

```bash
./relay-spam run --num-requests 100 --concurrency 20 --rate-limit 50
```

This runs relay spam with the configured applications and gateways.

## Command Line Options

- `--config, -c`: Config file (default: config.yml)
- `--num-requests, -n`: Number of requests per application-gateway pair (default: 10)
- `--concurrency, -p`: Concurrent requests (default: 10)
- `--num-accounts, -a`: Number of accounts to create (default: 10)
- `--rate-limit, -r`: Rate limit in requests per second (0 for no limit) (default: 0)
- `--keyring-backend`: Keyring backend to use (os, file, test, inmemory) (default: test)
- `--dry-run`: Show transactions without sending them (for stake commands)

## Workflow Examples

### Application Workflow

1. Create configuration
2. Create accounts
3. Fund accounts
4. Stake applications and delegate to gateways
5. Run relay spam

### Service and Supplier Workflow

1. Add service and supplier configurations to `config.yml`
2. Fund the service and supplier accounts
3. Stake services
4. Stake suppliers with services

## Metrics

After running relay spam, the tool outputs metrics including:

- Total requests
- Successful requests
- Failed requests
- Duration
- Requests per second

## Troubleshooting

### Common Issues

1. **Keyring Access Issues**: If you encounter problems with the keyring, try changing the keyring backend:
   ```bash
   ./relay-spam import --keyring-backend test
   ```

2. **Transaction Failures**: Ensure your accounts have sufficient funds and check that your RPC endpoint is correct.

3. **Connection Errors**: Verify that your Pocket Network node is running and that the RPC and GRPC endpoints are accessible.

## Code Structure

The tool is organized into modular components for maintainability and extensibility:

```
tools/relay-spam/
├── account/          # Account-related functionality
├── application/      # Application-related functionality
├── cmd/              # Command-line commands
├── config/           # Configuration loading and parsing
├── data/             # Directory for data storage
├── metrics/          # Metrics collection and reporting
├── relay/            # Relay request functionality
├── service/          # Service-related functionality
├── supplier/         # Supplier-related functionality
├── util/             # Shared utility functions
├── config.yml        # User configuration file
├── main.go           # Entry point
└── run-relay-spam.sh # Helper script
```

### Key Components

- **main.go**: Entry point that initializes the SDK configuration and executes the command line interface
- **cmd/**: Contains all command implementations for creating accounts, funding, staking, and running tests
- **application/**, **service/**, **supplier/**: Entity-specific operations
- **relay/**: Contains relay request logic (formatting, sending, response handling)
- **metrics/**: Collects and reports performance metrics

## Extending the Tool

To add new functionality:
1. Create a new command in the `cmd/` directory
2. Register the command in `cmd/root.go`
3. Implement the business logic in the appropriate module directory

## Dependencies

The Relay Spam tool requires:
- A running Pocket Network node
- Properly funded accounts for staking and transactions
- Access to the configured RPC and GRPC endpoints 