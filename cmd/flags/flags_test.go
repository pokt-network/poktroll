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
	testCmd.Flags().String(cosmosflags.FlagGRPCInsecure, "", "grpc insecure flag")

	boolValues := []string{
		BooleanTrueValue,
		BooleanFalseValue,
	}

	for _, boolValue := range boolValues {
		// Set the flag to the current boolean value.
		err := testCmd.Flag(cosmosflags.FlagGRPCInsecure).Value.Set(boolValue)
		require.NoError(t, err)

		// Assert that GetFlagValueString returns the expected value.
		networkFlagValue, err := GetFlagValueString(testCmd, cosmosflags.FlagGRPCInsecure)
		require.NoError(t, err)
		require.Equal(t, boolValue, networkFlagValue)

		// Test GetFlagBool function as well.
		boolFlagValue, err := GetFlagBool(testCmd, cosmosflags.FlagGRPCInsecure)
		require.NoError(t, err)
		require.Equal(t, boolValue == BooleanTrueValue, boolFlagValue)
	}

	// Test invalid boolean value
	err := testCmd.Flag(cosmosflags.FlagGRPCInsecure).Value.Set("invalid")
	require.NoError(t, err)

	_, err = GetFlagBool(testCmd, cosmosflags.FlagGRPCInsecure)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected 'true' or 'false'")
}

func TestGetFlag_NotRegistered(t *testing.T) {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test",
		Long:  `Test`,
	}

	// Test getting a flag that doesn't exist
	_, err := GetFlag(testCmd, "nonexistent-flag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag not registered")
	require.Contains(t, err.Error(), "nonexistent-flag")

	// Test GetFlagValueString with non-existent flag
	_, err = GetFlagValueString(testCmd, "nonexistent-flag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag not registered")

	// Test GetFlagBool with non-existent flag
	_, err = GetFlagBool(testCmd, "nonexistent-flag")
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag not registered")
}
