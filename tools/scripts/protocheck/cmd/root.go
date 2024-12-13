package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	rootCmd = &cobra.Command{
		Use:   "protocheck [subcommand] [flags]",
		Short: "A tool for heuristically identifying and fixing issues in protobuf files and usage.",
	}
)

func main() {
	zlConsoleWriter := zerolog.ConsoleWriter{
		Out: os.Stderr,
		// Remove the timestamp from the output
		FormatTimestamp: func(i interface{}) string {
			return ""
		},
	}
	logger := polyzero.NewLogger(
		polyzero.WithOutput(zlConsoleWriter),
	)

	loggerCtx := logger.WithContext(context.Background())
	if err := rootCmd.ExecuteContext(loggerCtx); err != nil {
		logger.Error().Err(err)
		os.Exit(CodeRootCmdErr)
	}
}
