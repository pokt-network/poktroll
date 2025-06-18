---
title: pocketd CLI Installation
sidebar_position: 1
---

:::tip TL;DR

To install `pocketd` on Linux or MacOS, run:

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
```

To upgrade `pocketd` to the latest version, run:

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --upgrade
```

To upgrade a `pocketd` to a specific release (e.g. `v0.1.12-dev3`), run:

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev3 --upgrade
```

:::

## Table of Contents <!-- omit in toc -->

- [Bash Install Script (Linux \& MacOS)](#bash-install-script-linux--macos)
- [Homebrew (MacOS only)](#homebrew-macos-only)
  - [Troubleshooting Homebrew](#troubleshooting-homebrew)
- [Alternative Methods](#alternative-methods)
  - [Using release binaries](#using-release-binaries)
  - [From Source (danger zone)](#from-source-danger-zone)
    - [Installation dependencies](#installation-dependencies)
    - [Build from source](#build-from-source)
  - [Building Release Binaries From Source](#building-release-binaries-from-source)
- [Windows (why!?)](#windows-why)
- [Publishing a new `pocketd` release](#publishing-a-new-pocketd-release)
  - [1. Create a new `dev` git tag](#1-create-a-new-dev-git-tag)
  - [2. Draft a new GitHub release](#2-draft-a-new-github-release)
  - [3. Wait for the release artifacts to be built (5 - 20 minutes)](#3-wait-for-the-release-artifacts-to-be-built-5---20-minutes)
  - [4. Verify via the `pocketd-install.sh` script](#4-verify-via-the-pocketd-installsh-script)
- [Service Quality Report (Network Agnostic)](#service-quality-report-network-agnostic)
- [Current Traffic Split](#current-traffic-split)
- [Service Availability Report - Main](#service-availability-report---main)
- [Service Availability Report - Beta](#service-availability-report---beta)
- [Commands Executed](#commands-executed)

---

## Bash Install Script (Linux & MacOS)

Easiest, fastest way to get started that works on both Linux and MacOS.

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
```

Verify installation:

```bash
pocketd version
pocketd --help
```

---

## Homebrew (MacOS only)

For MacOS users who prefer [Homebrew](https://brew.sh/).

```bash
brew tap pokt-network/poktroll
brew install pocketd
```

### Troubleshooting Homebrew

If you have problems installing or upgrading `pocketd` via Homebrew:

```bash
brew update
brew upgrade pocketd
```

If it's still not working, try:

```bash
brew tap --repair
brew untap pokt-network/poktroll
brew uninstall pocketd
brew tap pokt-network/poktroll
brew install pocketd
```

The source code for the Homebrew formula can be found at [homebrew-pocketd](https://github.com/pokt-network/homebrew-pocketd).

---

## Alternative Methods

### Using release binaries

:::tip TL;DR manual download

- Download the binary from the [latest release](https://github.com/pokt-network/poktroll/releases/latest)
- Choose the correct `pocket_${OS}_${ARCH}.tar.gz` for your system
- Untar and move the binary to `/usr/local/bin`

:::

```bash
# Download the correct binary for your OS and architecture
curl -LO "https://github.com/pokt-network/poktroll/releases/latest/download/pocket_$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz"

# Extract to /usr/local/bin
sudo tar -zxf "pocket_$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz" -C /usr/local/bin

# Make it executable
sudo chmod +x /usr/local/bin/pocketd

# Check version
pocketd version
```

Additional references and links:

- Pre-built binaries can be found on the [releases page](https://github.com/pokt-network/poktroll/releases)
- Latest release can be found [here](https://github.com/pokt-network/poktroll/releases/latest)

---

### From Source (danger zone)

:::warning
Do not continue unless you're a 🚀👨‍💻💎

For **ADVANCED** users only. Requires developer tools.
:::

#### Installation dependencies

- [Go](https://go.dev/doc/install) (v1.23+)
- [Make](https://www.gnu.org/software/make/)
- [Ignite CLI](https://docs.ignite.com/welcome/install)

#### Build from source

Clone the repository

```bash
git clone https://github.com/pokt-network/poktroll.git pocket
cd pocket
```

And build the dependencies

```bash
make go_develop
```

Then, you have a few options:

1. Use the `make` target helper to use ignite indirecly:

   ```bash
   make ignite_pocketd_build
   ```

2. Use `Ignite` to build the binary directly to the `GOPATH`:

   ```bash
   ignite chain build --skip-proto --debug -v -o $(shell go env GOPATH)/bin
   ```

3. Use `Ignite` to build the binary directly to the current directory:

   ```bash
   ignite chain build --skip-proto --debug -v -o
   ```

When you're done, verify the installation:

```bash
pocketd version
pocketd --help
```

### Building Release Binaries From Source

The official binaries in our [GitHub releases](https://github.com/pokt-network/poktroll/releases)
are built using [this GitHub workflow](https://github.com/pokt-network/poktroll/actions/workflows/release-artifacts.yml).

You can build the release binaries locally for all CPU architectures like so:

```bash
make ignite_release
```

---

## Windows (why!?)

:::danger

- Native Windows installation is **not supported**.
- Use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
- Follow the Linux install instructions above.

:::

---

## Publishing a new `pocketd` release

:::warning Devs only

This section is intended for core protocol developers only.

:::

This section is intended **only for core protocol developers**.

It is also only intended for for dev releases of the `pocketd` CLI.

If you are publishing an official protocol upgrade accompanies by a CLI update, visit the [release procedure docs](../../4_develop//upgrades/2_release_procedure.md).

:::

### 1. Create a new `dev` git tag

```bash
# Clone the repository if you haven't already
git clone git@github.com:pokt-network/poktroll.git poktroll
cd poktroll

# Create a new dev git tag and follow the on-screen instructions
make release_tag_rc

# Push the tag to GitHub
git push origin $(git tag)
```

### 2. Draft a new GitHub release

Draft a new release at [pokt-network/poktroll/releases/new](https://github.com/pokt-network/poktroll/releases/new) using the tag (e.g. `v0.1.12-dev3`) created in the previous step.

Make sure to mark as a `pre-release` and use the auto-generated release notes for simplicity.

### 3. Wait for the release artifacts to be built (5 - 20 minutes)

The [release artifacts workflow](https://github.com/pokt-network/poktroll/actions/workflows/release-artifacts.yml) will automatically build and publish the release artifacts to GitHub.

Wait for the release artifacts to be built and published to GitHub.

The artifacts will be attached an an `Asset` to your [release](https://github.com/pokt-network/poktroll/releases) once the workflow completes.

### 4. Verify via the `pocketd-install.sh` script

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev3 --upgrade
```

## Service Quality Report (Network Agnostic)

```bash
===== SERVICE SUMMARY =====
| Service          | Status | Requests | Successes | Failures             | Success Rate | P90 Latency | Avg Latency |
|------------------|--------|----------|-----------|----------------------|--------------|-------------|-------------|
| bsc              | 🟡      | 270      | 202       | 68 (max allowed: 13) | 74.81%       | 305ms       | 252ms       |
| eth              | 🟡      | 270      | 247       | 23 (max allowed: 13) | 91.48%       | 345ms       | 267ms       |
| gnosis           | 🟢      | 270      | 270       | 0                    | 100.00%      | 267ms       | 215ms       |
| poly             | 🟡      | 270      | 156       | 114 (max allowed: 13) | 57.78%       | 343ms       | 275ms       |
| xrpl_evm_test    | 🟢      | 270      | 270       | 0                    | 100.00%      | 133ms       | 103ms       |

===== END SERVICE SUMMARY =====
```

## Current Traffic Split

```bash
| Service                     | Beta (%) | Main (%) |
|-----------------------------|----------|----------|
| Ethereum                    | 50       | 50       |
| Pocket Network              | 80       | 20       |
| Base                        | 80       | 20       |
| BSC                         | 80       | 20       |
| Ethereum Holesky Testnet    | 80       | 20       |
| Ethereum Sepolia Testnet    | 80       | 20       |
| Gnosis                      | 80       | 20       |
| Optimism                    | 80       | 20       |
```

## Service Availability Report - Main

```bash
SERVICE ID           | SUPPLIERS | RESULT  | TLDS
---------------------+-----------+---------+-----------------
eth                  | 1455      | 🟢    | example.com,kalorius.tech,nodefleet.net,qspider.com,spacebelt.cloud,stakenodes.org
gnosis               | 510       | 🟢    | kalorius.tech,nodefleet.net,qspider.com,stakenodes.org
pocket               | 42        | 🟢    | kalorius.tech
poly                 | 474       | 🟢    | nodefleet.net,stakenodes.org
bsc                  | 505       | 🟡    | nodefleet.net,stakenodes.org
xrpl_evm_test        | 32        | 💔    | stakenodes.org
```

## Service Availability Report - Beta

```bash
SERVICE ID           | SUPPLIERS | RESULT  | TLDS
---------------------+-----------+---------+-----------------
bsc                  | 177       | 🟢    | dopokt.com,easy2stake.com:443,nodefleet.net,spacebelt.cloud
eth                  | 405       | 🟢    | dopokt.com,dopokt.com:443,easy2stake.com:443,kalorius.tech,nodefleet.net,spacebelt.cloud
pocket               | 35        | 🟢    | 223.133:26657,kalorius.tech
poly                 | 359       | 🟢    | dopokt.com,easy2stake.com:443,nodefleet.net,spacebelt.cloud
gnosis               | 200       | 🟡    | dopokt.com,kalorius.tech,nodefleet.net
xrpl_evm_test        | 1         | 💔    | poktroll.com:443
==============================
```

## Commands Executed

Load Test:

```bash
make load_test bsc,eth,pocket,poly,gnosis,xrpl_evm_test
```

Service availability on Main:

```bash
./e2e/scripts/shannon_preliminary_services_test.sh \
        --network main \
        --environment production \
        --portal_app_id ea7f9165 \
        --api_key ad61cfb38c1e79ab0ba58f9a1b5968f9 \
        --services bsc,eth,pocket,poly,gnosis,xrpl_evm_test
```

Service availability on Beta:

```bash
./e2e/scripts/shannon_preliminary_services_test.sh \
        --network beta \
        --environment production \
        --portal_app_id ea7f9165 \
        --api_key ad61cfb38c1e79ab0ba58f9a1b5968f9 \
        --services bsc,eth,pocket,poly,gnosis,xrpl_evm_test
```
