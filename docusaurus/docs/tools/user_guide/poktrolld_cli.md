---
title: poktrolld Installation
sidebar_position: 1
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
  - [Troubleshooting Homebrew](#troubleshooting-homebrew)
  - [From Source](#from-source)
    - [Installing dependencies](#installing-dependencies)
    - [Installing poktrolld](#installing-poktrolld)
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

### Troubleshooting Homebrew

:::info Having issues with Homebrew?

Read this section if you're having problems downloading or upgrading your `poktrolld` binary using Homebrew.

:::

The source code for the Homebrew formula is available in the [homebrew-poktroll](https://github.com/pokt-network/homebrew-poktroll) repository.

If you encounter any issues, like being unable to install the latest version, you can try the following:

```bash
brew update
brew upgrade poktrolld
```

Or as a last resort, you can try the following:

```bash
brew tap --repair
brew untap pokt-network/poktroll
brew uninstall poktrolld
brew tap pokt-network/poktroll
brew install poktrolld
```

### From Source

#### Installing dependencies

Ensure you have the following installed:

- [Go](https://go.dev/doc/install) (version 1.18 or later)
- [Ignite CLI](https://docs.ignite.com/welcome/install)

If you're on a Linux machine, you can follow the steps below for convenience:

```bash
# Install go 1.23
curl -o ./pkgx --compressed -f --proto '=https' https://pkgx.sh/$(uname)/$(uname -m)
sudo install -m 755 pkgx /usr/local/bin
pkgx install go@1.23.0
export PATH=$PATH:$HOME/go/bin/

# Install PATH Gateway required dependencies
apt-get update && apt-get install git make build-essential

# Install the ignite binary used to build the Pocket binary
curl https://get.ignite.com/cli! | bash
```

#### Installing poktrolld

Then, Retrieve the source code and build the `poktrolld` locally like so:

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
