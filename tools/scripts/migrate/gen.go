package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testmigration"
)

var (
	flagNumAccounts         = "num-accounts"
	flagShortNumAccounts    = "n"
	flagDistributionFn      = "dist-fn"
	flagShortDistributionFn = "d"

	logger = polyzero.NewLogger(polyzero.WithLevel(polyzero.InfoLevel))

	numAccounts        int
	distributionFnName string

	cmdGen = &cobra.Command{
		Use: "gen",
		// TODO_IN_THIS_COMMIT: ...
		//Short:                      "",
		//Long:                       "",
		//Example:                    "",
		Args: cobra.ExactArgs(0),
		RunE: runGen,
	}
)

const (
	roundRobinDistributionName     = "round-robin"
	allUnstakedDistributionName    = "unstaked"
	allApplicationDistributionName = "application"
	allSupplierDistributionName    = "supplier"
)

func init() {
	// TODO_IN_THIS_COMMIT: extract the following flags:
	// - distribution
	// - number of accounts
	// - output paths
	cmdGen.Flags().IntVarP(&numAccounts, flagNumAccounts, flagShortNumAccounts, 10, "The number of accounts to generate")
	cmdGen.Flags().StringVarP(&distributionFnName, flagDistributionFn, flagShortDistributionFn, roundRobinDistributionName, "The distribution function to use")
}

func main() {
	if err := cmdGen.Execute(); err != nil {
		logger.Error().Err(err).Msg("exiting due to error")
		os.Exit(1)
	}
}

func runGen(_ *cobra.Command, _ []string) error {
	distributionFn, err := getDistributionFn(distributionFnName)
	if err != nil {
		return err
	}

	morseStateExportBz, morseAccountStateBz, err := testmigration.NewMorseStateExportAndAccountStateBytes(numAccounts, distributionFn)
	if err != nil {
		return err
	}

	if err = os.WriteFile("morse_state_export.json", morseStateExportBz, 0644); err != nil {
		return err
	}

	return os.WriteFile("morse_account_state.json", morseAccountStateBz, 0644)
}

// TODO_IN_THIS_COMMIT: godoc and move...
func getDistributionFn(distributionFnName string) (distributionFn testmigration.MorseAccountActorTypeDistributionFn, err error) {
	switch distributionFnName {
	case roundRobinDistributionName:
		distributionFn = testmigration.RoundRobinAllMorseAccountActorTypes
	case allUnstakedDistributionName:
		distributionFn = testmigration.AllUnstakedMorseAccountActorType
	case allApplicationDistributionName:
		distributionFn = testmigration.AllApplicationMorseAccountActorType
	case allSupplierDistributionName:
		distributionFn = testmigration.AllSupplierMorseAccountActorType
	default:
		return nil, fmt.Errorf("unknown distribution function %q", distributionFnName)
	}

	return distributionFn, nil
}
