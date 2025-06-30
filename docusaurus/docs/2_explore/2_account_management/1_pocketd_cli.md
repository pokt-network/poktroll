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
  - [1. Create a new `dev` or `rc` git tag](#1-create-a-new-dev-or-rc-git-tag)
  - [2. Draft a new GitHub release](#2-draft-a-new-github-release)
  - [3. Wait for the release artifacts to be built (5 - 20 minutes)](#3-wait-for-the-release-artifacts-to-be-built-5---20-minutes)
  - [4. Verify via the `pocketd-install.sh` script](#4-verify-via-the-pocketd-installsh-script)

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

### 1. Create a new `dev` or `rc` git tag

```bash
# Clone the repository if you haven't already
git clone git@github.com:pokt-network/poktroll.git poktroll
cd poktroll

# Create a new rc tag from `main` or `master` and follow the on-screen instructions
make release_tag_rc
# OR
# Create a new dev tag from any branch and follow the on-screen instructions
make release_tag_dev

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
