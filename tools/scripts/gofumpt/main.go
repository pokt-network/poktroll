package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pokt-network/poktroll/tools/scripts/gci/filters"
)

var (
	defaultArgs = []string{
		"-w",
	}
	defaultIncludeFilters = []filters.FilterFn{
		filters.PathMatchesGoExtension,
	}
	defaultExcludeFilters = []filters.FilterFn{
		filters.PathMatchesProtobufGo,
		filters.PathMatchesProtobufGatewayGo,
		filters.PathMatchesMockGo,
		filters.PathMatchesTestGo,
		filters.ContentMatchesEmptyImportScaffold,
	}
)

func main() {
	root := "."
	var filesToProcess []string

	// Walk the file system and accumulate matching files
	err := filepath.Walk(root, walkRepoRootFn(
		root,
		defaultIncludeFilters,
		defaultExcludeFilters,
		&filesToProcess,
	))
	if err != nil {
		fmt.Printf("Error processing files: %s\n", err)
		return
	}

	// Run gci on all accumulated files - this writes changes in place
	if len(filesToProcess) > 0 {
		cmd := exec.Command("gofumpt", append(defaultArgs, filesToProcess...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Output: %s\nFailed running gci: %v\n", out, err)
		}
	}
}

func walkRepoRootFn(
	rootPath string,
	includeFilters []filters.FilterFn,
	excludeFilters []filters.FilterFn,
	filesToProcess *[]string,
) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Don't process the root directory but don't skip it either; that would
		// exclude everything.
		if info.Name() == rootPath {
			return nil
		}

		// No need to process directories
		if info.IsDir() {
			// Skip directories that start with a period
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Don't process paths which don't match any include filter.
		var shouldIncludePath bool
		for _, includeFilter := range includeFilters {
			pathMatches, err := includeFilter(path)
			if err != nil {
				panic(err)
			}

			if pathMatches {
				shouldIncludePath = true
				break
			}
		}
		if !shouldIncludePath {
			return nil
		}

		// Don't process paths which match any exclude filter.
		var shouldExcludePath bool
		for _, excludeFilter := range excludeFilters {
			pathMatches, err := excludeFilter(path)
			if err != nil {
				panic(err)
			}

			if pathMatches {
				shouldExcludePath = true
				break
			}
		}
		if shouldExcludePath {
			return nil
		}

		*filesToProcess = append(*filesToProcess, path)

		return nil
	}
}
