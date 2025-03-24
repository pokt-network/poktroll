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

	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoparse/ast"

	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// paramField is used to hold param message field information for interpolation
// into the param_field_row_template.md template.
type paramField struct {
	Module  string
	Name    string
	Type    string
	Comment string
}

const (
	paramsDocTemplatePath     = "./tools/scripts/docusaurus/params_doc_template.md"
	paramFieldRowTemplatePath = "./tools/scripts/docusaurus/param_field_row_template.md"
	destinationPath           = "docusaurus/docs/protocol/governance/params.md"
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
	templs, err := template.ParseFiles(paramsDocTemplatePath, paramFieldRowTemplatePath)
	if err != nil {
		logger.Error().Err(err).Msgf("Unable to parse template path %q", paramsDocTemplatePath)
		os.Exit(1)
	}

	// Interpolate templates.
	docs, err := prepareGovernanceParamsDocs(flagProtoRootPathValue, templs)
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

// prapareGovernanceParamsDocs recursively walks the filesystem starting from
// protoFilesRootDir, looking for files names matching "params.proto". For each
// matching proto file, the fields on any "Params" message types are discovered.
// All discovered param fields are interpolated into the templates provided by
// templs and the final output is returned.
func prepareGovernanceParamsDocs(protoFilesRootDir string, templs *template.Template) (string, error) {
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
			comment := ""
			for commentIdx, commentLine := range fieldNode.LeadingComments() {
				var commentFmt = " %s"
				if commentIdx == 0 {
					commentFmt = "%s"
				}

				comment += fmt.Sprintf(commentFmt, strings.Trim(commentLine.Text, " /"))
			}

			// Extract the field's type information.
			paramFields = append(paramFields, paramField{
				Module:  moduleName,
				Name:    fieldNode.Name.Val,
				Type:    string(fieldNode.FldType.AsIdentifier()),
				Comment: comment,
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
		if err := templs.ExecuteTemplate(paramFieldRowsOutputBuf, paramFieldRowTemplateFileName, param); err != nil {
			return "", err
		}
	}

	_, paramsDocTemplateFileName := filepath.Split(paramsDocTemplatePath)
	if err := templs.ExecuteTemplate(
		paramsDocOutputBuf,
		paramsDocTemplateFileName,
		paramFieldRowsOutputBuf.String(),
	); err != nil {
		return "", err
	}

	return paramsDocOutputBuf.String(), nil
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

// newCollectParamsFieldNodesInFileFn returns a function which receives a proto file
// path and walks its AST to discover all fields (*ast.fieldNode) which are present
// on any message named "Params", if present. Discovered fields are appended to the
// []*ast.fieldNode in paramsFieldNodesByModule for the corresponding module name key.
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

// newFilterMsgNodesVisitFn returns an ast.VisitFunc which filters out MessageNodes
// whose name does not match the given name. When the returned visit function is passed
// to ast.Walk(), msgNodeVisitFn will be called when a matching message node id discovered.
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

// newCollectParamFieldNodesVisitFn returns an ast.VisitFunc which collects all
// FieldNodes discovered and appends them to the []*ast.FieldNode slice in the
// paramsFieldNodesByModule map under the given moduleName key.
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
