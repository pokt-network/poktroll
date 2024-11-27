package protoast

import (
	"io/fs"
	"path/filepath"
)

// ForEachMatchingFileWalkFn returns a filepath.WalkFunc which does the following:
// 1. Iterates over files matching fileNamePattern against each file name.
// 2. For matching files, it calls fileMatchedFn with the respective path.
func ForEachMatchingFileWalkFn(
	fileNamePattern string,
	fileMatchedFn func(path string),
) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore directories
		if info.IsDir() {
			return nil
		}

		matched, matchErr := filepath.Match(fileNamePattern, info.Name())
		if matchErr != nil {
			return matchErr
		}

		if matched {
			fileMatchedFn(path)
		}
		return nil
	}
}
