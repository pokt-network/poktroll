package main

import (
	"context"
	"fmt"
	goast "go/ast"
	"path/filepath"
	"strings"

	protoparseast "github.com/jhump/protoreflect/desc/protoparse/ast"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/tools/scripts/protocheck/protoast"
)

var (
	checkResponsesCmd = &cobra.Command{
		Use:    "responses [flags]",
		Short:  "Checks that all message responses contain results.",
		PreRun: setupPrettyLogger,
		RunE:   runCheckResponses,
	}
)

func init() {
	//checkResponsesCmd.Flags().StringVarP(&flagModuleValue, flagModule, flagModuleShorthand, flagModuleValue, flagModuleUsage)
	checkResponsesCmd.Flags().StringVarP(&flagRootValue, flagRootName, flagRootShorthand, flagRootValue, flagRootUsage)
	checkResponsesCmd.Flags().StringVarP(&flagFileIncludePatternValue, flagFileIncludePatternName, flagFileIncludePatternShorthand, flagFileIncludePatternValue, flagFileIncludePatternUsage)
	checkResponsesCmd.Flags().StringVarP(&flagLogLevelValue, flagLogLevel, flagLogLevelShorthand, flagLogLevelValue, flagLogLevelUsage)
	rootCmd.AddCommand(checkResponsesCmd)
}

func runCheckResponses(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Check protobuf message types ending in "Response" for at least one field.
	responseMsgStats, err := findProtoResponseTypes(ctx)
	if err != nil {
		// TODO_IN_THIS_COMMIT: exit with non-zero code if error.
		return err
	}

	validResponseMsgStats, invalidResponseMsgStats := groupProtoResponseTypes(responseMsgStats)

	// Check message handler return responses for at least one field assignment.
	if err = checkValidResponseGoUsages(ctx, validResponseMsgStats); err != nil {
		// TODO_IN_THIS_COMMIT: exit with non-zero code if error.
		return err
	}

	printInvalidResponseMsgs(ctx, invalidResponseMsgStats)
	// TODO_IN_THIS_COMMIT: print valid response invalid usages.

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func findProtoResponseTypes(ctx context.Context) (responseMsgStats map[string]*protoast.ProtoMsgStat, err error) {
	responseMsgNodes := make(map[string]*protoast.ProtoMsgStat)

	if err = filepath.Walk(
		flagRootValue,
		protoast.ForEachMatchingFileWalkFn(
			flagFileIncludePatternValue,
			protoast.NewFindResponseProtosInFileFn(ctx, responseMsgNodes),
		),
	); err != nil {
		return nil, err
	}

	return responseMsgNodes, nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func groupProtoResponseTypes(responseMsgStats map[string]*protoast.ProtoMsgStat) (
	validResponseMsgStats,
	invalidResponseMsgStats map[string]*protoast.ProtoMsgStat,
) {
	validResponseMsgStats = make(map[string]*protoast.ProtoMsgStat)
	invalidResponseMsgStats = make(map[string]*protoast.ProtoMsgStat)

	for msgName, msgStat := range responseMsgStats {
		var hasField bool
		protoparseast.Walk(msgStat.Node, func(n protoparseast.Node) (bool, protoparseast.VisitFunc) {
			if hasField {
				return false, nil
			}

			if _, ok := n.(*protoparseast.FieldNode); ok {
				hasField = true
				return false, nil
			}

			return true, nil
		})

		if hasField {
			validResponseMsgStats[msgName] = msgStat
		} else {
			invalidResponseMsgStats[msgName] = msgStat
		}
	}

	return validResponseMsgStats, invalidResponseMsgStats
}

// TODO_IN_THIS_COMMIT: move & godoc...
func printInvalidResponseMsgs(ctx context.Context, responseMsgNodes map[string]*protoast.ProtoMsgStat) {
	logger := polylog.Ctx(ctx)

	if len(responseMsgNodes) == 0 {
		logger.Info().Msg("🎉 No offending proto messages found! 🎉")
		return
	}

	logger.Info().Msgf("🚨 Found %d offending proto messages in %q 🚨",
		len(responseMsgNodes), flagRootValue,
	)
	for msgName, msgStat := range responseMsgNodes {
		logger.Info().Str("msg_name", msgName).Msg(msgStat.Node.Start().String())
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func checkValidResponseGoUsages(ctx context.Context, responseMsgNodes map[string]*protoast.ProtoMsgStat) error {
	logger := polylog.Ctx(ctx) //.With("func", "checkValidResponseGoUsages")

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Tests: false,
	}

	logger.Debug().Msgf("checking %d used resopnse types", len(responseMsgNodes))

	modulePkgs, err := packages.Load(cfg, poktrollMoudlePkgsPattern)
	if err != nil {
		return err
	}

	for msgName, msgStat := range responseMsgNodes {
		// DEV_NOTE: Assumes that the go pkg path follows the patterh:
		// `github.com/pokt-network/poktroll/x/<module>/types`
		goModulePkgPath := filepath.Dir(msgStat.GoPkgPath)

		//logger = logger.With("go_module_pkg_path", goModulePkgPath)
		//logger.Debug().Send()

		// Look through corresponding keeper package for a selector expression where
		// the type pkg path of X is equal to the msgStat.GoPkgPath AND the Sel name
		// matches the msgStat.MsgName.
		for _, pkg := range modulePkgs {
			if !strings.HasPrefix(pkg.PkgPath, goModulePkgPath) {
				continue
			}

			// TODO_IN_THIS_COMMIT: refactor & reuse NewInspectLastReturnArgFn.
			// TODO_IN_THIS_COMMIT: refactor & reuse NewInspectLastReturnArgFn.
			// TODO_IN_THIS_COMMIT: refactor & reuse NewInspectLastReturnArgFn.
			// TODO_IN_THIS_COMMIT: refactor & reuse NewInspectLastReturnArgFn.
			// TODO_IN_THIS_COMMIT: refactor & reuse NewInspectLastReturnArgFn.

			// TODO_IN_THIS_COMMIT: extract... BEGIN
			// TODO_IN_THIS_COMMIT: use inspectFnNodeBody to log position of usages within fnNodeBody.
			inspectFnNodeBody := func(n goast.Node) bool {
				if n == nil {
					return false
				}

				if sel, ok := n.(*goast.SelectorExpr); ok {
					if xIdent, ok := sel.X.(*goast.Ident); ok {
						if sel.Sel.Name == msgName { //&&
							//xIdent.Name == msgStat.GoPkgPath {
							//logger.Debug().Msgf("found usage of %s.%s", xIdent.Name, sel.Sel.Name)
							xIdentPosition := pkg.Fset.Position(xIdent.Pos()).String()
							logger.Debug().
								Str("position", fmt.Sprintf(" %s ", xIdentPosition)).
								Msgf("found usage of %s", msgName)
							return false
						}
						//} else {
						//	logger.Warn().
						//		Str("node_type", fmt.Sprintf("%T", n)).
						//		Str("position", fmt.Sprintf(" %s ", pkg.Fset.Position(n.Pos()).String())).
						//		Msgf("skipping non ident selector X")
					}
					//} else {
					//	logger.Warn().
					//		Str("node_type", fmt.Sprintf("%T", n)).
					//		Str("position", fmt.Sprintf(" %s ", pkg.Fset.Position(n.Pos()).String())).
					//		Msgf("skipping non selector expression")
				}

				return true
			}
			inspectFirstReturnArgFn := func(fnNode *goast.FuncDecl) bool {
				fnNodePosition := pkg.Fset.Position(fnNode.Pos())

				if pathMatchesProtobufGenGo(fnNodePosition.Filename) {
					return false
				}

				fnPos := pkg.Fset.Position(fnNode.Pos())
				fnFilename := filepath.Base(fnPos.Filename)
				if !isNodeReceiverMethod("msgServer", fnNode) &&
					!strings.HasPrefix(fnFilename, "query_") {
					return false
				}

				//logger.Debug().Msgf("inspecting %s", fnNodePosition)

				// Ensure the first return argument matches the msgName.
				firstResultType := fnNode.Type.Results.List[0].Type
				// TODO_IN_THIS_COMMIT: investigate consolidating this with TraceExpressionStack.
				if firstResultTypePtr, ok := firstResultType.(*goast.StarExpr); ok {
					if firstResultTypeSel, ok := firstResultTypePtr.X.(*goast.SelectorExpr); ok {
						if firstResultTypeSel.Sel.Name == msgName {
							//logger.Debug().
							//	Str("position", fmt.Sprintf(" %s ", fnNodePosition)).
							//	Msgf("found usage of %s", msgName)

							goast.Inspect(fnNode.Body, inspectFnNodeBody)

							return false
						}
					}
				} else {
					// TODO_IN_THIS_COMMIT: handle cases where we need to trace the first result type.
					//logger.Warn().
					//	Str("node_type", fmt.Sprintf("%T", firstResultType)).
					//	Msg("skipping")

					//goast2.TraceExpressionStack(ctx, firstResultType, modulePkgs, pkg, fnNode,
					//	func(ctx context.Context, str string) {
					//
					//	},
					//)
				}

				return false
			}
			// --- END

			inspectReturnStmtFn := newInspectReturnStmtFn(ctx, pkg, inspectFirstReturnArgFn)

			for _, fileNode := range pkg.Syntax {
				goast.Inspect(fileNode, inspectReturnStmtFn)
			}
		}
	}

	return err
}
