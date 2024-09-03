---
title: CLI Installation
sidebar_position: 0
---

- [Release binaries](#release-binaries)
- [Installing from source](#installing-from-source)
  - [Prerequisites](#prerequisites)
- [Homebrew and pkgx](#homebrew-and-pkgx)
- [Windows Users](#windows-users)

## Release binaries

Pre-built binaries are available on our [releases page](https://github.com/pokt-network/poktroll/releases).

The following snippet downloads/upgrades the binary to the latest released version (Linux and macOS only):

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

## Installing from source

### Prerequisites

Ensure you have the following installed:

- [Go](https://go.dev/doc/install) (version 1.18 or later)
- [Ignite CLI](https://docs.ignite.com/welcome/install)

```bash
git clone https://github.com/pokt-network/poktroll.git
cd poktroll
make go_develop
make ignite_poktrolld_build
```

Verify it worked by running:

```bash
poktrolld --help
```

## Homebrew and pkgx

:::tip
We have an [open GitHub issue](https://github.com/pokt-network/poktroll/issues/535)
to introduce `poktrolld` to [brew](https://brew.sh/) and [pkgx](https://github.com/pkgxdev/pkgx).

Please reach out to us in the ticket if you want to pick this ticket!
:::

## Windows Users

Currently, we do not support native Windows installation. Windows users are encouraged
to use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
and follow the Linux installation instructions.
