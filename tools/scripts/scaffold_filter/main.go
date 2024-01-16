package main

import (
	"fmt"
	"os"

	"github.com/pokt-network/poktroll/tools/scripts/gci/filters"
)

var defaultExcludeFilters = []filters.FilterFn{
	filters.ContentMatchesEmptyImportScaffold,
	filters.ImportBlockContainsScaffoldComment,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <path>...\n", os.Args[0])
		os.Exit(1)
	}

	var filesToProcess []string
	// For each file passed as an argument check whether to exclude it
	// according to the filters, from a golangci-lint check. This is
	// purely because gci (the golanci-lint linter for imports) doesn't
	// support this type of filtering natively.
	for _, path := range os.Args[1:] {
		goFile, err := filters.PathMatchesGoExtension(path)
		if err != nil {
			fmt.Printf("Error processing file %s: %s\n", path, err)
			continue
		}
		if !goFile {
			continue
		}
		for _, excludeFilter := range defaultExcludeFilters {
			shouldExclude, err := excludeFilter(path)
			if err != nil {
				fmt.Printf("Error processing file %s: %s\n", path, err)
				continue
			}
			if !shouldExclude {
				filesToProcess = append(filesToProcess, path)
			}
		}
	}

	// Print all files to be processed by golangci-lint.
	// This is so gci doesn't remove the scaffold comments in
	// the import block for files scaffolded by ignite.
	for _, file := range filesToProcess {
		// We print the file names as this is a helper used in the
		// testing workflow and the output of this file is appended
		// to a file from which golangci-lint will be run on, in CI.
		fmt.Println(file)
	}
}
