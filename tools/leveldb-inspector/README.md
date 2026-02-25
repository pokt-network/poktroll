# LevelDB Inspector <!-- omit in toc -->

A high-performance CLI tool for analyzing LevelDB databases, specifically designed for inspecting CometBFT transaction indexer databases. Features optimized algorithms, tabular output, and intelligent key analysis.

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Database Statistics](#database-statistics)
    - [Basic Usage](#basic-usage)
    - [Configurable Prefix Display](#configurable-prefix-display)
  - [Browse Database Keys](#browse-database-keys)
  - [Retrieve Specific Values](#retrieve-specific-values)
  - [Analyze Size Distribution](#analyze-size-distribution)
  - [Advanced Prefix Analysis](#advanced-prefix-analysis)
- [CometBFT Transaction Indexer Analysis](#cometbft-transaction-indexer-analysis)
  - [Common Patterns Identified:](#common-patterns-identified)
  - [Use Cases:](#use-cases)
- [Output Formats](#output-formats)
- [Performance Features](#performance-features)
- [Requirements](#requirements)
- [Troubleshooting](#troubleshooting)
  - [Database Access Issues](#database-access-issues)
  - [Large Database Performance](#large-database-performance)
- [Example: Analyzing a Real CometBFT Database](#example-analyzing-a-real-cometbft-database)

## Features

- **📊 Stats**: Comprehensive database statistics with tabular output

  - Key count, total size, and size distributions
  - Configurable top N key prefixes with longest common prefix analysis (default: 10)
  - Remaining keys count for prefixes not shown in top N
  - Sorted size distributions (largest to smallest)

- **🔍 Keys**: Browse and filter database keys

  - Pagination with customizable limits
  - Prefix filtering (hex or string)
  - Automatic key truncation for readability
  - Sort by value size (memory-efficient top-k algorithm)

- **📝 Get**: Retrieve specific key-value pairs

  - Support for hex and string keys
  - Automatic truncation of large values (with --full flag to show complete content)
  - Both hex and string output modes
  - Helpful truncation messages guide users to --full flag

- **📈 Size Analysis**: Identify space consumption patterns

  - Top 20 largest entries in tabular format
  - Key size, value size, and total size breakdown
  - Helps identify what's consuming database space

- **🏷️ Prefix Analysis**: Advanced key organization insights
  - Multi-length prefix analysis (1-8 bytes)
  - Size-based ranking of prefixes
  - Understand data categorization patterns

## Installation

```bash
cd tools/leveldb-inspector
go mod tidy
go build -o leveldb-inspector
```

## Usage

### Database Statistics

Get comprehensive database overview with intelligent prefix analysis:

#### Basic Usage

```bash
./leveldb-inspector -d /path/to/leveldb stats

# Show top 20 prefixes instead of default 10
./leveldb-inspector -d /path/to/leveldb stats --top-prefixes 20

# Show only top 5 prefixes
./leveldb-inspector -d /path/to/leveldb stats -t 5
```

#### Configurable Prefix Display

The `--top-prefixes` (or `-t`) flag controls how many top prefixes to show:

- **Default**: 10 prefixes
- **Range**: 1 to total number of unique prefixes
- **Remaining count**: Shows count and percentage of keys not in top N
- **Works with all output formats**: table, JSON, and CSV

**Example Output:**

```
Database Statistics:
===================
Total Keys: 60546
Total Size: 10.2 MB
Max Key Size: 1408 bytes
Max Value Size: 7140643 bytes

Top 10 Key Prefixes (first 4 bytes):
====================================
Hex Prefix   Common Prefix                                 Count  Percent
----------   -------------                                 -----  -------
0x626c6f63   block_events                                  60465   99.87%
0x636f696e   coin                                             24    0.04%
0x6d657373   message.                                         17    0.03%

...          (remaining prefixes)                            40    0.06%

... (7 more prefixes)

Key Size Distribution:
=====================
Size Range           Count  Percent
----------           -----  -------
64-127 bytes         20196   33.36%
32-63 bytes          38591   63.74%
0-31 bytes            1757    2.90%
```

### Browse Database Keys

List and filter keys with intelligent truncation:

```bash
# List first 50 keys
./leveldb-inspector -d /path/to/leveldb keys -l 50

# Filter by hex prefix
./leveldb-inspector -d /path/to/leveldb keys -p "0x626c6f63"

# Filter by string prefix
./leveldb-inspector -d /path/to/leveldb keys -p "block_events"

# Output in hex format
./leveldb-inspector -d /path/to/leveldb keys --hex

# Sort by value size (descending) - requires --limit
./leveldb-inspector -d /path/to/leveldb keys --sort-by-size --limit 20

# Sort by size with prefix filter
./leveldb-inspector -d /path/to/leveldb keys --sort-by-size --limit 10 --prefix "block_"
```

### Retrieve Specific Values

Get individual key-value pairs:

```bash
# String key lookup
./leveldb-inspector -d /path/to/leveldb get "block_eventsblock.height"

# Hex key lookup
./leveldb-inspector -d /path/to/leveldb get "0x626c6f636b5f6576656e7473" --hex

# Display full value without truncation
./leveldb-inspector -d /path/to/leveldb get "some_key" --full

# Full output in hex format
./leveldb-inspector -d /path/to/leveldb get "some_key" --hex --full
```

### Analyze Size Distribution

Identify what's consuming space:

```bash
./leveldb-inspector -d /path/to/leveldb size
```

**Example Output:**

```
Top 20 Largest Entries:
=======================
Rank Key (hex)                                             Key Size   Value Size      Total
---- ----------                                            --------   ----------      -----
1    b73be3552b5734dea8c3e5d6653e813e4ce1d1f8536da67d0...       32      7140643     6.8 MB
2    e699f385aee68077a4ffbd93796d8c9b936fca938024c204a...       32         2220     2.2 KB
```

### Advanced Prefix Analysis

Understand data organization patterns:

```bash
./leveldb-inspector -d /path/to/leveldb prefixes
```

**Example Output:**

```
Top Prefixes by Size:
====================
Len  Prefix (hex)               Size    Count
---  ------------               ----    -----
8    0xb73be3552b5734de       6.8 MB        1
4    0x626c6f63               3.3 MB    60465
```

## CometBFT Transaction Indexer Analysis

This tool is particularly effective for analyzing CometBFT transaction indexer databases:

### Common Patterns Identified:

- **`block_events`**: Block event indexing (typically 99%+ of keys)
- **`coin`**: Coin transfer and receipt events
- **`message.`**: Message event indexing
- **`transfer.`**: Transfer operation events
- **`tx.fee`**: Transaction fee indexing
- **`tx.signature/`**: Transaction signature lookups
- **`tx.height/`**: Transaction by height indexing
- **`pocket.migration.Event`**: Migration-specific events

### Use Cases:

- **Space Analysis**: Identify what's consuming database space
- **Performance Optimization**: Understand query patterns
- **Data Cleanup**: Find oversized entries or unnecessary data
- **Migration Planning**: Analyze data structure for upgrades

## Output Formats

The tool supports multiple output formats for different use cases:

- **`table`** (default): Human-readable tabular format with proper alignment
- **`json`**: Machine-readable JSON for scripting and integration
- **`csv`**: Comma-separated values for spreadsheet analysis

```bash
# JSON output for scripting
./leveldb-inspector -d /path/to/leveldb stats -o json

# CSV output for spreadsheets
./leveldb-inspector -d /path/to/leveldb stats -o csv

# Show top 20 prefixes in JSON format
./leveldb-inspector -d /path/to/leveldb stats -o json --top-prefixes 20
```

## Performance Features

- **⚡ Optimized Algorithms**: O(n×m) complexity instead of O(n×m×k)
- **🧠 Memory Efficient**: Incremental analysis without storing all keys
- **📏 Smart Truncation**: Prevents terminal overflow from large keys/values
- **📊 Tabular Output**: Professional formatting for easy reading
- **🔄 Incremental LCP**: Longest common prefix calculated on-the-fly

## Requirements

- Go 1.25.7 or later
- LevelDB database (read-only access)

## Troubleshooting

### Database Access Issues

```bash
# Ensure database exists and is readable
ls -la /path/to/leveldb

# Check permissions
./leveldb-inspector -d /path/to/leveldb stats
```

### Large Database Performance

The tool is optimized for large databases, but for databases with millions of keys:

- Use `keys -l` to limit output
- Use prefix filtering to focus analysis
- Consider the `size` command for quick space analysis

## Example: Analyzing a Real CometBFT Database

```bash
# Quick overview
./leveldb-inspector -d ~/.pocket/data/tx_index.db stats

# Find what's using space
./leveldb-inspector -d ~/.pocket/data/tx_index.db size

# Analyze specific prefix
./leveldb-inspector -d ~/.pocket/data/tx_index.db keys -p "block_events" -l 10
```

This will help you understand your database structure and identify optimization opportunities.
