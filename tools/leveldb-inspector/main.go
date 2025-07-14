package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Stats struct {
	TotalKeys             int64
	TotalSize             int64
	KeyPrefixes           map[string]int64
	KeySizes              map[string]int64
	ValueSizes            map[string]int64
	LargestKey            string
	LargestValue          string
	MaxKeySize            int
	MaxValueSize          int
	LongestCommonPrefixes map[string]string
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "leveldb-inspector",
		Short: "Inspect LevelDB databases for size analysis and key distribution",
		Long:  `A CLI tool to analyze LevelDB databases, particularly useful for CometBFT tx indexer databases.`,
	}

	var dbPath string
	var output string
	var limit int
	var hexOutput bool
	var prefix string
	var topPrefixes int
	var sortBySize bool
	var sortByKeySize bool
	var fullOutput bool

	rootCmd.PersistentFlags().StringVarP(&dbPath, "db", "d", "", "Path to LevelDB database (required)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format (table, json, csv)")
	rootCmd.PersistentFlags().BoolVar(&hexOutput, "hex", false, "Display keys/values in hex format")
	rootCmd.MarkPersistentFlagRequired("db")

	// Stats command
	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Display database statistics",
		Long:  `Analyze the database and display key statistics including size distribution and prefix analysis.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStats(dbPath, output, hexOutput, topPrefixes)
		},
	}

	statsCmd.Flags().IntVarP(&topPrefixes, "top-prefixes", "t", 10, "Number of top prefixes to display")

	// Keys command
	var keysCmd = &cobra.Command{
		Use:   "keys",
		Short: "List keys in the database",
		Long:  `List all keys in the database with optional filtering and limiting.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that sort flags are not used together
			if sortBySize && sortByKeySize {
				return fmt.Errorf("cannot use both --sort-by-size and --sort-by-key-size together")
			}
			// Validate that limit is specified when using any sort flag
			if (sortBySize || sortByKeySize) && limit == 0 {
				return fmt.Errorf("--limit must be specified when using sorting flags")
			}
			return runKeys(dbPath, output, limit, hexOutput, prefix, sortBySize, sortByKeySize)
		},
	}

	keysCmd.Flags().IntVarP(&limit, "limit", "l", 0, "Maximum number of keys to display (default 100 if not sorting)")
	keysCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "Filter keys by prefix")
	keysCmd.Flags().BoolVarP(&sortBySize, "sort-by-size", "s", false, "Sort keys by value size in descending order (requires --limit)")
	keysCmd.Flags().BoolVarP(&sortByKeySize, "sort-by-key-size", "k", false, "Sort keys by key size in descending order (requires --limit)")

	// Get command
	var getCmd = &cobra.Command{
		Use:   "get <key>",
		Short: "Get value for a specific key",
		Long:  `Retrieve and display the value for a specific key.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(dbPath, args[0], hexOutput, fullOutput)
		},
	}

	getCmd.Flags().BoolVarP(&fullOutput, "full", "f", false, "Display full value without truncation")

	// Size command
	var sizeCmd = &cobra.Command{
		Use:   "size",
		Short: "Analyze size distribution",
		Long:  `Analyze the size distribution of keys and values to identify what's taking up space.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSizeAnalysis(dbPath, output)
		},
	}

	// Prefixes command
	var prefixesCmd = &cobra.Command{
		Use:   "prefixes",
		Short: "Analyze key prefixes",
		Long:  `Analyze key prefixes to understand data organization and identify large data categories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrefixAnalysis(dbPath, output)
		},
	}

	rootCmd.AddCommand(statsCmd, keysCmd, getCmd, sizeCmd, prefixesCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func openDB(path string) (*leveldb.DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database path does not exist: %s", absPath)
	}

	db, err := leveldb.OpenFile(absPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return db, nil
}

func runStats(dbPath, output string, hexOutput bool, topPrefixes int) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	stats, err := collectStats(db)
	if err != nil {
		return err
	}

	return displayStats(stats, output, hexOutput, topPrefixes)
}

func collectStats(db *leveldb.DB) (*Stats, error) {
	stats := &Stats{
		KeyPrefixes: make(map[string]int64),
		KeySizes:    make(map[string]int64),
		ValueSizes:  make(map[string]int64),
	}

	// Track LCP incrementally - much more efficient
	prefixLCPs := make(map[string]string)
	prefixInitialized := make(map[string]bool)

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keyStr := string(key)

		stats.TotalKeys++
		keySize := len(key)
		valueSize := len(value)
		stats.TotalSize += int64(keySize + valueSize)

		// Track largest key and value
		if keySize > stats.MaxKeySize {
			stats.MaxKeySize = keySize
			stats.LargestKey = keyStr
		}
		if valueSize > stats.MaxValueSize {
			stats.MaxValueSize = valueSize
			stats.LargestValue = keyStr
		}

		// Analyze key prefixes (first 4 bytes as hex for CometBFT indexer)
		var prefixKey string
		if len(key) >= 4 {
			prefixKey = hex.EncodeToString(key[:4])
		} else {
			prefixKey = hex.EncodeToString(key)
		}
		stats.KeyPrefixes[prefixKey]++

		// Incrementally update LCP - O(m) per key instead of O(n×m) at the end
		if !prefixInitialized[prefixKey] {
			prefixLCPs[prefixKey] = keyStr
			prefixInitialized[prefixKey] = true
		} else {
			prefixLCPs[prefixKey] = findCommonPrefix(prefixLCPs[prefixKey], keyStr)
		}

		// Size distribution
		keySizeRange := getSizeRange(keySize)
		valueSizeRange := getSizeRange(valueSize)
		stats.KeySizes[keySizeRange]++
		stats.ValueSizes[valueSizeRange]++
	}

	stats.LongestCommonPrefixes = prefixLCPs
	return stats, iter.Error()
}

func getSizeRange(size int) string {
	if size < 32 {
		return "0-31"
	} else if size < 64 {
		return "32-63"
	} else if size < 128 {
		return "64-127"
	} else if size < 256 {
		return "128-255"
	} else if size < 512 {
		return "256-511"
	} else if size < 1024 {
		return "512-1023"
	} else if size < 2048 {
		return "1024-2047"
	} else if size < 4096 {
		return "2048-4095"
	} else {
		return "4096+"
	}
}

func getSizeRangeValue(sizeRange string) int {
	switch sizeRange {
	case "0-31":
		return 31
	case "32-63":
		return 63
	case "64-127":
		return 127
	case "128-255":
		return 255
	case "256-511":
		return 511
	case "512-1023":
		return 1023
	case "1024-2047":
		return 2047
	case "2048-4095":
		return 4095
	case "4096+":
		return 999999 // Treat as largest
	default:
		return 0
	}
}

func displayStats(stats *Stats, output string, hexOutput bool, topPrefixes int) error {
	switch output {
	case "table":
		return displayStatsTable(stats, hexOutput, topPrefixes)
	case "json":
		return displayStatsJSON(stats, topPrefixes)
	case "csv":
		return displayStatsCSV(stats, topPrefixes)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func displayStatsTable(stats *Stats, hexOutput bool, topPrefixes int) error {
	fmt.Printf("Database Statistics:\n")
	fmt.Printf("===================\n")
	fmt.Printf("Total Keys: %d\n", stats.TotalKeys)
	fmt.Printf("Total Logical Size: %s (uncompressed key+value data)\n", formatBytes(stats.TotalSize))
	fmt.Printf("Max Key Size: %s\n", formatBytes(int64(stats.MaxKeySize)))
	fmt.Printf("Max Value Size: %s\n", formatBytes(int64(stats.MaxValueSize)))
	fmt.Printf("\nNote: Actual disk usage may be smaller due to LevelDB compression.\n")
	fmt.Printf("      Use 'du -hs <db_path>' to see actual disk space used.\n")

	if hexOutput {
		fmt.Printf("Largest Key (hex): %s\n", truncateKey(hex.EncodeToString([]byte(stats.LargestKey)), 100))
		fmt.Printf("Key with Largest Value (hex): %s\n", truncateKey(hex.EncodeToString([]byte(stats.LargestValue)), 100))
	} else {
		fmt.Printf("Largest Key: %s\n", truncateKey(stats.LargestKey, 100))
		fmt.Printf("Key with Largest Value: %s\n", truncateKey(stats.LargestValue, 100))
	}

	fmt.Printf("\nTop %d Key Prefixes (first 4 bytes):\n", topPrefixes)
	fmt.Printf("====================================\n")
	prefixPairs := make([]struct {
		prefix string
		count  int64
	}, 0, len(stats.KeyPrefixes))
	for prefix, count := range stats.KeyPrefixes {
		prefixPairs = append(prefixPairs, struct {
			prefix string
			count  int64
		}{prefix, count})
	}
	sort.Slice(prefixPairs, func(i, j int) bool {
		return prefixPairs[i].count > prefixPairs[j].count
	})

	maxPrefixes := topPrefixes
	if len(prefixPairs) < maxPrefixes {
		maxPrefixes = len(prefixPairs)
	}

	fmt.Printf("%-12s %-40s %10s %8s\n", "Hex Prefix", "Common Prefix", "Count", "Percent")
	fmt.Printf("%-12s %-40s %10s %8s\n", "----------", "-------------", "-----", "-------")

	for i := 0; i < maxPrefixes; i++ {
		pair := prefixPairs[i]
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		lcp := stats.LongestCommonPrefixes[pair.prefix]
		if lcp == "" {
			lcp = decodeHexPrefix(pair.prefix)
		}
		lcp = truncateKey(lcp, 40)
		fmt.Printf("%-12s %-40s %10d %7.2f%%\n", "0x"+pair.prefix, lcp, pair.count, percentage)
	}

	// Calculate remaining keys count
	if len(prefixPairs) > maxPrefixes {
		remainingCount := int64(0)
		for i := maxPrefixes; i < len(prefixPairs); i++ {
			remainingCount += prefixPairs[i].count
		}
		remainingPercentage := float64(remainingCount) / float64(stats.TotalKeys) * 100
		fmt.Printf("\n%-12s %-40s %10d %7.2f%%\n", "...", "(remaining prefixes)", remainingCount, remainingPercentage)
		fmt.Printf("\n... (%d more prefixes)\n", len(prefixPairs)-maxPrefixes)
	}

	fmt.Printf("\nKey Size Distribution:\n")
	fmt.Printf("=====================\n")
	keySizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.KeySizes))
	for size, count := range stats.KeySizes {
		keySizePairs = append(keySizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(keySizePairs, func(i, j int) bool {
		return getSizeRangeValue(keySizePairs[i].size) > getSizeRangeValue(keySizePairs[j].size)
	})

	fmt.Printf("%-15s %10s %8s\n", "Size Range", "Count", "Percent")
	fmt.Printf("%-15s %10s %8s\n", "----------", "-----", "-------")
	for _, pair := range keySizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("%-15s %10d %7.2f%%\n", pair.size+" bytes", pair.count, percentage)
	}

	fmt.Printf("\nValue Size Distribution:\n")
	fmt.Printf("=======================\n")
	valueSizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.ValueSizes))
	for size, count := range stats.ValueSizes {
		valueSizePairs = append(valueSizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(valueSizePairs, func(i, j int) bool {
		return getSizeRangeValue(valueSizePairs[i].size) > getSizeRangeValue(valueSizePairs[j].size)
	})

	fmt.Printf("%-15s %10s %8s\n", "Size Range", "Count", "Percent")
	fmt.Printf("%-15s %10s %8s\n", "----------", "-----", "-------")
	for _, pair := range valueSizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("%-15s %10d %7.2f%%\n", pair.size+" bytes", pair.count, percentage)
	}

	return nil
}

func displayStatsJSON(stats *Stats, topPrefixes int) error {
	fmt.Printf("{\n")
	fmt.Printf("  \"total_keys\": %d,\n", stats.TotalKeys)
	fmt.Printf("  \"total_logical_size\": %d,\n", stats.TotalSize)
	fmt.Printf("  \"max_key_size\": %d,\n", stats.MaxKeySize)
	fmt.Printf("  \"max_value_size\": %d,\n", stats.MaxValueSize)
	fmt.Printf("  \"largest_key\": %q,\n", truncateKey(stats.LargestKey, 100))
	fmt.Printf("  \"key_with_largest_value\": %q,\n", truncateKey(stats.LargestValue, 100))

	// Top prefixes with LCP
	fmt.Printf("  \"top_prefixes\": [\n")
	prefixPairs := make([]struct {
		prefix string
		count  int64
	}, 0, len(stats.KeyPrefixes))
	for prefix, count := range stats.KeyPrefixes {
		prefixPairs = append(prefixPairs, struct {
			prefix string
			count  int64
		}{prefix, count})
	}
	sort.Slice(prefixPairs, func(i, j int) bool {
		return prefixPairs[i].count > prefixPairs[j].count
	})

	maxPrefixes := topPrefixes
	if len(prefixPairs) < maxPrefixes {
		maxPrefixes = len(prefixPairs)
	}

	for i := 0; i < maxPrefixes; i++ {
		pair := prefixPairs[i]
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		lcp := stats.LongestCommonPrefixes[pair.prefix]
		if lcp == "" {
			lcp = decodeHexPrefix(pair.prefix)
		}
		lcp = truncateKey(lcp, 40)

		fmt.Printf("    {\n")
		fmt.Printf("      \"hex_prefix\": \"0x%s\",\n", pair.prefix)
		fmt.Printf("      \"common_prefix\": %q,\n", lcp)
		fmt.Printf("      \"count\": %d,\n", pair.count)
		fmt.Printf("      \"percentage\": %.2f\n", percentage)
		fmt.Printf("    }")
		if i < maxPrefixes-1 {
			fmt.Printf(",")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("  ],\n")

	// Key size distribution
	fmt.Printf("  \"key_size_distribution\": [\n")
	keySizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.KeySizes))
	for size, count := range stats.KeySizes {
		keySizePairs = append(keySizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(keySizePairs, func(i, j int) bool {
		return getSizeRangeValue(keySizePairs[i].size) > getSizeRangeValue(keySizePairs[j].size)
	})

	for i, pair := range keySizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("    {\n")
		fmt.Printf("      \"size_range\": \"%s bytes\",\n", pair.size)
		fmt.Printf("      \"count\": %d,\n", pair.count)
		fmt.Printf("      \"percentage\": %.2f\n", percentage)
		fmt.Printf("    }")
		if i < len(keySizePairs)-1 {
			fmt.Printf(",")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("  ],\n")

	// Value size distribution
	fmt.Printf("  \"value_size_distribution\": [\n")
	valueSizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.ValueSizes))
	for size, count := range stats.ValueSizes {
		valueSizePairs = append(valueSizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(valueSizePairs, func(i, j int) bool {
		return getSizeRangeValue(valueSizePairs[i].size) > getSizeRangeValue(valueSizePairs[j].size)
	})

	for i, pair := range valueSizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("    {\n")
		fmt.Printf("      \"size_range\": \"%s bytes\",\n", pair.size)
		fmt.Printf("      \"count\": %d,\n", pair.count)
		fmt.Printf("      \"percentage\": %.2f\n", percentage)
		fmt.Printf("    }")
		if i < len(valueSizePairs)-1 {
			fmt.Printf(",")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("  ]\n")
	fmt.Printf("}\n")
	return nil
}

func displayStatsCSV(stats *Stats, topPrefixes int) error {
	// Basic metrics
	fmt.Println("metric,value")
	fmt.Printf("total_keys,%d\n", stats.TotalKeys)
	fmt.Printf("total_logical_size,%d\n", stats.TotalSize)
	fmt.Printf("max_key_size,%d\n", stats.MaxKeySize)
	fmt.Printf("max_value_size,%d\n", stats.MaxValueSize)

	// Top prefixes
	fmt.Println("\ntype,hex_prefix,common_prefix,count,percentage")
	prefixPairs := make([]struct {
		prefix string
		count  int64
	}, 0, len(stats.KeyPrefixes))
	for prefix, count := range stats.KeyPrefixes {
		prefixPairs = append(prefixPairs, struct {
			prefix string
			count  int64
		}{prefix, count})
	}
	sort.Slice(prefixPairs, func(i, j int) bool {
		return prefixPairs[i].count > prefixPairs[j].count
	})

	maxPrefixes := topPrefixes
	if len(prefixPairs) < maxPrefixes {
		maxPrefixes = len(prefixPairs)
	}

	for i := 0; i < maxPrefixes; i++ {
		pair := prefixPairs[i]
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		lcp := stats.LongestCommonPrefixes[pair.prefix]
		if lcp == "" {
			lcp = decodeHexPrefix(pair.prefix)
		}
		lcp = truncateKey(lcp, 40)
		fmt.Printf("prefix,0x%s,\"%s\",%d,%.2f\n", pair.prefix, lcp, pair.count, percentage)
	}

	// Key size distribution
	fmt.Println("\ntype,size_range,count,percentage")
	keySizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.KeySizes))
	for size, count := range stats.KeySizes {
		keySizePairs = append(keySizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(keySizePairs, func(i, j int) bool {
		return getSizeRangeValue(keySizePairs[i].size) > getSizeRangeValue(keySizePairs[j].size)
	})

	for _, pair := range keySizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("key_size,%s bytes,%d,%.2f\n", pair.size, pair.count, percentage)
	}

	// Value size distribution
	valueSizePairs := make([]struct {
		size  string
		count int64
	}, 0, len(stats.ValueSizes))
	for size, count := range stats.ValueSizes {
		valueSizePairs = append(valueSizePairs, struct {
			size  string
			count int64
		}{size, count})
	}
	sort.Slice(valueSizePairs, func(i, j int) bool {
		return getSizeRangeValue(valueSizePairs[i].size) > getSizeRangeValue(valueSizePairs[j].size)
	})

	for _, pair := range valueSizePairs {
		percentage := float64(pair.count) / float64(stats.TotalKeys) * 100
		fmt.Printf("value_size,%s bytes,%d,%.2f\n", pair.size, pair.count, percentage)
	}

	return nil
}

func runKeys(dbPath, output string, limit int, hexOutput bool, prefix string, sortBySize bool, sortByKeySize bool) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var iter iterator.Iterator
	if prefix != "" {
		// Convert hex prefix to bytes if it starts with 0x
		var prefixBytes []byte
		if strings.HasPrefix(prefix, "0x") {
			prefixBytes, err = hex.DecodeString(prefix[2:])
			if err != nil {
				return fmt.Errorf("invalid hex prefix: %w", err)
			}
		} else {
			prefixBytes = []byte(prefix)
		}
		iter = db.NewIterator(util.BytesPrefix(prefixBytes), nil)
	} else {
		iter = db.NewIterator(nil, nil)
	}
	defer iter.Release()

	// Apply default limit if not specified and not sorting
	if limit == 0 && !sortBySize && !sortByKeySize {
		limit = 100
	}

	if sortBySize || sortByKeySize {
		// Memory-efficient top-k algorithm
		type keyEntry struct {
			key       []byte
			keySize   int
			valueSize int
		}

		// Maintain a slice of top entries
		topEntries := make([]keyEntry, 0, limit)

		for iter.Next() {
			key := make([]byte, len(iter.Key()))
			copy(key, iter.Key())
			keySize := len(iter.Key())
			valueSize := len(iter.Value())

			entry := keyEntry{key: key, keySize: keySize, valueSize: valueSize}

			// Determine which size to compare based on sort mode
			var entrySize int
			if sortByKeySize {
				entrySize = keySize
			} else {
				entrySize = valueSize
			}

			if len(topEntries) < limit {
				// Haven't filled the limit yet, just add
				topEntries = append(topEntries, entry)
				// Sort to maintain order
				sort.Slice(topEntries, func(i, j int) bool {
					if sortByKeySize {
						return topEntries[i].keySize < topEntries[j].keySize
					} else {
						return topEntries[i].valueSize < topEntries[j].valueSize
					}
				})
			} else {
				// Check if this entry is larger than the smallest
				var smallestSize int
				if sortByKeySize {
					smallestSize = topEntries[0].keySize
				} else {
					smallestSize = topEntries[0].valueSize
				}

				if entrySize > smallestSize {
					// Replace the smallest entry
					topEntries[0] = entry
					// Re-sort to maintain order
					sort.Slice(topEntries, func(i, j int) bool {
						if sortByKeySize {
							return topEntries[i].keySize < topEntries[j].keySize
						} else {
							return topEntries[i].valueSize < topEntries[j].valueSize
						}
					})
				}
			}
		}

		// Display in descending order (largest first)
		for i := len(topEntries) - 1; i >= 0; i-- {
			entry := topEntries[i]
			if hexOutput {
				fmt.Printf("Key: %s, Key Size: %s, Value Size: %s\n",
					truncateKey(hex.EncodeToString(entry.key), 80),
					formatBytes(int64(entry.keySize)),
					formatBytes(int64(entry.valueSize)))
			} else {
				fmt.Printf("Key: %s, Key Size: %s, Value Size: %s\n",
					truncateKey(string(entry.key), 80),
					formatBytes(int64(entry.keySize)),
					formatBytes(int64(entry.valueSize)))
			}
		}

		if len(topEntries) > 0 {
			if sortByKeySize {
				fmt.Printf("... (showing top %d keys by key size)\n", len(topEntries))
			} else {
				fmt.Printf("... (showing top %d keys by value size)\n", len(topEntries))
			}
		}

		return iter.Error()
	}

	// Original behavior when not sorting
	count := 0
	for iter.Next() && count < limit {
		key := iter.Key()
		value := iter.Value()

		if hexOutput {
			fmt.Printf("Key: %s, Value Size: %s\n", truncateKey(hex.EncodeToString(key), 80), formatBytes(int64(len(value))))
		} else {
			fmt.Printf("Key: %s, Value Size: %s\n", truncateKey(string(key), 80), formatBytes(int64(len(value))))
		}
		count++
	}

	if count == limit {
		fmt.Printf("... (showing first %d keys)\n", limit)
	}

	return iter.Error()
}

func runGet(dbPath, key string, hexOutput bool, fullOutput bool) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var keyBytes []byte
	if strings.HasPrefix(key, "0x") {
		keyBytes, err = hex.DecodeString(key[2:])
		if err != nil {
			return fmt.Errorf("invalid hex key: %w", err)
		}
	} else {
		keyBytes = []byte(key)
	}

	value, err := db.Get(keyBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	if hexOutput {
		keyOutput, keyTruncated := truncateWithMessage(hex.EncodeToString(keyBytes), 100, fullOutput, "key")
		valueOutput, valueTruncated := truncateWithMessage(hex.EncodeToString(value), 200, fullOutput, "value")

		fmt.Printf("Key: %s\n", keyOutput)
		if keyTruncated {
			fmt.Printf("     (key truncated, use --full to see complete key)\n")
		}

		fmt.Printf("Value (%s): %s\n", formatBytes(int64(len(value))), valueOutput)
		if valueTruncated {
			fmt.Printf("      (value truncated, use --full to see complete value)\n")
		}
	} else {
		keyOutput, keyTruncated := truncateWithMessage(string(keyBytes), 100, fullOutput, "key")
		valueOutput, valueTruncated := truncateWithMessage(string(value), 200, fullOutput, "value")

		fmt.Printf("Key: %s\n", keyOutput)
		if keyTruncated {
			fmt.Printf("     (key truncated, use --full to see complete key)\n")
		}

		fmt.Printf("Value (%s): %s\n", formatBytes(int64(len(value))), valueOutput)
		if valueTruncated {
			fmt.Printf("      (value truncated, use --full to see complete value)\n")
		}
	}

	return nil
}

func runSizeAnalysis(dbPath, output string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	type keyInfo struct {
		key       []byte
		keySize   int
		valueSize int
	}

	var keys []keyInfo
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keys = append(keys, keyInfo{
			key:       key,
			keySize:   len(iter.Key()),
			valueSize: len(iter.Value()),
		})
	}

	// Sort by total size (key + value)
	sort.Slice(keys, func(i, j int) bool {
		return (keys[i].keySize + keys[i].valueSize) > (keys[j].keySize + keys[j].valueSize)
	})

	fmt.Printf("Top 20 Largest Entries:\n")
	fmt.Printf("=======================\n")
	fmt.Printf("%-4s %-62s %10s %12s %10s\n", "Rank", "Key (hex)", "Key Size", "Value Size", "Total")
	fmt.Printf("%-4s %-62s %10s %12s %10s\n", "----", "----------", "--------", "----------", "-----")
	for i, keyInfo := range keys {
		if i >= 20 {
			break
		}
		totalSize := keyInfo.keySize + keyInfo.valueSize
		fmt.Printf("%-4d %-62s %10s %12s %10s\n",
			i+1, truncateKey(hex.EncodeToString(keyInfo.key), 60), formatBytes(int64(keyInfo.keySize)), formatBytes(int64(keyInfo.valueSize)), formatBytes(int64(totalSize)))
	}

	return iter.Error()
}

func runPrefixAnalysis(dbPath, output string) error {
	db, err := openDB(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	prefixSizes := make(map[string]int64)
	prefixCounts := make(map[string]int64)

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Analyze different prefix lengths
		for prefixLen := 1; prefixLen <= 8 && prefixLen <= len(key); prefixLen++ {
			prefix := hex.EncodeToString(key[:prefixLen])
			prefixKey := fmt.Sprintf("%d:%s", prefixLen, prefix)
			prefixSizes[prefixKey] += int64(len(key) + len(value))
			prefixCounts[prefixKey]++
		}
	}

	// Sort by size
	type prefixInfo struct {
		prefix string
		size   int64
		count  int64
	}

	var prefixes []prefixInfo
	for prefix, size := range prefixSizes {
		prefixes = append(prefixes, prefixInfo{
			prefix: prefix,
			size:   size,
			count:  prefixCounts[prefix],
		})
	}

	sort.Slice(prefixes, func(i, j int) bool {
		return prefixes[i].size > prefixes[j].size
	})

	fmt.Printf("Top Prefixes by Size:\n")
	fmt.Printf("====================\n")
	fmt.Printf("%-4s %-20s %10s %8s\n", "Len", "Prefix (hex)", "Size", "Count")
	fmt.Printf("%-4s %-20s %10s %8s\n", "---", "------------", "----", "-----")
	for i, prefix := range prefixes {
		if i >= 30 {
			break
		}
		parts := strings.Split(prefix.prefix, ":")
		if len(parts) == 2 {
			fmt.Printf("%-4s %-20s %10s %8d\n",
				parts[0], "0x"+parts[1], formatBytes(prefix.size), prefix.count)
		}
	}

	return iter.Error()
}

// findCommonPrefix finds the common prefix between two strings - O(min(len1, len2))
func findCommonPrefix(str1, str2 string) string {
	minLen := len(str1)
	if len(str2) < minLen {
		minLen = len(str2)
	}

	for i := 0; i < minLen; i++ {
		if str1[i] != str2[i] {
			return str1[:i]
		}
	}

	return str1[:minLen]
}

func truncateKey(key string, maxLen int) string {
	if len(key) <= maxLen {
		return key
	}
	if maxLen <= 3 {
		return key[:maxLen]
	}
	return key[:maxLen-3] + "..."
}

func truncateWithMessage(content string, maxLen int, full bool, contentType string) (string, bool) {
	if full || len(content) <= maxLen {
		return content, false
	}
	if maxLen <= 3 {
		return content[:maxLen], true
	}
	return content[:maxLen-3] + "...", true
}

func decodeHexPrefix(hexStr string) string {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "invalid"
	}

	// Convert to string and replace non-printable characters
	decoded := make([]rune, 0, len(bytes))
	for _, b := range bytes {
		if b >= 32 && b <= 126 { // printable ASCII range
			decoded = append(decoded, rune(b))
		} else {
			decoded = append(decoded, '.')
		}
	}

	result := string(decoded)
	if result == "" {
		return "binary"
	}
	return result
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
