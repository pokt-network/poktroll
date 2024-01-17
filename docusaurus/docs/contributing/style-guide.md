---
sidebar_position: 2
title: Style Guide
---

# Style Guide <!-- omit in toc -->

:::warning
We are still settling on our linting guides and have linters in place, we plan
to provide detailed configurations of the tools mentioned below for each editor
you may use but this is a work in progress.
:::

<!-- toc -->

- [Tools](#tools)
  * [Imports](#imports)
    + [Configuration](#configuration)
  * [Formatting](#formatting)
  * [Line Length](#line-length)
    + [Configuration](#configuration-1)

<!-- tocstop -->

## Tools

The tools you should have setup in your editor of choice are listed below with
their configurations.

We have a commands to run to check your code is formatted correctly, simply run:

```sh
make install_deps
make go_lint
```

This runs a custom script that ensures the files are properly formatted,
skipping certain files (like protobuf generated files) and ultimately runs
`golangci-lint` according to its config found in `.golangi.yml`.

### Imports

Our CI and local linting process uses `gci`, a linter available in
`golangci-lint`, you can install this locally and use it by installing it
manually (however this is installed when you run `make install_deps`).

```sh
go install github.com/daixiang0/gci@latest
```

:::tip
However, there is another tool (with **better integrations with modern editors**)
that does the same thing (when using our `.golangi.yml` config) - that is
`goimports-reviser`.
:::

The `goimports-reviser` tool can be installed with the following command:

```sh
go install golang.org/incu6us/goimports-reviser/v3@latest
```

The idea of these tools is to have our imports in `*.go` files (with some
exceptions) formatted in the three blocks as seen in the following example:

:::warning
Both tools remove isolated comments from import blocks.
:::

```go
import (
  // The first group is: Golang's std library packages - this comment would be removed
  "fmt" // this comment would not be removed

  // The second group is: External packages - this comment would be removed
  sdkerrors "cosmossdk.io/errors" // this comment would not be removed

  // The third group is: Internal packages - this comment would be removed
  "github.com/pokt-network/poktroll/x/application/types" // this comment would not be removed
)
```

:::info
The reason we do not use `goimports` is that it groups the second two blocks
together, which is not what we want.
:::

To automatically lint imports for the entire codebase the following command
can be run

```sh
make go_gci
```

This runs a custom script that formats the entire repo's import blocks to match
the format detailed above. It uses `gci` and the rules defined in `.golangci.yml`
to format files.

:::info
Any import blocks that contain ignite scaffold comments are skipped, when this
command is run. It is expected that you format these blocks and manually put
back in the ignite scaffold comment in the correct place. For more information
on how this script works see the `tools/scripts/gci/` directory.
:::

#### Configuration

When using `gci` refer to `.golangci.yml` for more details on how `gci` should
be used.

```yaml
linters-settings:
  gci:
    sections:
      - standard # std lib
      - default # external
      - prefix(github.com/pokt-network/poktroll) # local imports
    skip-generated: true
    custom-order: true
```

When using `goimports-reviser` the defaults are to separate the import block
into the 3 groups the way we want. It does however also remove all comments from
the import block, so import blocks that contain ignite scaffold comments should
be skipped when using this tool or they should be manually put back in.

It is preferred to use some form of `goimports-reviser` plugin in your editor as
its default configuration is the one we use. The only caveat to these tools is
that you have to put back in the ignite scaffold comment in the correct place
after formatting, if it is present. But this is rarely needed.

### Formatting

We use `gofumpt` for formatting our code - this is a stricter version of `gofmt`
and can integrate with all modern editors.

To install `gofumpt` run the following command (this is also installed when you
run `make install_deps`):

```sh
go install mvdan.cc/gofumpt@latest
```

To format the entire codebase you can simply run the following command:

```sh
make go_gofumpt
```

This runs a script to format, selectively, all `*.go` files in the repo.

:::info
Here, selectively means: only go code that was written by hand excluding mocks
and protobuf generated files, among other filters. For more information see
the `tools/scripts/gofumpt/` directory.
:::

### Line Length

Our CI and local linting process uses `lll` a linter available in `golangci-lint`
, however the tool is not installed with `make install_deps` and you will need
to install it manually **if you decide to use it**.

To install `lll` run the following command:

```sh
go install github.com/walle/lll/...@latest
```

However another tool is available for this purpose: `golines`. Which is
supported by all major code editors and can be installed with the following
command:

```sh
go install github.com/segmentio/golines@latest
```

We format our lines in the following way:

1. We try to keep lines between 80-90 characters long.
1. We enforce a max line length of 120 characters.

:::info
The enforcement of 120 characters applies to all files except `errors.go` files
where the line length is ignored as for **these files only** keeping the
definition of errors to a single line is much easier to read.
:::

#### Configuration

When using `lll` refer to `.golangci.yml` for more details on how `lll` should
be used.

```yaml
linters-settings:
  lll:
    line-length: 120
    tab-width: 4
issues:
  exclude-rules:
    - path: errors\.go$
      linters:
        - lll
```

When using `golines` refer to the
[developer tooling](https://github.com/segmentio/golines#developer-tooling-integration)
section of their repo for more information on how to configure it specifically
for your editor but these arguments should be used:

```sh
golines --max-len=120 \
--base-formatter="gofumpt" \
--tab-len=4 \
--ignore-generated \
--write-output <paths...>
```

:::tip
If using (neo)vim `null-ls`/`none-ls` has support for golines out of the box and
can configured according to the rules above as such:

```lua
local sources = {
    null_ls.builtins.formatting.goimports_reviser,
}

-- The poktroll repo requires long lines for the errors.go files
local specific_repo_path = "<path to poktroll repo here>"

local add_golines  = function()
    -- Check if the current file is 'errors.go' in the specific repository
    local current_file = vim.fn.expand("%:p")
    local is_errors_file = current_file:match("errors%.go$")
    local in_specific_repo = current_file:match(specific_repo_path)

    if not in_specific_repo and not is_errors_file then
      table.insert(sources, null_ls.builtins.formatting.golines.with({
        extra_args = {
          "--max-len=120",
          "--base-formatter=gofumpt",
          "--tab-len=4",
          "--ignore-generated",
        },
      }))
  end
end

add_golines()
```

This would be in your custom `null-ls` options definitions file.
:::
