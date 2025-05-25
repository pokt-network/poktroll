---
title: pocketd CLI Installation
sidebar_position: 1
---

:::tip TL;DR
To install `pocketd` on Linux or MacOS, run:

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
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
- [Windows (why!?)](#windows-why)
- [Publishing a new CLI release](#publishing-a-new-cli-release)
- [8. Update the `homebrew-tap` Formula](#8-update-the-homebrew-tap-formula)

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

## Windows (why!?)

:::danger

- Native Windows installation is **not supported**.
- Use [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install)
- Follow the Linux install instructions above.
  :::

---

## Publishing a new CLI release

## 8. Update the `homebrew-tap` Formula

Once the upgrade is validated, update the tap so users can install the new CLI.

**Steps:**

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocketd
make tap_update_version
git commit -am "Update pocket tap from v.X1.Y1.Z1 to v.X1.Y2.Z2"
git push
```

**Reinstall the CLI:**

```bash
brew reinstall pocketd
```

**Or install for the first time:**

```bash
brew tap pocket-network/homebrew-pocketd
brew install pocketd
```

See [pocketd CLI docs](../../2_explore/2_account_management/1_pocketd_cli.md) for more info.

:::note
You can review [all prior releases here](https://github.com/pokt-network/poktroll/releases).
:::

1. **Tag the release** using one of the following and follow on-screen prompts:

   ```bash
   make release_tag_bug_fix
   # or
   make release_tag_minor_release
   ```

2. **Publish the release** by:

   - [Drafting a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Using the tag from the step above

3. **Update the description in the release** by:

   - Clicking `Generate release notes` in the GitHub UI
   - Add this table **ABOVE** the auto-generated notes (below)

     ```markdown
     ## Protocol Upgrades

     | Category                     | Applicable | Notes                                |
     | ---------------------------- | ---------- | ------------------------------------ |
     | Planned Upgrade              | ✅         | New features.                        |
     | Consensus Breaking Change    | ✅         | Yes, see upgrade here: #1216         |
     | Manual Intervention Required | ❓         | Cosmosvisor managed everything well. |

     | Network       | Upgrade Height | Upgrade Transaction Hash | Notes |
     | ------------- | -------------- | ------------------------ | ----- |
     | Alpha TestNet | ⚪             | ⚪                       | ⚪    |
     | Beta TestNet  | ⚪             | ⚪                       | ⚪    |
     | MainNet       | ⚪             | ⚪                       | ⚪    |

     **Legend**:

     - ⚠️ - Warning/Caution Required
     - ✅ - Yes
     - ❌ - No
     - ⚪ - Will be filled out throughout the release process / To Be Determined
     - ❓ - Unknown / Needs Discussion

     ## What's Changed

     <!-- Auto-generated GitHub Release Notes continue here -->
     ```

4. **Set as a pre-release** (change to `latest release` after upgrade completes).
