package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoparse/ast"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	stableMarshalerAllOptName             = "(gogoproto.stable_marshaler_all)"
	expectedStableMarshalerAllOptionValue = "true"
	gogoImportName                        = "gogoproto/gogo.proto"
)

type protoFileStat struct {
	pkgSource        *ast.SourcePos
	lastOptSource    *ast.SourcePos
	lastImportSource *ast.SourcePos
	hasGogoImport    bool
}

var (
	flagFixName      = "fix"
	flagFixShorthand = "f"
	flagFixValue     = false
	flagFixUsage     = "If present, protocheck will add the 'gogoproto.stable_marshaler_all' option to files which were discovered to be unstable."

	unstableCmd = &cobra.Command{
		Use:   "unstable [flags]",
		Short: "Recursively list or fix all protobuf files which omit the 'stable_marshaler_all' option.",
		RunE:  runUnstable,
	}

	protoParser = protoparse.Parser{
		IncludeSourceCodeInfo: true,
	}

	stableMarshalerAllOptionSource = fmt.Sprintf(`option %s = %s;`, stableMarshalerAllOptName, expectedStableMarshalerAllOptionValue)
	gogoImportSource               = fmt.Sprintf(`import "%s";`, gogoImportName)
)

func init() {
	unstableCmd.Flags().BoolVarP(&flagFixValue, flagFixName, flagFixShorthand, flagFixValue, flagFixUsage)
	rootCmd.AddCommand(unstableCmd)
}

func runUnstable(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	logger := polylog.Ctx(ctx)

	unstableProtoFilesByPath := make(map[string]*protoFileStat)

	logger.Info().Msgf("Recursively checking for files matching %q in %q", flagFileIncludePatternValue, flagRootValue)

	// 1. Walk the directory tree.
	// 2. For each matching file:
	//   2a. Add it to the unstableProtoFilesByPath.
	//   2b. Walk the AST of the matching file.
	//   2c. Exclude files which contain the stable_marshaler_all option.
	if pathWalkErr := filepath.Walk(
		flagRootValue,
		forEachMatchingFileWalkFn(
			flagFileIncludePatternValue,
			findUnstableProtosInFileFn(ctx, unstableProtoFilesByPath),
		),
	); pathWalkErr != nil {
		logger.Error().Err(pathWalkErr)
		os.Exit(CodePathWalkErr)
	}

	if len(unstableProtoFilesByPath) == 0 {
		logger.Info().Msg("No unstable marshaler proto files found! 🥳🙌")
		return nil
	}

	// Fix discovered unstable marshaler proto files if the fix flag is set.
	if flagFixValue {
		runFixUnstable(ctx, unstableProtoFilesByPath)
	}

	if len(unstableProtoFilesByPath) == 0 {
		return nil
	}

	logger.Info().Msgf("Found %d unstable marshaler proto files:", len(unstableProtoFilesByPath))

	for unstableProtoFile := range unstableProtoFilesByPath {
		logger.Info().Msgf("\t%s", unstableProtoFile)
	}

	return nil
}

func runFixUnstable(ctx context.Context, unstableProtoFilesByPath map[string]*protoFileStat) {
	logger := polylog.Ctx(ctx)
	logger.Info().Msg("Fixing unstable marshaler proto files...")

	var fixedProtoFilePaths []string
	for unstableProtoFile, protoStat := range unstableProtoFilesByPath {
		if protoStat != nil {
			if insertErr := insertStableMarshalerAllOption(unstableProtoFile, protoStat); insertErr != nil {
				logger.Error().Err(insertErr).Msgf("unable to fix unstable marshaler proto file: %q", unstableProtoFile)
				continue
			}

			fixedProtoFilePaths = append(fixedProtoFilePaths, unstableProtoFile)
			delete(unstableProtoFilesByPath, unstableProtoFile)
		}
	}

	logger.Info().Msgf("Fixed the %d unstable marshaler proto files:", len(fixedProtoFilePaths))

	for _, protoFilePath := range fixedProtoFilePaths {
		logger.Info().Msgf("\t%s", protoFilePath)
	}
}

// forEachMatchingFileWalkFn returns a filepath.WalkFunc which does the following:
// 1. Iterates over files matching fileNamePattern against each file name.
// 2. For matching files, it calls fileMatchedFn with the respective path.
func forEachMatchingFileWalkFn(
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

// findUnstableProtosInFileFn returns a function which is expected to be called for
// each proto file found. The function walks that file's AST and excludes it from
// the list of unstable files if it contains the stable_marshaler_all option.
func findUnstableProtosInFileFn(
	ctx context.Context,
	unstableProtoFilesByPath map[string]*protoFileStat,
) func(path string) {
	logger := polylog.Ctx(ctx)

	return func(protoFilePath string) {
		// Parse the .proto file into file nodes.
		// NB: MUST use #ParseToAST instead of #ParseFiles to get source positions.
		protoAST, parseErr := protoParser.ParseToAST(protoFilePath)
		if parseErr != nil {
			logger.Error().Err(parseErr).Msgf("Unable to parse proto file: %q", protoFilePath)
		}

		// Iterate through the file nodes and build a protoFileStat for each file.
		// NB: There should only be one file node per file.
		for _, fileNode := range protoAST {
			protoStat := newProtoFileStat(fileNode)

			// Add all proto files to unstableProtoFilePaths by default. If a file
			// has a stable_marshaler_all option, that file protoFilePath will be
			// removed from the map when the option is traversed (found).
			unstableProtoFilesByPath[protoFilePath] = protoStat

			ast.Walk(
				fileNode,
				excludeFileIfStableVisitFn(
					ctx,
					protoFilePath,
					unstableProtoFilesByPath,
				),
			)
		}
	}
}

// excludeStableMarshalersVisitFn returns an ast.VisitFunc which removes proto files
// from unstableProtoFilesByPath if they contain the stable_marshaler_all option, and
// it is set to true.
func excludeFileIfStableVisitFn(
	ctx context.Context,
	protoFilePath string,
	unstableProtoFilesByPath map[string]*protoFileStat,
) ast.VisitFunc {
	logger := polylog.Ctx(ctx)

	return func(n ast.Node) (bool, ast.VisitFunc) {
		optNode, optNodeOk := n.(*ast.OptionNode)
		if !optNodeOk {
			return true, nil
		}

		optSrc := optNode.Start()

		optName, optNameOk := getOptNodeName(optNode)
		if !optNameOk {
			logger.Warn().Msgf(
				"unable to extract option name from option node at %s:%d:%d",
				protoFilePath, optSrc.Line, optSrc.Col,
			)
			return true, nil
		}

		if optName != stableMarshalerAllOptName {
			// Not the option we're looking for, continue traversing...
			return true, nil
		}

		optValueNode := optNode.GetValue().Value()
		optValue, ok := optValueNode.(ast.Identifier)
		if !ok {
			logger.Error().Msgf(
				"unable to cast option value to ast.Identifier for option %q, got: %T at %s:%d:%d",
				optName, optValueNode, protoFilePath, optSrc.Line, optSrc.Col,
			)
			return true, nil
		}

		if optValue != "true" {
			// Not the value we're looking for, continue traversing...
			logger.Warn().Msgf(
				"discovered an unstable_marshaler_all option with unexpected value %q at %s:%d:%d",
				optValue, protoFilePath, optSrc.Line, optSrc.Col,
			)
			return true, nil
		}

		// Remove stable proto file from unstableProtoFilesByPath.
		delete(unstableProtoFilesByPath, protoFilePath)

		// Stop traversing the AST after finding the stable_marshaler_all option.
		// We only expect one stable_marshaler_all option per file.
		return false, nil
	}
}

// getOptNodeName returns the name of the option node as a string and a boolean
// indicating whether the name was successfully extracted.
func getOptNodeName(optNode *ast.OptionNode) (optName string, ok bool) {
	optNameNode, optNameNodeOk := optNode.GetName().(*ast.OptionNameNode)
	if !optNameNodeOk {
		return "", false
	}

	if len(optNameNode.Parts) < 1 {
		return "", false
	}

	for i, optNamePart := range optNameNode.Parts {
		// Only insert delimiters if there is more than one part.
		if i > 0 {
			optName += "."
		}
		optName += optNamePart.Value()
	}

	return optName, true
}

func newProtoFileStat(fileNode *ast.FileNode) *protoFileStat {
	var (
		pkgNode         *ast.PackageNode
		lastOptionNode  *ast.OptionNode
		lastImportNode  *ast.ImportNode
		foundGogoImport bool
	)

	for _, n := range fileNode.Children() {
		switch node := n.(type) {
		case *ast.PackageNode:
			pkgNode = node
		case *ast.ImportNode:
			lastImportNode = node

			if node.Name.AsString() == gogoImportName {
				foundGogoImport = true
			}
		case *ast.OptionNode:
			lastOptionNode = node
		}
	}

	protoStat := &protoFileStat{
		pkgSource:     pkgNode.Start(),
		hasGogoImport: foundGogoImport,
	}
	if lastOptionNode != nil {
		protoStat.lastOptSource = lastOptionNode.Start()
	}
	if lastImportNode != nil {
		protoStat.lastImportSource = lastImportNode.Start()
	}

	return protoStat
}

func insertStableMarshalerAllOption(protoFilePath string, protoFile *protoFileStat) (err error) {
	var (
		importInsertLine,
		importInsertCol,
		optionInsertLine,
		optionInsertCol,
		numInsertedLines int

		optionLine = stableMarshalerAllOptionSource
		importLine = gogoImportSource
	)

	if protoFile.lastOptSource == nil {
		optionInsertLine = protoFile.pkgSource.Line + 1
		optionInsertCol = protoFile.pkgSource.Col
		optionLine += "\n"
		numInsertedLines += 2
	} else {
		optionInsertLine = protoFile.lastOptSource.Line
		optionInsertCol = protoFile.lastOptSource.Col
		numInsertedLines++
	}

	if err = insertLine(
		protoFilePath,
		optionInsertLine,
		optionInsertCol,
		optionLine,
	); err != nil {
		return err
	}

	if protoFile.hasGogoImport {
		return nil
	}

	if protoFile.lastImportSource == nil {
		importInsertLine = optionInsertLine + 1 + numInsertedLines
		importInsertCol = optionInsertCol
		importLine += "\n"
	} else {
		importInsertLine = protoFile.lastImportSource.Line + numInsertedLines
		importInsertCol = protoFile.lastImportSource.Col
	}

	return insertLine(
		protoFilePath,
		importInsertLine,
		importInsertCol,
		importLine,
	)
}

func insertLine(filePath string, lineNumber int, columnNumber int, textToInsert string) error {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the file into a slice of strings (each string is a line)
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	// Check if the line number is within the range of lines in the file
	if lineNumber < 1 || lineNumber > len(lines) {
		return fmt.Errorf("line number %d is out of range", lineNumber)
	}

	// Create the new line with the specified amount of leading whitespace
	whitespace := strings.Repeat(" ", columnNumber-1)
	newLine := whitespace + textToInsert

	// Insert the new line after the specified line
	lineIndex := lineNumber - 1
	lines = append(lines[:lineIndex+1], append([]string{newLine}, lines[lineIndex+1:]...)...)

	// Open the file for writing
	file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the modified lines back to the file
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
