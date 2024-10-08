package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoparse/ast"

	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

type ProtoField struct {
	Name        string
	Type        string
	Tag         string
	Options     string
	Description string
}

type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

const (
	destinationFile = "docusaurus/docs/protocol/governance/params.md"
	tableRowFmt     = "| `%-10s` | `%-10s` | `%-10s` | %-7s |\n"
)

var (
	paramsDocsTemplateStr string
	protoParser           = protoparse.Parser{
		IncludeSourceCodeInfo: true,
	}
	logger = polylog.DefaultContextLogger

	flagProtoRootPathValue string
)

func init() {
	flag.StringVar(&flagProtoRootPathValue, "proto-root", "./proto", "path to the proto files root directory; this directory will be walked, looking for Params.proto files")
	flag.Parse()

	paramsTempalteFile, err := os.ReadFile("./tools/scripts/docusaurus/params_template.md")
	if err != nil {
		polylog.DefaultContextLogger.Error().Err(err).Send()
		os.Exit(1)
	}
	paramsDocsTemplateStr = string(paramsTempalteFile) + "\n"
}

// writeContentToFile writes the given content to the specified file.
func writeContentToFile(file_path, content string) error {
	file, err := os.Create(file_path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the string to the file
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func prepareGovernanceParamsDocs(protoFilesRootDir string, template string) (string, error) {
	paramsFieldNodesByModule := make(map[string][]*ast.FieldNode)

	if pathWalkErr := filepath.Walk(
		protoFilesRootDir,
		forEachMatchingFileWalkFn(
			"params.proto",
			newCollectParamsFieldNodesFn(paramsFieldNodesByModule),
		),
	); pathWalkErr != nil {
		logger.Error().Err(pathWalkErr)
		os.Exit(1)
	}

	for moduleName, fieldNodes := range paramsFieldNodesByModule {
		for _, fieldNode := range fieldNodes {
			// Uncomment and concatenate the field's comment lines.
			comment := ""
			for commentIdx, commentLine := range fieldNode.LeadingComments() {
				var commentFmt = " %s"
				if commentIdx == 0 {
					commentFmt = "%s"
				}

				comment += fmt.Sprintf(commentFmt, strings.Trim(commentLine.Text, " /"))
			}
			logger.Info().Str("comment", comment).Send()

			// Extract the field's type information.
			fieldType := fieldNode.FldType.AsIdentifier()

			template += fmt.Sprintf(tableRowFmt, moduleName, fieldType, fieldNode.Name.Val, comment)
		}
	}

	return template, nil
}

func newCollectParamsFieldNodesFn(paramsFieldNodesByModule map[string][]*ast.FieldNode) func(protoFilePath string) {
	return func(protoFilePath string) {
		logger.Debug().Str("protoFilePath", protoFilePath).Send()
		protoFileNodes, parseErr := protoParser.ParseToAST(protoFilePath)
		if parseErr != nil {
			logger.Error().Err(parseErr).Msgf("Unable to parse proto file: %q", protoFilePath)
		}

		var moduleName string
		pathParts := strings.Split(protoFilePath, string(filepath.Separator))
		for idx, pathPart := range pathParts {
			// Module name is ALWAYS the child directory of "proto/poktroll".
			if pathPart != "poktroll" {
				continue
			}
			moduleName = pathParts[idx+1]
			break
		}

		// Iterate through the file node and walk its AST.
		for _, fileNode := range protoFileNodes {
			ast.Walk(fileNode, func(node ast.Node) (bool, ast.VisitFunc) {
				// Search for Message nodes:
				if msgNode, ok := node.(*ast.MessageNode); ok {
					// Filter out messages with names other than "Params".
					if msgNode.Name.Val == "Params" {
						return true, func(node ast.Node) (bool, ast.VisitFunc) {
							if fieldNode, ok := node.(*ast.FieldNode); ok {
								moduleFieldNodes := append(paramsFieldNodesByModule[moduleName], fieldNode)
								paramsFieldNodesByModule[moduleName] = moduleFieldNodes
							}
							return false, nil
						}
					}
					return false, nil
				}
				return true, nil
			})
		}
	}
}

func main() {
	// This is necessary because multiline strings in golang do not support embedded backticks.
	template := fmt.Sprintf(paramsDocsTemplateStr, "```", "```")

	docs, err := prepareGovernanceParamsDocs(flagProtoRootPathValue, template)
	if err != nil {
		fmt.Println("Error preparing governance params docs:", err)
		return
	}

	err = writeContentToFile(destinationFile, docs)
	if err != nil {
		fmt.Println("Error writing content to file:", err)
		return
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
