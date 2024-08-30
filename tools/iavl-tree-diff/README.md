# iavl-tree-diff

**iavl-tree-diff** is a tool designed to compare IAVL tree databases in Cosmos SDK-based blockchains. It identifies differences between two IAVL tree versions across multiple prefixes and highlights key-value discrepancies. This is particularly useful for debugging non-deterministic issues in distributed systems where consistency is critical.

## Features

- **Prefix Extraction**: Automatically extracts prefixes used in the databases for different modules.
- **Version Comparison**: Compares two databases at the latest common version for each prefix.
- **Tree Hash Comparison**: Computes and compares tree hashes to quickly detect discrepancies.
- **Key-Value Difference Detection**: Identifies and outputs differences in keys and values when tree hashes differ.

## How It Works

1. **Prefix Detection**: The tool scans the provided databases to extract all relevant prefixes.
2. **Version Selection**: For each prefix, it determines the latest common version between the two databases.
3. **Tree Comparison**: The tool compares the tree hashes at the selected version. If the hashes differ, it proceeds to identify specific key-value differences.
4. **Output**: The differences, if any, are printed to the console, showing the exact keys and values that vary between the two databases.

## Usage

```bash
go run . <path_to_db1> <path_to_db2>
```

Example:

```bash
go run . $HOME/pocket/testnet/halt-08-26/data-val/application.db $HOME/pocket/testnet/halt-08-26/data-fullnode/application.db
```

## Example Output

```
Checking prefix: s/k:group/
Using lower version: 15910
Tree hash from db1: A89113C07AE262E84FA49E6D9111BF7C638679604357FA388526A976CFD021E0
Tree hash from db2: A89113C07AE262E84FA49E6D9111BF7C638679604357FA388526A976CFD021E0
Tree hashes are identical.
Checking prefix: s/k:supplier/
Using lower version: 15910
Tree hash from db1: 148180BFC904425FDBDD7046D4ECAA7238281DC2E76C66E079DBA76E5BCCD15C
Tree hash from db2: 31E6BCCD1B2712342B634C29651CE4DF4072080B70EBF26185BB4836D0A89B68
Hashes differ, checking for differences in keys/values...
Key: Supplier/operator_address/pokt10a2lwlrkraqx6sud6gc8mk4ewvmtany6e9z7mp/
Value in db1: 0A2B706F6B7431...
Value in db2: 0A2B706F6B74313061326C776C72...
```
