---
title: poktrolld Installation
sidebar_position: 1
---

:::tip[TLDR: If you know what you're doing.]

```bash
brew tap pokt-network/poktroll
brew install poktrolld
```

or grab a binary from the [releases page](https://github.com/pokt-network/poktroll/releases).

:::

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

:::warning

This method is only recommended for advanced users as it requires working with developer tools.

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

Currently, we do not support native Windows installation. Windows users are encouraged
to use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
and follow the Linux installation instructions.
