package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func setupPrettyLogger(cmd *cobra.Command, _ []string) {
	logger := polyzero.NewLogger(
		polyzero.WithWriter(zerolog.ConsoleWriter{
			Out: os.Stdout,
			PartsExclude: []string{
				zerolog.TimestampFieldName,
			},
		}),
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevelValue)),
	)

	cmd.SetContext(logger.WithContext(cmd.Context()))
}
