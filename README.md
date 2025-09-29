<!-- markdownlint-disable MD033 -->
<!-- markdownlint-disable MD045 -->

<div align="center">
  <a href="https://www.pokt.network">
    <img src="https://github.com/user-attachments/assets/5dbddd4a-d932-4c44-8396-270f140f086a" alt="Pocket Network logo" width="340"/>
  </a>
</div>

<div>
  <a href="https://discord.gg/pokt"><img src="https://img.shields.io/discord/553741558869131266"/></a>
  <a  href="https://github.com/pokt-network/poktroll/releases"><img src="https://img.shields.io/github/release-pre/pokt-network/pocket.svg"/></a>
  <a  href="https://github.com/pokt-network/poktroll/pulse"><img src="https://img.shields.io/github/contributors/pokt-network/pocket.svg"/></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/pulse"><img src="https://img.shields.io/github/last-commit/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/pulls"><img src="https://img.shields.io/github/issues-pr/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/releases"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos-pink.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/issues"><img src="https://img.shields.io/github/issues/pokt-network/pocket.svg"/></a>
  <a href="https://github.com/pokt-network/poktroll/issues"><img src="https://img.shields.io/github/issues-closed/pokt-network/pocket.svg"/></a>
  <a href="https://godoc.org/github.com/pokt-network/pocket"><img src="https://img.shields.io/badge/godoc-reference-blue.svg"/></a>
  <a href="https://goreportcard.com/report/github.com/pokt-network/pocket"><img src="https://goreportcard.com/badge/github.com/pokt-network/pocket"/></a>
  <a href="https://golang.org"><img  src="https://img.shields.io/badge/golang-v1.24-green.svg"/></a>
  <a href="https://github.com/tools/godep" ><img src="https://img.shields.io/badge/godep-dependency-71a3d9.svg"/></a>
</div>

# pocket <!-- omit in toc -->

**pocket** is the source code for [Pocket Network's](https://pokt.network/)
[Shannon upgrade](https://docs.pokt.network/pokt-protocol/the-shannon-upgrade).

For technical documentation, visit [dev.poktroll.com](https://dev.poktroll.com).

Documentation is maintained in the [docusaurus repo](./docusaurus) and is
automatically deployed to the link above.

# Prerequisites

```bash
# Install required tools
brew install gnu-sed
brew install grep
brew install yq
brew install kind
brew install tilt

# Verify Go version (must be 1.23.x, NOT 1.24+)
go version
# Expected output: go1.23.12 darwin/arm64 

```



# Quickstart (Localnet)
```
# Start the local network
make localnet_up

# Initialize accounts (required for relays & tests)
make acc_initialize_pubkeys POCKET_NODE=http://localhost:26657
```
Troubleshooting

If you see this error:

It usually means you forgot to run the make acc_initialize_pubkeys step.
```



## License

This project is licensed under the MIT License.  
See the [LICENSE](https://github.com/pokt-network/poktroll/blob/main/LICENSE) file for details.

