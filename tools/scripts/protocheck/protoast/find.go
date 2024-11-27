package protoast

import (
	"context"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse/ast"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const StableMarshalerAllOptName = "(gogoproto.stable_marshaler_all)"

const GogoImportName = "gogoproto/gogo.proto"

// TODO_IN_THIS_COMMIT: godoc...
func NewFindResponseProtosInFileFn(
	ctx context.Context,
	responseMsgNodes map[string]*ast.MessageNode,
) func(path string) {
	logger := polylog.Ctx(ctx).With("func", "NewFindResponseProtosInFileFn")

	return func(protoFilePath string) {
		// Parse the .proto file into file nodes.
		// NB: MUST use #ParseToAST instead of #ParseFiles to get source positions.
		protoAST, parseErr := DefaultParser.ParseToAST(protoFilePath)
		if parseErr != nil {
			logger.Error().Err(parseErr).Msgf("Unable to parse proto file: %q", protoFilePath)
		}

		// Iterate through the file nodes and build a protoFileStat for each file.
		// NB: There should only be one file node per file.
		for _, fileNode := range protoAST {
			ast.Walk(fileNode, func(n ast.Node) (bool, ast.VisitFunc) {
				if msg, ok := n.(*ast.MessageNode); ok {
					if strings.HasSuffix(msg.Name.Val, "Response") {
						logger.Debug().Msgf("found response message: %s", msg.Name.Val)

						responseMsgNodes[msg.Name.Val] = msg
					}

					return false, nil
				}

				return true, nil
			})
		}
	}
}

// NewFindUnstableProtosInFileFn returns a function which is expected to be called for
// each proto file found. The function walks that file's AST and excludes it from
// the list of unstable files if it contains the stable_marshaler_all option.
func NewFindUnstableProtosInFileFn(
	ctx context.Context,
	unstableProtoFilesByPath map[string]*ProtoFileStat,
) func(path string) {
	logger := polylog.Ctx(ctx)

	return func(protoFilePath string) {
		// Parse the .proto file into file nodes.
		// NB: MUST use #ParseToAST instead of #ParseFiles to get source positions.
		protoAST, parseErr := DefaultParser.ParseToAST(protoFilePath)
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
	unstableProtoFilesByPath map[string]*ProtoFileStat,
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

		if optName != StableMarshalerAllOptName {
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

// TODO_IN_THIS_COMMIT: godoc...
func newProtoFileStat(fileNode *ast.FileNode) *ProtoFileStat {
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

			if node.Name.AsString() == GogoImportName {
				foundGogoImport = true
			}
		case *ast.OptionNode:
			lastOptionNode = node
		}
	}

	protoStat := &ProtoFileStat{
		PkgSource:     pkgNode.Start(),
		HasGogoImport: foundGogoImport,
	}
	if lastOptionNode != nil {
		protoStat.LastOptSource = lastOptionNode.Start()
	}
	if lastImportNode != nil {
		protoStat.LastImportSource = lastImportNode.Start()
	}

	return protoStat
}
