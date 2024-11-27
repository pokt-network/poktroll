---
title: CLI Installation
sidebar_position: 0
---

:::warning

tl;dr IFF you know what you're doing.

```bash
brew tap pokt-network/poktroll
brew install poktrolld
```

:::

- [MacOS \& Linux Users](#macos--linux-users)
  - [Using Homebrew](#using-homebrew)
  - [From Source](#from-source)
  - [Using release binaries](#using-release-binaries)
- [Windows Users](#windows-users)

## MacOS & Linux Users

### Using Homebrew

Ensure you have [Homebrew](https://brew.sh/) installed.

Then run the following commands:

```bash
brew tap pokt-network/poktroll
brew install poktrolld
```

And verify it worked by running:

```bash
poktrolld version
poktrolld --help
```

:::tip
See the [homebrew-poktroll](https://github.com/pokt-network/homebrew-poktroll/)
repository for details on how to install homebrew or other details to install
or debug the CLI.
:::

### From Source

Ensure you have the following installed:

- [Go](https://go.dev/doc/install) (version 1.18 or later)
- [Ignite CLI](https://docs.ignite.com/welcome/install)

Then run the following commands:

```bash
git clone https://github.com/pokt-network/poktroll.git
cd poktroll
make go_develop
make ignite_poktrolld_build
```

And verify it worked by running:

```bash
poktrolld version
poktrolld --help
```

### Using release binaries

Pre-built binaries are available on our [releases page](https://github.com/pokt-network/poktroll/releases).

The following snippet downloads/upgrades the binary to the latest released version:

```bash
# Download the correct binary based on the OS and architecture
curl -LO "https://github.com/pokt-network/poktroll/releases/latest/download/poktroll_$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz"

# Extract the downloaded tarball to /usr/local/bin
sudo tar -zxf "poktroll_$(uname | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz" -C /usr/local/bin

# Make the binary executable
sudo chmod +x /usr/local/bin/poktrolld

# Check version
poktrolld version
```

## Windows Users

Currently, we do not support native Windows installation. Windows users are encouraged
to use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
and follow the Linux installation instructions.
