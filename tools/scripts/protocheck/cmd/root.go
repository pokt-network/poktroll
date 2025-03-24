package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/pokt-network/pocket/pkg/polylog/polyzero"
	_ "github.com/pokt-network/pocket/pkg/polylog/polyzero"
)

var (
	flagRootName      = "root"
	flagRootShorthand = "r"
	flagRootValue     = "./proto"
	flagRootUsage     = "Set the path of the directory from which to start walking the filesystem tree in search of files matching --file-pattern."

	flagFileIncludePatternName      = "file-pattern"
	flagFileIncludePatternShorthand = "p"
	flagFileIncludePatternValue     = "*.proto"
	flagFileIncludePatternUsage     = "Set the pattern passed to filepath.Match(), used to include file names which match."

	rootCmd = &cobra.Command{
		Use:   "protocheck [subcommand] [flags]",
		Short: "A tool for heuristically identifying and fixing issues in protobuf files and usage.",
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagRootValue, flagRootName, flagRootShorthand, flagRootValue, flagRootUsage)
	rootCmd.PersistentFlags().StringVarP(&flagFileIncludePatternValue, flagFileIncludePatternName, flagFileIncludePatternShorthand, flagFileIncludePatternValue, flagFileIncludePatternUsage)
}

func main() {
	flag.Parse()

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
