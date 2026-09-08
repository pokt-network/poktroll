package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	//nolint:staticcheck // SA1019 TODO_TECHDEBT: switch from protoparse to github.com/bufbuild/protocompile.
	// More info: https://github.com/jhump/protoreflect/issues/637#issuecomment-2867273251
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoparse/ast"

	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// paramField holds information about a param message field for interpolation into the param_field_row_template.md template.
// Fields:
// - Module: Name of the module
// - Name: Name of the field
// - Type: Type of the field
// - Comment: Associated comment for the field
type paramField struct {
	Module  string
	Name    string
	Type    string
	Comment string
}

const (
	paramsDocTemplatePath     = "./tools/scripts/docusaurus/params_doc_template.md"
	paramFieldRowTemplatePath = "./tools/scripts/docusaurus/param_field_row_template.md"
	destinationPath           = "docusaurus/docs/3_protocol/governance/2_gov_params.md"
)

var (
	protoParser = protoparse.Parser{
		IncludeSourceCodeInfo: true,
	}
	logger = polylog.DefaultContextLogger

	flagProtoRootPathValue string
)

func init() {
	flag.StringVar(&flagProtoRootPathValue, "proto-root", "./proto", "path to the proto files root directory; this directory will be walked, looking for Params.proto files")
	flag.Parse()
}

func main() {
	// Parse templates.
	templates, err := template.ParseFiles(paramsDocTemplatePath, paramFieldRowTemplatePath)
	if err != nil {
		logger.Error().Err(err).Msgf("Unable to parse template path %q", paramsDocTemplatePath)
		os.Exit(1)
	}

	// Interpolate templates.
	docs, err := prepareGovernanceParamsDocs(flagProtoRootPathValue, templates)
	if err != nil {
		logger.Error().Err(err).Msg("Error preparing governance params docs")
		os.Exit(1)
	}

	// Write output to destination.
	err = writeContentToFile(destinationPath, docs)
	if err != nil {
		logger.Error().Err(err).Msg("Error writing content to file")
		os.Exit(1)
	}

}

// prepareGovernanceParamsDocs does the following:
// - Recursively walks the filesystem starting from protoFilesRootDir
// - Looks for files named "params.proto"
// - For each matching proto file, discovers fields on any "Params" message types
// - Interpolates all discovered param fields into the provided templates
// - Returns the final output as a string
func prepareGovernanceParamsDocs(protoFilesRootDir string, templates *template.Template) (string, error) {
	paramsDocOutputBuf := new(bytes.Buffer)
	paramFieldRowsOutputBuf := new(bytes.Buffer)
	paramsFieldNodesByModule := make(map[string][]*ast.FieldNode)

	if pathWalkErr := filepath.Walk(
		protoFilesRootDir,
		forEachMatchingFileWalkFn(
			"params.proto",
			newCollectParamsFieldNodesInFileFn(paramsFieldNodesByModule),
		),
	); pathWalkErr != nil {
		logger.Error().Err(pathWalkErr)
		os.Exit(1)
	}

	var paramFields = make([]paramField, 0)
	for moduleName, fieldNodes := range paramsFieldNodesByModule {
		for _, fieldNode := range fieldNodes {
			// Uncomment and concatenate the field's comment lines.
			var comment strings.Builder
			for commentIdx, commentLine := range fieldNode.LeadingComments() {
				var commentFmt = " %s"
				if commentIdx == 0 {
					commentFmt = "%s"
				}

				comment.WriteString(fmt.Sprintf(commentFmt, strings.Trim(commentLine.Text, " /")))
			}

			// Extract the field's type information.
			paramFields = append(paramFields, paramField{
				Module:  moduleName,
				Type:    string(fieldNode.FldType.AsIdentifier()),
				Name:    fieldNode.Name.Val,
				Comment: comment.String(),
			})
		}
	}

	// Sort param field rows by module name and field name.
	sort.Slice(paramFields, func(i, j int) bool {
		if paramFields[i].Module == paramFields[j].Module {
			return paramFields[i].Name < paramFields[j].Name
		}
		return paramFields[i].Module < paramFields[j].Module
	})

	for _, param := range paramFields {
		_, paramFieldRowTemplateFileName := filepath.Split(paramFieldRowTemplatePath)
		if err := templates.ExecuteTemplate(paramFieldRowsOutputBuf, paramFieldRowTemplateFileName, param); err != nil {
			return "", err
		}
	}

	_, paramsDocTemplateFileName := filepath.Split(paramsDocTemplatePath)
	if err := templates.ExecuteTemplate(
		paramsDocOutputBuf,
		paramsDocTemplateFileName,
		paramFieldRowsOutputBuf.String(),
	); err != nil {
		return "", err
	}

	return paramsDocOutputBuf.String(), nil
}

// forEachMatchingFileWalkFn returns a filepath.WalkFunc that:
// - Iterates over files matching fileNamePattern against each file name
// - For matching files, calls fileMatchedFn with the respective path
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

// newCollectParamsFieldNodesInFileFn returns a function that:
// - Receives a proto file path
// - Walks its AST to discover all fields (*ast.FieldNode) present on any message named "Params"
// - Appends discovered fields to the []*ast.FieldNode in paramsFieldNodesByModule for the corresponding module name key

func newCollectParamsFieldNodesInFileFn(paramsFieldNodesByModule map[string][]*ast.FieldNode) func(protoFilePath string) {
	return func(protoFilePath string) {
		protoFileNodes, parseErr := protoParser.ParseToAST(protoFilePath)
		if parseErr != nil {
			logger.Error().Err(parseErr).Msgf("Unable to parse proto file: %q", protoFilePath)
		}

		var moduleName string
		pathParts := strings.Split(protoFilePath, string(filepath.Separator))
		for idx, pathPart := range pathParts {
			// Module name is ALWAYS the child directory of "proto/pocket".
			if pathPart != "pocket" {
				continue
			}
			moduleName = pathParts[idx+1]
			break
		}

		collectParamFieldNodesVisitFn := newCollectParamFieldNodesVisitFn(moduleName, paramsFieldNodesByModule)
		filterMsgNodesVisitFn := newFilterMsgNodesVisitFn("Params", collectParamFieldNodesVisitFn)

		// Iterate through the file node and walk its AST.
		for _, fileNode := range protoFileNodes {
			ast.Walk(fileNode, filterMsgNodesVisitFn)
		}
	}
}

// newFilterMsgNodesVisitFn returns an ast.VisitFunc that:
// - Filters out MessageNodes whose name does not match the given name
// - When passed to ast.Walk(), calls msgNodeVisitFn when a matching message node is discovered

func newFilterMsgNodesVisitFn(name string, msgNodeVisitFn ast.VisitFunc) ast.VisitFunc {
	return func(node ast.Node) (bool, ast.VisitFunc) {

		// Continue walking the AST if the node is not a MessageNode.
		msgNode, isMsgNode := node.(*ast.MessageNode)
		if !isMsgNode {
			return true, nil
		}

		// Filter out messages with names other than those with matching name.
		if msgNode.Name.Val != name {
			return false, nil
		}

		// MsgNode found, walk its AST using msgNodeVisitFn.
		return true, msgNodeVisitFn
	}
}

// newCollectParamFieldNodesVisitFn returns an ast.VisitFunc that:
// - Collects all FieldNodes discovered
// - Appends them to the []*ast.FieldNode slice in the paramsFieldNodesByModule map under the given moduleName key

func newCollectParamFieldNodesVisitFn(
	moduleName string, paramsFieldNodesByModule map[string][]*ast.FieldNode,
) ast.VisitFunc {
	return func(node ast.Node) (bool, ast.VisitFunc) {
		fieldNode, isFieldNode := node.(*ast.FieldNode)
		if !isFieldNode {
			return true, nil
		}

		moduleFieldNodes := append(paramsFieldNodesByModule[moduleName], fieldNode)
		paramsFieldNodesByModule[moduleName] = moduleFieldNodes
		return false, nil
	}
}

// writeContentToFile writes the given content to the specified file path.
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
