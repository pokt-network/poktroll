package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/tools/scripts/protocheck/protoast"
)

var (
	checkUnstableCmd = &cobra.Command{
		Use:    "unstable [flags]",
		Short:  "Recursively list or fix all protobuf files which omit the 'stable_marshaler_all' option.",
		PreRun: setupPrettyLogger,
		RunE:   runUnstable,
	}
)

func init() {
	checkUnstableCmd.Flags().StringVarP(&flagRootValue, flagRootName, flagRootShorthand, flagRootValue, flagRootUsage)
	checkUnstableCmd.Flags().StringVarP(&flagFileIncludePatternValue, flagFileIncludePatternName, flagFileIncludePatternShorthand, flagFileIncludePatternValue, flagFileIncludePatternUsage)
	checkUnstableCmd.Flags().BoolVarP(&flagFixValue, flagFixName, flagFixShorthand, flagFixValue, flagFixUsage)
	rootCmd.AddCommand(checkUnstableCmd)
}

func runUnstable(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	logger := polylog.Ctx(ctx)

	unstableProtoFilesByPath := make(map[string]*protoast.ProtoFileStat)

	logger.Info().Msgf("Recursively checking for files matching %q in %q", flagFileIncludePatternValue, flagRootValue)

	// 1. Walk the directory tree.
	// 2. For each matching file:
	//   2a. Add it to the unstableProtoFilesByPath.
	//   2b. Walk the AST of the matching file.
	//   2c. Exclude files which contain the stable_marshaler_all option.
	if pathWalkErr := filepath.Walk(
		flagRootValue,
		protoast.ForEachMatchingFileWalkFn(
			flagFileIncludePatternValue,
			protoast.FindUnstableProtosInFileFn(ctx, unstableProtoFilesByPath),
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

	os.Exit(CodeUnstableProtosFound)
	return nil
}

func runFixUnstable(ctx context.Context, unstableProtoFilesByPath map[string]*protoast.ProtoFileStat) {
	logger := polylog.Ctx(ctx)
	logger.Info().Msg("Fixing unstable marshaler proto files...")

	var fixedProtoFilePaths []string
	for unstableProtoFile, protoStat := range unstableProtoFilesByPath {
		if protoStat != nil {
			if insertErr := protoast.InsertStableMarshalerAllOption(unstableProtoFile, protoStat); insertErr != nil {
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
