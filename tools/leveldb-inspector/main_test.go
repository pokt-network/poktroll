package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func TestTruncateKey(t *testing.T) {
	tests := []struct {
		key      string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact_length", 12, "exact_length"},
		{"this_is_too_long", 10, "this_is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"a", 1, "a"},
		{"ab", 1, "a"},
	}

	for _, test := range tests {
		result := truncateKey(test.key, test.maxLen)
		if result != test.expected {
			t.Errorf("truncateKey(%q, %d) = %q, expected %q", test.key, test.maxLen, result, test.expected)
		}
	}
}

func TestDecodeHexPrefix(t *testing.T) {
	tests := []struct {
		hex      string
		expected string
	}{
		{"626c6f63", "bloc"},
		{"636f696e", "coin"},
		{"74782e66", "tx.f"},
		{"00010203", "...."},
		{"", "binary"},
		{"invalid", "invalid"},
	}

	for _, test := range tests {
		result := decodeHexPrefix(test.hex)
		if result != test.expected {
			t.Errorf("decodeHexPrefix(%q) = %q, expected %q", test.hex, result, test.expected)
		}
	}
}

func TestGetSizeRange(t *testing.T) {
	tests := []struct {
		size     int
		expected string
	}{
		{0, "0-31"},
		{31, "0-31"},
		{32, "32-63"},
		{63, "32-63"},
		{64, "64-127"},
		{127, "64-127"},
		{128, "128-255"},
		{255, "128-255"},
		{256, "256-511"},
		{511, "256-511"},
		{512, "512-1023"},
		{1023, "512-1023"},
		{1024, "1024-2047"},
		{2047, "1024-2047"},
		{2048, "2048-4095"},
		{4095, "2048-4095"},
		{4096, "4096+"},
		{10000, "4096+"},
	}

	for _, test := range tests {
		result := getSizeRange(test.size)
		if result != test.expected {
			t.Errorf("getSizeRange(%d) = %q, expected %q", test.size, result, test.expected)
		}
	}
}

func TestGetSizeRangeValue(t *testing.T) {
	tests := []struct {
		sizeRange string
		expected  int
	}{
		{"0-31", 31},
		{"32-63", 63},
		{"64-127", 127},
		{"128-255", 255},
		{"256-511", 511},
		{"512-1023", 1023},
		{"1024-2047", 2047},
		{"2048-4095", 4095},
		{"4096+", 999999},
		{"unknown", 0},
	}

	for _, test := range tests {
		result := getSizeRangeValue(test.sizeRange)
		if result != test.expected {
			t.Errorf("getSizeRangeValue(%q) = %d, expected %d", test.sizeRange, result, test.expected)
		}
	}
}

func TestFindCommonPrefix(t *testing.T) {
	tests := []struct {
		str1     string
		str2     string
		expected string
	}{
		{"hello", "help", "hel"},
		{"test", "test", "test"},
		{"abc", "def", ""},
		{"", "anything", ""},
		{"anything", "", ""},
		{"block_events", "block_height", "block_"},
		{"tx.fee", "tx.hash", "tx."},
	}

	for _, test := range tests {
		result := findCommonPrefix(test.str1, test.str2)
		if result != test.expected {
			t.Errorf("findCommonPrefix(%q, %q) = %q, expected %q", test.str1, test.str2, result, test.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{5368709120, "5.0 GB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %q, expected %q", test.bytes, result, test.expected)
		}
	}
}

// createTestDB creates a temporary LevelDB for testing
func createTestDB(t *testing.T) *leveldb.DB {
	stor := storage.NewMemStorage()
	db, err := leveldb.Open(stor, nil)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return db
}

func TestCollectStats(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Add test data
	testData := map[string]string{
		"block_events_1": "value1",
		"block_events_2": "value2",
		"coin_spent":     "larger_value_here",
		"tx.fee":         "small",
		"message.test":   "msg",
	}

	for key, value := range testData {
		err := db.Put([]byte(key), []byte(value), nil)
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}

	stats, err := collectStats(db)
	if err != nil {
		t.Fatalf("collectStats failed: %v", err)
	}

	// Verify basic stats
	if stats.TotalKeys != int64(len(testData)) {
		t.Errorf("Expected %d keys, got %d", len(testData), stats.TotalKeys)
	}

	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}

	// Verify prefixes were collected
	if len(stats.KeyPrefixes) == 0 {
		t.Error("Expected key prefixes to be collected")
	}

	// Verify LCP was calculated
	if len(stats.LongestCommonPrefixes) == 0 {
		t.Error("Expected longest common prefixes to be calculated")
	}

	// Check that block_events keys share a common prefix
	blockPrefix := "626c6f63" // hex for "bloc"
	if lcp, exists := stats.LongestCommonPrefixes[blockPrefix]; exists {
		if !strings.HasPrefix(lcp, "block_events") {
			t.Errorf("Expected LCP for block prefix to start with 'block_events', got %q", lcp)
		}
	}
}

func TestDisplayStatsJSON(t *testing.T) {
	stats := &Stats{
		TotalKeys:    100,
		TotalSize:    1024,
		MaxKeySize:   50,
		MaxValueSize: 200,
		LargestKey:   "test_key",
		LargestValue: "test_value",
		KeyPrefixes: map[string]int64{
			"626c6f63": 80,
			"636f696e": 20,
		},
		LongestCommonPrefixes: map[string]string{
			"626c6f63": "block_events",
			"636f696e": "coin",
		},
		KeySizes: map[string]int64{
			"32-63": 90,
			"0-31":  10,
		},
		ValueSizes: map[string]int64{
			"0-31":  80,
			"32-63": 20,
		},
	}

	// Capture JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayStatsJSON(stats, 10)
	if err != nil {
		t.Fatalf("displayStatsJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify key fields exist
	expectedFields := []string{"total_keys", "total_logical_size", "top_prefixes", "key_size_distribution", "value_size_distribution"}
	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("JSON output missing field: %s", field)
		}
	}

	// Verify top_prefixes structure
	if prefixes, ok := result["top_prefixes"].([]interface{}); ok {
		if len(prefixes) == 0 {
			t.Error("Expected non-empty top_prefixes array")
		}
		if firstPrefix, ok := prefixes[0].(map[string]interface{}); ok {
			expectedPrefixFields := []string{"hex_prefix", "common_prefix", "count", "percentage"}
			for _, field := range expectedPrefixFields {
				if _, exists := firstPrefix[field]; !exists {
					t.Errorf("Prefix object missing field: %s", field)
				}
			}
		}
	} else {
		t.Error("top_prefixes should be an array")
	}
}

func TestDisplayStatsCSV(t *testing.T) {
	stats := &Stats{
		TotalKeys:    100,
		TotalSize:    1024,
		MaxKeySize:   50,
		MaxValueSize: 200,
		KeyPrefixes: map[string]int64{
			"626c6f63": 80,
			"636f696e": 20,
		},
		LongestCommonPrefixes: map[string]string{
			"626c6f63": "block_events",
			"636f696e": "coin",
		},
		KeySizes: map[string]int64{
			"32-63": 90,
			"0-31":  10,
		},
		ValueSizes: map[string]int64{
			"0-31":  80,
			"32-63": 20,
		},
	}

	// Capture CSV output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayStatsCSV(stats, 10)
	if err != nil {
		t.Fatalf("displayStatsCSV failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	output := buf.String()

	// Verify CSV structure
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		t.Errorf("Expected at least 5 lines in CSV output, got %d", len(lines))
	}

	// Verify headers
	if !strings.Contains(lines[0], "metric,value") {
		t.Error("Expected CSV to start with metric,value header")
	}

	// Verify prefix section exists
	prefixFound := false
	for _, line := range lines {
		if strings.Contains(line, "prefix,0x626c6f63") {
			prefixFound = true
			break
		}
	}
	if !prefixFound {
		t.Error("Expected to find prefix data in CSV output")
	}

	// Verify size distribution sections exist
	keySizeFound := false
	valueSizeFound := false
	for _, line := range lines {
		if strings.Contains(line, "key_size,") {
			keySizeFound = true
		}
		if strings.Contains(line, "value_size,") {
			valueSizeFound = true
		}
	}
	if !keySizeFound {
		t.Error("Expected to find key_size data in CSV output")
	}
	if !valueSizeFound {
		t.Error("Expected to find value_size data in CSV output")
	}
}

// Integration test with real file-based LevelDB
func TestIntegrationWithFileDB(t *testing.T) {
	// Create temporary directory for test DB
	tmpDir, err := os.MkdirTemp("", "leveldb_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create and populate test database
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testData := map[string]string{
		"block_events_key1": "small_value",
		"block_events_key2": "another_small_value",
		"coin_spent_tx1":    "medium_value_here_with_more_data",
		"tx.fee_hash":       "tiny",
		"message.type":      "msg_data",
	}

	for key, value := range testData {
		err := db.Put([]byte(key), []byte(value), nil)
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}
	db.Close()

	// Test opening and analyzing the database
	db2, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db2.Close()

	stats, err := collectStats(db2)
	if err != nil {
		t.Fatalf("collectStats failed: %v", err)
	}

	// Verify results
	if stats.TotalKeys != int64(len(testData)) {
		t.Errorf("Expected %d keys, got %d", len(testData), stats.TotalKeys)
	}

	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}

	// Verify block_events prefix has correct LCP
	blockPrefix := "626c6f63" // hex for "bloc"
	if lcp, exists := stats.LongestCommonPrefixes[blockPrefix]; exists {
		if !strings.HasPrefix(lcp, "block_events") {
			t.Errorf("Expected LCP for block prefix to start with 'block_events', got %q", lcp)
		}
	}

	// Test that prefix counts are correct
	if count, exists := stats.KeyPrefixes[blockPrefix]; exists {
		if count != 2 {
			t.Errorf("Expected 2 keys with block prefix, got %d", count)
		}
	} else {
		t.Error("Expected block prefix to exist in key prefixes")
	}
}

func TestOpenDBErrors(t *testing.T) {
	// Test with non-existent path
	_, err := openDB("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent database path")
	}

	// Test with invalid path (not a directory)
	tmpFile, err := os.CreateTemp("", "not_a_db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = openDB(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid database path")
	}
}

func TestConfigurableTopPrefixes(t *testing.T) {
	stats := &Stats{
		TotalKeys:    100,
		TotalSize:    1024,
		MaxKeySize:   50,
		MaxValueSize: 200,
		KeyPrefixes: map[string]int64{
			"626c6f63": 80,
			"636f696e": 15,
			"74782e66": 3,
			"6d657373": 2,
		},
		LongestCommonPrefixes: map[string]string{
			"626c6f63": "block_events",
			"636f696e": "coin",
			"74782e66": "tx.f",
			"6d657373": "mess",
		},
		KeySizes: map[string]int64{
			"32-63": 90,
			"0-31":  10,
		},
		ValueSizes: map[string]int64{
			"0-31":  80,
			"32-63": 20,
		},
	}

	// Test with top 2 prefixes
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayStatsTable(stats, false, 2)
	if err != nil {
		t.Fatalf("displayStatsTable failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	output := buf.String()

	// Should show top 2 prefixes and remaining count
	if !strings.Contains(output, "Top 2 Key Prefixes") {
		t.Error("Expected 'Top 2 Key Prefixes' in output")
	}

	// Should show remaining prefixes line
	if !strings.Contains(output, "(remaining prefixes)") {
		t.Error("Expected '(remaining prefixes)' line in output")
	}

	// Should show correct remaining count (3 + 2 = 5 keys)
	if !strings.Contains(output, "5") {
		t.Error("Expected remaining count of 5 in output")
	}

	// Should show "2 more prefixes"
	if !strings.Contains(output, "2 more prefixes") {
		t.Error("Expected '2 more prefixes' in output")
	}
}
