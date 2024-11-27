package main

import (
	"context"
	"path/filepath"

	"github.com/jhump/protoreflect/desc/protoparse/ast"
	"github.com/spf13/cobra"

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
	if err := checkProtoResponseTypes(ctx); err != nil {
		return err
	}

	// Check message handler return responses for at least one field assignment.
	// TODO_IN_THIS_COMMIT: ...

	return nil
}

// TODO_IN_THIS_COMMIT: move & godoc...
func checkProtoResponseTypes(ctx context.Context) error {
	//logger := polylog.Ctx(ctx).With("func", "checkProtoResponseTypes")

	responseMsgNodes := make(map[string]*ast.MessageNode)

	if err := filepath.Walk(
		flagRootValue,
		protoast.ForEachMatchingFileWalkFn(
			flagFileIncludePatternValue,
			protoast.NewFindResponseProtosInFileFn(ctx, responseMsgNodes),
		),
	); err != nil {
		return err
	}

	//logger.Debug().Msgf("responseMsgNodes: %+v", responseMsgNodes)
	for msgName, msgNode := range responseMsgNodes {
		var hasField bool
		ast.Walk(msgNode, func(n ast.Node) (bool, ast.VisitFunc) {
			if hasField {
				return false, nil
			}

			if _, ok := n.(*ast.FieldNode); ok {
				hasField = true
				return false, nil
			}

			return true, nil
		})

		if hasField {
			delete(responseMsgNodes, msgName)
		}
	}

	printCheckResponsesResults(ctx, responseMsgNodes)

	return nil
}

func printCheckResponsesResults(ctx context.Context, responseMsgNodes map[string]*ast.MessageNode) {
	logger := polylog.Ctx(ctx)

	if len(responseMsgNodes) == 0 {
		logger.Info().Msg("🎉 No offending response messages found! 🎉")
		return
	}

	logger.Info().Msgf("🚨 Found %d offending response messages in %q 🚨",
		len(responseMsgNodes), flagRootValue,
	)
	for msgName, msgNode := range responseMsgNodes {
		logger.Info().Str("msg_name", msgName).Msg(msgNode.Start().String())
	}
}
