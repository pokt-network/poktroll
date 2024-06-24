package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pokt-network/poktroll/tools/scripts/goimports/filters"
)

// defaultArgs are always passed to goimports.
// -w: write result to (source) file instead of stdout
// -local: put imports beginning with this string after 3rd-party packages (comma-separated list)
// (see: goimports -h for more info)
var (
	defaultArgs           = []string{"-w", "-local", "github.com/pokt-network/poktroll"}
	defaultIncludeFilters = []filters.FilterFn{
		filters.PathMatchesGoExtension,
	}
	defaultExcludeFilters = []filters.FilterFn{
		filters.PathMatchesProtobufGo,
		filters.PathMatchesProtobufGatewayGo,
		filters.PathMatchesMockGo,
		filters.PathMatchesTestGo,
		filters.PathMatchesPulsarGo,
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

	// Run goimports on all accumulated files
	if len(filesToProcess) > 0 {
		cmd := exec.Command("goimports", append(defaultArgs, filesToProcess...)...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed running goimports: %v\n", err)
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
