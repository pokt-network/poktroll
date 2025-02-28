---
title: poktrolld Installation
sidebar_position: 1
---

:::tip TL;DR If you have brew

```bash
brew tap pokt-network/poktroll
brew install poktrolld
```

:::

## Table of Contents <!-- omit in toc -->

- [MacOS Users](#macos-users)
  - [Using Homebrew (recommended)](#using-homebrew-recommended)
  - [Using release binaries (if you don't have brew)](#using-release-binaries-if-you-dont-have-brew)
  - [From Source (danger zone)](#from-source-danger-zone)
- [Windows Users (why!?)](#windows-users-why)

## MacOS Users

### Using Homebrew (recommended)

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

### Using release binaries (if you don't have brew)

:::tip tl;dr manual download

1. Grab a binary from the [latest release](https://github.com/pokt-network/poktroll/releases/latest)
2. Download the appropriate `poktroll_${OS}_${ARCH}.tar.gz` for your environment
3. Untar the downloaded file to retrieve the `poktrolld` binary
4. Extract the binary to `/usr/local/bin`
   :::

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

### From Source (danger zone)

:::warning Do not continue unless you're a 🚀👨‍💻💎

This method is only recommended for **ADVANCED** users as it requires working with developer tools.

:::

#### Installing dependencies <!-- omit in toc -->

Ensure you have the following installed:

- [Go](https://go.dev/doc/install) (version 1.23 or later)
- [Make](https://www.gnu.org/software/make/)
- [Ignite CLI](https://docs.ignite.com/welcome/install)

#### Installing poktrolld <!-- omit in toc -->

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

## Windows Users (why!?)

:::danger

Currently, we do not support native Windows installation. Windows users are encouraged
to use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
and follow the Linux installation instructions.

:::

