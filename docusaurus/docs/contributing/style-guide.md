---
sidebar_position: 1
title: Style Guide
---

# Style Guide <!-- omit in toc -->

:::note
We are still settling on our linting guides and have linters in place, we plan
to provide detailed configurations of the tools mentioned below for each editor
you may use but this is a work in progress.
:::

<!-- toc -->

- [Tools](#tools)
  * [`goimports-reviser`](#goimports-reviser)

<!-- tocstop -->

## Tools

The tools you should have setup in your editor of choice are listed below with
their configurations.

### Imports

Our CI and local linting process uses `lll` a linter available in `golangci-lint`
but you can install this locally and use it by installing it manually:

```sh
go install github.com/walle/lll/...@latest
```

:::tip
However, there is another tool (with **better integrations with modern editors**)
that does the same thing (when using our `.golangi.yml` config)
`goimports-reviser`.
:::

The `goimports-reviser` tool can be installed with the following command:

```sh
go install golang.org/incu6us/goimports-reviser/v3@latest
```

The idea of these tools is to have our imports in `*.go` files (with some
exceptions) formatted in the three blocks as seen in the following example:

:::warning
Both tools remove comments from import blocks.
:::

```go
import (
  // The first group is: Golang's std library packages
  "fmt" 

  // The second group is: External packages
  sdkerrors "cosmossdk.io/errors" 

  // The third group is: Internal packages
  "github.com/pokt-network/poktroll/x/application/types"
)
```

:::info
The reason we do not use `goimports` is that it groups the second two blocks
together, which is not what we want.
:::

To automatically lint imports for the entire codebase the following command can
be run

```sh
make go_gci
```

This runs a custom script that formats the entire repo's import blocks to match
the format detailed above.

:::info
Any import blocks that contain ignite scaffold comments are skipped, when this
command is run. **It is expected that you format these blocks and manually
put back in the ignite scaffold comment in the correct place**.
:::

#### Configuration

When using `lll` refer to `.golangci.yml` for more details on how `lll` should
be used.

When using `goimports-reviser` the defaults are to seperate the import block
into the 3 groups we want. It does however also remove all comments from the
import block, so import blocks that contain ignite scaffold comments should be
skipped when using this tool or they should be manually put back in.

It is preferred to use `goimports-reviser` in your editor as its default
configuration is the one we use. The only caveat to these tools is that you
have to put back in the ignite scaffold comment in the correct place after
formatting, if it is present.
