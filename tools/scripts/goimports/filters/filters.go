// The filters package contains functions that can be used to filter file paths.

package filters

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

const igniteScaffoldComment = "// this line is used by starport scaffolding"

var (
	importStart = []byte("import (")
	importEnd   = []byte(")")
)

// FilterFn is a function that returns true if the given path matches the
// filter's criteria.
type FilterFn func(path string) (bool, error)

// PathMatchesGoExtension matches go source files.
func PathMatchesGoExtension(path string) (bool, error) {
	return filepath.Ext(path) == ".go", nil
}

// PathMatchesProtobufGo matches generated protobuf go source files.
func PathMatchesProtobufGo(path string) (bool, error) {
	return strings.HasSuffix(path, ".pb.go"), nil
}

// PathMatchesProtobufGatewayGo matches generated protobuf gateway go source files.
func PathMatchesProtobufGatewayGo(path string) (bool, error) {
	return strings.HasSuffix(path, ".pb.gw.go"), nil
}

// PathMatchesMockGo matches generated mock go source files.
func PathMatchesMockGo(path string) (bool, error) {
	return (strings.HasSuffix(path, "_mock.go") || strings.Contains(path, "/gomock_reflect_")), nil
}

// PathMatchesTestGo matches go test files.
func PathMatchesTestGo(path string) (bool, error) {
	return strings.HasSuffix(path, "_test.go"), nil
}

func PathMatchesPulsarGo(path string) (bool, error) {
	return strings.HasSuffix(path, ".pulsar.go"), nil
}

// ContentMatchesEmptyImportScaffold matches files that can't be goimport'd due
// to ignite incompatibility.
func ContentMatchesEmptyImportScaffold(path string) (bool, error) {
	return containsEmptyImportScaffold(path)
}

// containsEmptyImportScaffold checks if the go file at goSrcPath contains an
// import statement like the following:
//
// import (
// // this line is used by starport scaffolding ...
// )
func containsEmptyImportScaffold(goSrcPath string) (isEmptyImport bool, _ error) {
	file, err := os.Open(goSrcPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// The default buffer size is 64KB, which is insufficient.
	// Set a larger buffer size (e.g., 1 MB) to avoid the following error:
	// bufio.Scanner: token too long
	const maxBufferSize = 1024 * 1024 // 1 MB
	buf := make([]byte, maxBufferSize)
	scanner.Buffer(buf, maxBufferSize)

	scanner.Split(importBlockSplit)

	for scanner.Scan() {
		trimmedImportBlock := strings.Trim(scanner.Text(), "\n\t")
		if strings.HasPrefix(trimmedImportBlock, igniteScaffoldComment) {
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
func importBlockSplit(data []byte, _ bool) (advance int, token []byte, err error) {
	// Search for the beginning of the import block
	startIdx := bytes.Index(data, importStart)
	if startIdx == -1 {
		return 0, nil, nil
	}

	// Search for the end of the import block from the start index
	endIdx := bytes.Index(data[startIdx:], importEnd)
	if endIdx == -1 {
		return 0, nil, nil
	}

	// Return the entire import block, including "import (" and ")"
	importBlock := data[startIdx+len(importStart) : startIdx-len(importEnd)+endIdx+1]
	return startIdx + endIdx + 1, importBlock, nil
}
