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
		"run",
		"--timeout=15m",
		"--build-tags=e2e,test,integration",
	}
	defaultIncludeFilters = []filters.FilterFn{
		filters.PathMatchesGoExtension,
	}
	defaultExcludeFilters = []filters.FilterFn{
		filters.PathMatchesProtobufGo,
		filters.PathMatchesProtobufGatewayGo,
		filters.PathMatchesMockGo,
		filters.ContentMatchesEmptyImportScaffold,
	}
)

func main() {
	root := "."
	var (
		filesToProcessWithGCI    []string
		filesToProcessWithoutGCI []string
	)

	// Walk the file system and accumulate matching files
	err := filepath.Walk(root, walkRepoRootFn(
		root,
		defaultIncludeFilters,
		defaultExcludeFilters,
		&filesToProcessWithGCI,
	))
	if err != nil {
		fmt.Printf("Error processing files: %s\n", err)
		return
	}

	// Run golangci-lint on all files that don't have a scaffold comment in
	// their import block - so it can be run normally with gci
	if len(filesToProcessWithGCI) > 0 {
		args := append(defaultArgs, "--disable=gci")
		cmd := exec.Command("golangci-lint", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Output: %s\nFailed running golangci-lint with gci: %v\n", out, err)
		}
	}

	// Run golangci-lint on all files that don't have a scaffold comment in
	// their import block - so it can be run normally with gci
	if len(filesToProcessWithoutGCI) > 0 {
		cmd := exec.Command("golangci-lint", append(defaultArgs..., "--disable=gci")...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Output: %s\nFailed running golangci-lint without gci: %v\n", out, err)
		}
	}
}

func walkRepoRootFn(
	rootPath string,
	includeFilters []filters.FilterFn,
	excludeFilters []filters.FilterFn,
	filesToProcessWithGCI *[]string,
	filesToProcessWithoutGCI *[]string,
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

		// Check if the file contains a scaffold comment in the import block,
		// skip it and tell golangci-lin to disable the gci linter
		containsImportScaffoldComment, err := filters.ImportBlockContainsScaffoldComment(path)
		if err != nil {
			panic(err)
		}
		if containsImportScaffoldComment {
			enableGCI.CompareAndSwap(false, true)
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

		if containsImportScaffoldComment {
			*filesToProcessWithoutGCI = append(*filesToProcessWithoutGCI, path)
		}

		*filesToProcessWithGCI = append(*filesToProcessWithGCI, path)

		return nil
	}
}
