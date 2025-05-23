---
title: pocketd CLI Installation
sidebar_position: 1
---

:::tip TL;DR
To install `pocketd` on Linux or MacOS, run:

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/scripts/install.sh | bash
```

:::

## Table of Contents <!-- omit in toc -->

- [1. Install Script (Linux \& MacOS)](#1-install-script-linux--macos)
- [2. Homebrew (MacOS only)](#2-homebrew-macos-only)
  - [Troubleshooting Homebrew](#troubleshooting-homebrew)
- [3. Alternative Methods](#3-alternative-methods)
  - [Using release binaries](#using-release-binaries)
  - [From Source (danger zone)](#from-source-danger-zone)
    - [Installation dependencies](#installation-dependencies)
    - [Build from source](#build-from-source)
- [4. Windows (why!?)](#4-windows-why)

---

## 1. Install Script (Linux & MacOS)

Easiest, fastest way to get started that works on both Linux and MacOS.

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/scripts/install.sh | bash
```

Verify installation:

```bash
pocketd version
pocketd --help
```

---

## 2. Homebrew (MacOS only)

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

## 3. Alternative Methods

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

```bash
# Clone the repository
git clone https://github.com/pokt-network/poktroll.git pocket
cd pocket

# Optional: Checkout a specific version (recommended)
# Replace v0.0.12 with your desired version from https://github.com/pokt-network/poktroll/releases
git checkout v0.0.12

# Build the binary
make go_develop
make ignite_pocketd_build
```

**Verify installation:**

```bash
pocketd version
pocketd --help
```

---

## 4. Windows (why!?)

:::danger

- Native Windows installation is **not supported**.
- Use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
- Follow the Linux install instructions above.
  :::

---
