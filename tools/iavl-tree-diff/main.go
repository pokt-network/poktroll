package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/iavl"
	dbm "github.com/cosmos/iavl/db"
)

const (
	DefaultCacheSize int = 10000
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: iavl-tree-diff <db1 dir> <db2 dir>")
		os.Exit(1)
	}

	dbDir1 := os.Args[1]
	dbDir2 := os.Args[2]

	prefixes, err := getPrefixes(dbDir1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting prefixes from db1: %s\n", err)
		os.Exit(1)
	}

	for _, prefix := range prefixes {
		fmt.Printf("Checking prefix: %s\n", prefix)

		version1, err := getLatestVersion(dbDir1, prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting latest version from db1 for prefix %s: %s\n", prefix, err)
			continue
		}

		version2, err := getLatestVersion(dbDir2, prefix)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting latest version from db2 for prefix %s: %s\n", prefix, err)
			continue
		}

		lowerVersion := min(version1, version2)
		fmt.Printf("Using lower version: %d\n", lowerVersion)

		tree1, db1, err := ReadTree(dbDir1, lowerVersion, []byte(prefix))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading tree from db1 for prefix %s: %s\n", prefix, err)
			if db1 != nil {
				db1.Close()
			}
			continue
		}

		tree2, db2, err := ReadTree(dbDir2, lowerVersion, []byte(prefix))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading tree from db2 for prefix %s: %s\n", prefix, err)
			if db2 != nil {
				db2.Close()
			}
			db1.Close()
			continue
		}

		hash1 := tree1.Hash()
		hash2 := tree2.Hash()

		fmt.Printf("Tree hash from db1: %X\n", hash1)
		fmt.Printf("Tree hash from db2: %X\n", hash2)

		if !bytes.Equal(hash1, hash2) {
			fmt.Println("Hashes differ, checking for differences in keys/values...")

			diffKeys := findDifferences(tree1, tree2)
			for _, key := range diffKeys {
				val1, _ := tree1.Get(key)
				val2, _ := tree2.Get(key)
				fmt.Printf("Key: %s\n", key)
				fmt.Printf("Value in db1: %X\n", val1)
				fmt.Printf("Value in db2: %X\n", val2)
			}
		} else {
			fmt.Println("Tree hashes are identical.")
		}

		// Explicitly close the databases after each prefix comparison
		db1.Close()
		db2.Close()
	}
}

func getPrefixes(dir string) ([]string, error) {
	db, err := OpenDB(dir)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	prefixes := make(map[string]struct{})
	itr, err := db.Iterator(nil, nil)
	if err != nil {
		return nil, err
	}
	defer itr.Close()

	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		prefix := extractPrefix(key)
		if prefix != "" {
			prefixes[prefix] = struct{}{}
		}
	}

	var prefixList []string
	for prefix := range prefixes {
		prefixList = append(prefixList, prefix)
	}

	return prefixList, nil
}

func extractPrefix(key []byte) string {
	parts := bytes.SplitN(key, []byte("/"), 3)
	if len(parts) >= 2 && string(parts[0]) == "s" {
		subParts := bytes.SplitN(parts[1], []byte(":"), 2)
		if len(subParts) == 2 {
			return fmt.Sprintf("s/k:%s/", subParts[1])
		}
	}
	return ""
}

func getLatestVersion(dir string, prefix string) (int64, error) {
	tree, db, err := ReadTree(dir, 0, []byte(prefix))
	if err != nil {
		if db != nil {
			db.Close()
		}
		return 0, err
	}
	defer db.Close()

	return tree.Version(), nil
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func ReadTree(dir string, version int64, prefix []byte) (*iavl.MutableTree, dbm.DB, error) {
	db, err := OpenDB(dir)
	if err != nil {
		return nil, nil, err
	}
	if len(prefix) != 0 {
		db = dbm.NewPrefixDB(db, prefix)
	}

	tree := iavl.NewMutableTree(db, DefaultCacheSize, false, nil)

	_, err = tree.LoadVersion(version)
	if err != nil {
		return nil, db, fmt.Errorf("failed to load version %d: %w", version, err)
	}

	return tree, db, nil
}

func OpenDB(dir string) (dbm.DB, error) {
	switch {
	case strings.HasSuffix(dir, ".db"):
		dir = dir[:len(dir)-3]
	case strings.HasSuffix(dir, ".db/"):
		dir = dir[:len(dir)-4]
	default:
		return nil, fmt.Errorf("database directory must end with .db")
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	cut := strings.LastIndex(dir, "/")
	if cut == -1 {
		return nil, fmt.Errorf("cannot cut paths on %s", dir)
	}
	name := dir[cut+1:]
	db, err := dbm.NewGoLevelDB(name, dir[:cut])
	if err != nil {
		return nil, err
	}
	return db, nil
}

func findDifferences(tree1, tree2 *iavl.MutableTree) [][]byte {
	var differences [][]byte

	tree1.Iterate(func(key []byte, value []byte) bool {
		val2, err := tree2.Get(key)
		if err != nil || !bytes.Equal(value, val2) {
			differences = append(differences, key)
		}
		return false
	})

	return differences
}
