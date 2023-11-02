package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// defaultArgs are always passed to goimports.
// -w: write result to (source) file instead of stdout
// -local: put imports beginning with this string after 3rd-party packages (comma-separated list)
// (see: goimports -h)
var defaultArgs = []string{"-w", "-local", "github.com/pokt-network/poktroll"}

func main() {
	root := "."
	var filesToProcess []string

	// Walk the file system and accumulate matching files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == root {
			return nil
		}

		// Skip directories that start with a period
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Add eligible Go files to the list
		if filepath.Ext(path) == ".go" &&
			// Ignore generated protobuf & protobuf gateway files.
			!(strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".pb.gw.go")) {

			// Ignore files that can't be goimport'd due to ignite compatibility.
			isEmptyImport, err := containsEmptyImportScaffold(path)
			if err != nil {
				panic(err)
			}
			if !isEmptyImport {
				filesToProcess = append(filesToProcess, path)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error processing files: %s\n", err)
		return
	}

	// Run goimports on all accumulated files
	if len(filesToProcess) > 0 {
		cmd := exec.Command("goimports", append(defaultArgs, filesToProcess...)...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed running goimports: %v\n", err)
		}
	}
}

/*
	containsEmptyImportScaffold checks if the go file at goSrcPath contains an
	import statement like the following:

import (
// this line is used by starport scaffolding # genesis/types/import
)
*/
func containsEmptyImportScaffold(goSrcPath string) (isEmptyImport bool, _ error) {
	file, err := os.Open(goSrcPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(importBlockSplit)

	for scanner.Scan() {
		block := scanner.Text()
		if strings.Contains(block, "// this line is used by starport scaffolding # genesis/types/import") {
			return true, nil
		}
	}

	if scanner.Err() != nil {
		return false, scanner.Err()
	}

	return false, nil
}

// importBlockSplit is a split function intended to be used with bufio.Scanner
// to extract the contents of a multi-line go import block.
func importBlockSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Search for the beginning of the import block
	startIdx := bytes.Index(data, []byte("import ("))
	if startIdx == -1 {
		return 0, nil, nil
	}

	// Search for the end of the import block from the start index
	endIdx := bytes.Index(data[startIdx:], []byte(")"))
	if endIdx == -1 {
		return 0, nil, nil
	}

	// Return the entire import block, including "import (" and ")"
	return startIdx + endIdx + 1, data[startIdx : startIdx+endIdx+1], nil
}
