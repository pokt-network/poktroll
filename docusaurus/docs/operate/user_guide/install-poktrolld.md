---
title: Installing poktrolld
sidebar_position: 0
---

- [Full environment setup](#full-environment-setup)
- [`poktrolld` Installation from src](#poktrolld-installation-from-src)
- [\[TODO\] Package manager](#todo-package-manager)

## Full environment setup

We recommend following the instructions provided in our [Developer Quickstart Guide](../../develop/developer_guide/quickstart.md) to make your environment and tools are fully ready for development.
It will build `poktrolld` from source as a bi-product.

## `poktrolld` Installation from src

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

## [TODO] Package manager

_TODO(@okdas): Add ready-to-use binaries (available via homebrew, tea or other package
managers)._
