---
title: poktrolld Installation
sidebar_position: 1
---

:::tip TL;DR If you know what you're doing

If you have `brew`:

```bash
brew tap pokt-network/poktroll
brew install poktrolld
```

If you don't have `brew`:

1. Grab a binary from the [latest release](https://github.com/pokt-network/poktroll/releases/latest)
2. Download the appropriate `poktroll_${OS}_${ARCH}.tar.gz` for your environment
3. Untar the downloaded file to retrieve the `poktrolld` binary
4. Extract the binary to `/usr/local/bin`

or grab a binary from the [releases page](https://github.com/pokt-network/poktroll/releases).

:::

## Table of Contents <!-- omit in toc -->

- [MacOS \& Linux Users](#macos--linux-users)
  - [Using Homebrew](#using-homebrew)
  - [Using release binaries](#using-release-binaries)
  - [From Source](#from-source)
    - [Installing dependencies](#installing-dependencies)
    - [Installing poktrolld](#installing-poktrolld)
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

<details>
<summary>
<h3>Troubleshooting Homebrew</h3>
<p>
Read this section if you're having problems downloading or upgrading your `poktrolld` binary using Homebrew.
</p>
</summary>

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

</details>

### Using release binaries

Pre-built binaries are available on our [releases page](https://github.com/pokt-network/poktroll/releases).

You can view the latest release directly by clicking [here](https://github.com/pokt-network/poktroll/releases/latest).

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

### From Source

:::warning Do not continue unless you're a 🚀👨‍💻💎

This method is only recommended for **ADVANCED** users as it requires working with developer tools.

:::

#### Installing dependencies

Ensure you have the following installed:

- [Go](https://go.dev/doc/install) (version 1.23 or later)
  - Make sure to add `export PATH=$PATH:$(go env GOPATH)/bin/` to your `.bashrc` or `.zshrc` file.
- [Ignite CLI](https://docs.ignite.com/welcome/install)

#### Installing poktrolld

Then, Retrieve the source code and build the `poktrolld` locally like so:

```bash
# Clone the repository
git clone https://github.com/pokt-network/poktroll.git
cd poktroll

# Optional: Switch to a specific version (recommended)
# Replace v0.0.12 with your desired version from https://github.com/pokt-network/poktroll/releases
git checkout v0.0.12

# Build the binary
make go_develop
make ignite_poktrolld_build
```

And verify it worked by running:

```bash
poktrolld version
poktrolld --help
```

## Windows Users

:::danger

Why? 🥴 ⁉️

:::

Currently, we do not support native Windows installation. Windows users are encouraged
to use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
and follow the Linux installation instructions.
