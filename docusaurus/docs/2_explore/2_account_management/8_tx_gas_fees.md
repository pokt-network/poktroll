---
title: Transaction Gas Fees
sidebar_position: 8
---

## Quick Start (TL;DR) <!-- omit in toc -->

```bash
pocketd [...] --gas=auto --fees=1upokt
```

This auto-estimates gas and uses a simple fixed fee that should work for most common transactions.

---

## Table of Contents

- [Table of Contents](#table-of-contents)
- [⚙️ Network Configuration](#️-network-configuration)
- [📊 Examples](#-examples)
  - [Standard Send (Recommended)](#standard-send-recommended)
  - [Conservative Send (Higher Fee)](#conservative-send-higher-fee)
  - [Gas Price Method Send](#gas-price-method-send)
  - [Manual Gas Send](#manual-gas-send)
- [Cosmos SDK Introduction](#cosmos-sdk-introduction)
- [Transaction Fee Calculation](#transaction-fee-calculation)
  - [Recommended Fee Configuration](#recommended-fee-configuration)
- [CLI Flag Reference](#cli-flag-reference)
  - [Gas \& Fee Flags](#gas--fee-flags)
  - [Valid Combinations](#valid-combinations)

---

## ⚙️ Network Configuration

[Pocket Network Suggested Validator Configuration (app.toml)](https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/app.toml)

- **Minimum gas price:** `0.000001upokt`
- **Mempool max transactions:** `100,000`
- **RPC timeout:** `60 seconds`

## 📊 Examples

All examples use the `send` transaction with different fee configurations:

### Standard Send (Recommended)

```bash
pocketd tx bank send [from] [to] 1000000upokt \
  --gas=auto \
  --fees=1upokt \
  --network=main
```

### Conservative Send (Higher Fee)

```bash
pocketd tx bank send [from] [to] 1000000upokt \
  --gas=auto \
  --fees=2upokt \
  --network=main
```

### Gas Price Method Send

```bash
pocketd tx bank send [from] [to] 1000000upokt \
  --gas=auto \
  --gas-prices=0.000001upokt \
  --gas-adjustment=1.2 \
  --network=main
```

### Manual Gas Send

```bash
pocketd tx bank send [from] [to] 1000000upokt \
  --gas=500000 \
  --fees=1upokt \
  --network=main
```

## Cosmos SDK Introduction

**POKT Network inherits all transaction fee mechanics from the Cosmos SDK.** Learn more in their [official docs](https://docs.cosmos.network/main/learn/beginner/gas-fees).

<details>

<summary>Quick Cosmos SDK Introduction</summary>

- Gas and fee calculation works exactly like other Cosmos chains
- All standard Cosmos SDK CLI flags are supported
- Fee estimation and validation follows Cosmos standards
- Transaction structure and error handling is identical

The main difference is POKT's specific network configuration (minimum gas prices, mempool settings, etc.).

</details>

## Transaction Fee Calculation

**Formula:** `Total Fee = Gas Limit × Gas Price`
**Network minimum gas price:** `0.000001upokt` (from app.toml)

| Component     | Description                               | Example                        |
| ------------- | ----------------------------------------- | ------------------------------ |
| **Gas Limit** | Maximum gas units the transaction can use | `500000`                       |
| **Gas Price** | Cost per gas unit                         | `0.000001upokt`                |
| **Total Fee** | Final amount charged                      | `500000 × 0.000001 = 0.5upokt` |

### Recommended Fee Configuration

| Scenario                   | Configuration                                                | Use Case                       |
| -------------------------- | ------------------------------------------------------------ | ------------------------------ |
| **Standard (Recommended)** | `--gas=auto --fees=1upokt`                                   | Most transactions              |
| **Conservative**           | `--gas=auto --fees=2upokt`                                   | Critical transactions          |
| **Gas Price Method**       | `--gas=auto --gas-prices=0.000001upokt --gas-adjustment=1.2` | When you want price-based fees |

<details>

<summary>Failed Transaction Types</summary>

Common failure categories to monitor:

- **Insufficient gas**: Gas limit too low
- **Insufficient fees**: Below minimum gas price
- **Invalid sequence**: Nonce/sequence number issues
- **Account errors**: Insufficient balance, invalid account

</details>

## CLI Flag Reference

:::tip `pocketd` CLI Installation

You can install the `pocketd` CLI tool from [here](./1_pocketd_cli.md).

:::

### Gas & Fee Flags

| Flag               | Purpose                                 | Example                        | Notes                         |
| ------------------ | --------------------------------------- | ------------------------------ | ----------------------------- |
| `--gas`            | Set gas limit or auto-estimate          | `--gas=auto` or `--gas=500000` | Use `auto` for estimation     |
| `--gas-prices`     | Price per gas unit                      | `--gas-prices=0.000001upokt`   | Minimum: `0.000001upokt`      |
| `--fees`           | Total fee amount (overrides gas-prices) | `--fees=1upokt`                | Don't use with `--gas-prices` |
| `--gas-adjustment` | Multiplier for auto gas estimation      | `--gas-adjustment=1.2`         | Only with `--gas=auto`        |

### Valid Combinations

✅ **Use ONE of these patterns:**

1. **Auto gas + fixed fee (recommended):**

   ```bash
   --gas=auto --fees=1upokt
   ```

2. **Auto gas + gas prices:**

   ```bash
   --gas=auto --gas-prices=0.000001upokt --gas-adjustment=1.2
   ```

3. **Manual gas + gas prices:**

   ```bash
   --gas=500000 --gas-prices=0.000001upokt
   ```

4. **Manual gas + fixed fee:**
   ```bash
   --gas=500000 --fees=1upokt
   ```

❌ **Don't mix these:**

```bash
--fees=1upokt --gas-prices=0.000001upokt  # Conflicting flags
```
