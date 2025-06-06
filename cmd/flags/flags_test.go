package flags

import (
	"testing"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetFlagValueString(t *testing.T) {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test",
		Long:  `Test`,
	}

	// Register relevant flags for testing, defaulting to empty values.
	testCmd.Flags().String(FlagNetwork, "", "network flag")

	networkNames := []string{
		LocalNetworkName,
		AlphaNetworkName,
		BetaNetworkName,
		MainNetworkName,
	}

	for _, networkName := range networkNames {
		// Set the network flag to the correct value.
		err := testCmd.Flag(FlagNetwork).Value.Set(networkName)
		require.NoError(t, err)

		// Assert that GetFlagValueString returns the expected value.
		networkFlagValue, err := GetFlagValueString(testCmd, FlagNetwork)
		require.NoError(t, err)
		require.Equal(t, networkName, networkFlagValue)
	}
}

func TestGetFlagValueBool(t *testing.T) {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test",
		Long:  `Test`,
	}

	// Register relevant flags for testing, defaulting to empty values.
	testCmd.Flags().String(cosmosflags.FlagGRPCInsecure, "", "network flag")

	boolValues := []string{
		BooleanTrueValue,
		BooleanFalseValue,
	}

	for _, boolValue := range boolValues {
		// Set the network flag to the correct value.
		err := testCmd.Flag(cosmosflags.FlagGRPCInsecure).Value.Set(boolValue)
		require.NoError(t, err)

		// Assert that GetFlagValueString returns the expected value.
		networkFlagValue, err := GetFlagValueString(testCmd, cosmosflags.FlagGRPCInsecure)
		require.NoError(t, err)
		require.Equal(t, boolValue, networkFlagValue)
	}
}
