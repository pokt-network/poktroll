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

// TODO_OPTIMISATION: This could probably be optimised for speed.
func main() {
	root := "."
	var (
		packagesWithGCI          map[string][]string
		packagesWithoutGCI       map[string][]string
		filesToProcessWithGCI    []string
		filesToProcessWithoutGCI []string
	)

	// Create the maps
	packagesWithGCI = make(map[string][]string)
	packagesWithoutGCI = make(map[string][]string)

	// Walk the file system and accumulate matching files
	err := filepath.Walk(root, walkRepoRootFn(
		root,
		defaultIncludeFilters,
		defaultExcludeFilters,
		&filesToProcessWithGCI,
		&filesToProcessWithoutGCI,
	))
	if err != nil {
		fmt.Printf("Error processing files: %s\n", err)
		return
	}

	// Organise each file found by package
	if len(filesToProcessWithGCI) > 0 {
		for _, path := range filesToProcessWithGCI {
			pkgPath := filepath.Dir(path)
			packagesWithGCI[pkgPath] = append(packagesWithGCI[pkgPath], path)
		}
	}
	if len(filesToProcessWithoutGCI) > 0 {
		for _, path := range filesToProcessWithoutGCI {
			pkgPath := filepath.Dir(path)
			packagesWithoutGCI[pkgPath] = append(packagesWithoutGCI[pkgPath], path)
		}
	}

	errorFound := false
	totalOut := ""
	if len(packagesWithGCI) > 0 {
		fmt.Println("Linting files without scaffold comments in their import blocks...")
		// Run golangci-lint on all files that don't have a scaffold comment in
		// their import block - so it can be run normally with gci
		for _, path := range packagesWithGCI {
			args := append(defaultArgs, []string{"--enable=gci", "--enable=lll", "--enable=gofumpt"}...)
			cmd := exec.Command("golangci-lint", append(args, path...)...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Output: %s\nFailed running golangci-lint with gci: %v\n", out, err)
				errorFound = true
			}
			totalOut += fmt.Sprintf("%s\n", out)
		}
	}

	if len(packagesWithoutGCI) > 0 {
		fmt.Println("Linting files with scaffold comments in their import blocks...")
		// Run golangci-lint on all files that do have a scaffold comment in
		// their import block - so it can't be run with gci as it would remove it
		for _, path := range packagesWithoutGCI {
			args := append(defaultArgs, []string{"--enable=lll", "--enable=gofumpt"}...)
			cmd := exec.Command("golangci-lint", append(args, path...)...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Output: %s\nFailed running golangci-lint without gci: %v\n", out, err)
				errorFound = true
			}
			totalOut += fmt.Sprintf("%s\n", out)
		}
	}

	// Fail if any errors were found
	if errorFound {
		fmt.Println(totalOut)
		os.Exit(1)
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

		// Don't look in specific dirctories for files to lint
		if strings.HasPrefix(path, "ts-client") {
			return nil
		}
		if strings.HasPrefix(path, "docs") {
			return nil
		}
		if strings.HasPrefix(path, "bin") {
			return nil
		}
		if strings.HasPrefix(path, ".git") {
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

		// Check if the file contains a scaffold comment in the import block,
		// skip it and tell golangci-lin to disable the gci linter
		containsImportScaffoldComment, err := filters.ImportBlockContainsScaffoldComment(path)
		if err != nil {
			panic(err)
		}
		if containsImportScaffoldComment {
			*filesToProcessWithoutGCI = append(*filesToProcessWithoutGCI, path)
		} else {
			*filesToProcessWithGCI = append(*filesToProcessWithGCI, path)
		}

		return nil
	}
}
