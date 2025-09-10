package cmd

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

var (
	// flagsToLog is a list of flags that will be logged by the logger in the
	// test command for the purpose of asserting against the log output.
	flagsToLog = []string{
		flags.FlagNetwork,
		cosmosflags.FlagChainID,
		cosmosflags.FlagNode,
		cosmosflags.FlagGRPC,
		cosmosflags.FlagGRPCInsecure,
		flags.FlagFaucetBaseURL,
	}
	flagLogFmt = "flag %q value: %s"

	testLogBuffer = new(bytes.Buffer)
)

// testPreRunE resets the log buffer, sets up the global logger, and sets it on the command context.
func testPreRunE(cmd *cobra.Command, args []string) error {
	// Reset the log buffer on each command execution.
	testLogBuffer.Reset()

	// Set the log level to debug, and write to a buffer for testing.
	logger.Logger = polyzero.NewLogger(
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithSetupFn(logger.NewSetupConsoleWriter(testLogBuffer)),
	)

	// Normally, the root command's PersistentPreRunE would call this.
	return ParseAndSetNetworkRelatedFlags(cmd)
}

// testRunE logs the values of the flags that were set on the command.
func testRunE(cmd *cobra.Command, _ []string) error {
	for _, flagName := range flagsToLog {
		if cmd.Flag(flagName) == nil {
			return fmt.Errorf("flag %q not registered", flagName)
		}

		flagValue := cmd.Flag(flagName).Value.String()
		logger.Logger.Debug().Msgf(flagLogFmt, flagName, flagValue)
	}

	return nil
}

func TestNetworkRelatedFlags(t *testing.T) {
	testCases := []struct {
		networkFlagValue               string
		expectedChainIDFlagValue       string
		expectedNodeFlagValue          string
		expectedGRPCAddrFlagValue      string
		expectedGRPCInsecureFlagValue  bool
		expectedFaucetBaseURLFlagValue string
	}{
		{
			networkFlagValue:               flags.LocalNetworkName,
			expectedChainIDFlagValue:       pocket.LocalNetChainId,
			expectedNodeFlagValue:          pocket.LocalNetRPCURL,
			expectedGRPCAddrFlagValue:      pocket.LocalNetGRPCAddr,
			expectedGRPCInsecureFlagValue:  true,
			expectedFaucetBaseURLFlagValue: pocket.LocalNetFaucetBaseURL,
		},
		{
			networkFlagValue:               flags.AlphaNetworkName,
			expectedChainIDFlagValue:       pocket.AlphaTestNetChainId,
			expectedNodeFlagValue:          pocket.AlphaTestNetRPCURL,
			expectedGRPCAddrFlagValue:      pocket.AlphaNetGRPCAddr,
			expectedGRPCInsecureFlagValue:  false,
			expectedFaucetBaseURLFlagValue: pocket.AlphaTestNetFaucetBaseURL,
		},
		{
			networkFlagValue:               flags.BetaNetworkName,
			expectedChainIDFlagValue:       pocket.BetaTestNetChainId,
			expectedNodeFlagValue:          pocket.BetaTestNetRPCURL,
			expectedGRPCAddrFlagValue:      pocket.BetaNetGRPCAddr,
			expectedGRPCInsecureFlagValue:  false,
			expectedFaucetBaseURLFlagValue: pocket.BetaTestNetFaucetBaseURL,
		},
		{
			networkFlagValue:               flags.MainNetworkName,
			expectedChainIDFlagValue:       pocket.MainNetChainId,
			expectedNodeFlagValue:          pocket.MainNetRPCURL,
			expectedGRPCAddrFlagValue:      pocket.MainNetGRPCAddr,
			expectedGRPCInsecureFlagValue:  false,
			expectedFaucetBaseURLFlagValue: pocket.MainNetFaucetBaseURL,
		},
	}

	for _, test := range testCases {
		desc := fmt.Sprintf("with %q network flag value", test.networkFlagValue)
		t.Run(desc, func(t *testing.T) {
			// Create a fresh command for each test to avoid flag state pollution
			testCmd := &cobra.Command{
				Use:     "test",
				Short:   "Test",
				Long:    `Test`,
				PreRunE: testPreRunE,
				RunE:    testRunE,
			}

			// Register relevant flags for testing, defaulting to empty values.
			testCmd.Flags().String(flags.FlagNetwork, "", "network flag")
			testCmd.Flags().String(cosmosflags.FlagGRPC, "", "grpc addr flag")
			testCmd.Flags().String(cosmosflags.FlagGRPCInsecure, "", "grpc insecure flag")
			testCmd.Flags().String(flags.FlagFaucetBaseURL, "", "faucet base url flag")
			cosmosflags.AddTxFlagsToCmd(testCmd)

			// Set flags and execute as if invoked from the command line.
			testCmd.SetArgs([]string{
				fmt.Sprintf("--%s=%s", flags.FlagNetwork, test.networkFlagValue),
			})

			err := testCmd.ExecuteContext(t.Context())
			require.NoError(t, err)

			t.Logf("test log buffer:\n%s", testLogBuffer.String())

			// Assert that the flags were set as expected.
			chainIDFlag, err := flags.GetFlag(testCmd, cosmosflags.FlagChainID)
			require.NoError(t, err)
			require.Equal(t, test.expectedChainIDFlagValue, chainIDFlag.Value.String())
			expectedLogString := fmt.Sprintf(flagLogFmt, cosmosflags.FlagChainID, test.expectedChainIDFlagValue)
			require.Contains(t, testLogBuffer.String(), expectedLogString)

			nodeFlag, err := flags.GetFlag(testCmd, cosmosflags.FlagNode)
			require.NoError(t, err)
			require.Equal(t, test.expectedNodeFlagValue, nodeFlag.Value.String())
			expectedLogString = fmt.Sprintf(flagLogFmt, cosmosflags.FlagNode, test.expectedNodeFlagValue)
			require.Contains(t, testLogBuffer.String(), expectedLogString)

			grpcAddrFlag, err := flags.GetFlag(testCmd, cosmosflags.FlagGRPC)
			require.NoError(t, err)
			require.Equal(t, test.expectedGRPCAddrFlagValue, grpcAddrFlag.Value.String())
			expectedLogString = fmt.Sprintf(flagLogFmt, cosmosflags.FlagGRPC, test.expectedGRPCAddrFlagValue)
			require.Contains(t, testLogBuffer.String(), expectedLogString)

			grpcInsecureFlagBool, err := flags.GetFlagBool(testCmd, cosmosflags.FlagGRPCInsecure)
			require.NoError(t, err)
			require.Equal(t, test.expectedGRPCInsecureFlagValue, grpcInsecureFlagBool)
			expectedLogString = fmt.Sprintf(flagLogFmt, cosmosflags.FlagGRPCInsecure, fmt.Sprintf("%v", test.expectedGRPCInsecureFlagValue))
			require.Contains(t, testLogBuffer.String(), expectedLogString)

			faucetBaseURLFlag, err := flags.GetFlag(testCmd, flags.FlagFaucetBaseURL)
			require.NoError(t, err)
			require.Equal(t, test.expectedFaucetBaseURLFlagValue, faucetBaseURLFlag.Value.String())
			expectedLogString = fmt.Sprintf(flagLogFmt, flags.FlagFaucetBaseURL, test.expectedFaucetBaseURLFlagValue)
			require.Contains(t, testLogBuffer.String(), expectedLogString)
		})
	}
}

func TestNetworkRelatedFlags_DoesNotOverrideExistingFlags(t *testing.T) {
	testCmd := &cobra.Command{
		Use:     "test",
		Short:   "Test",
		Long:    `Test`,
		PreRunE: testPreRunE,
		RunE:    testRunE,
	}

	// Register relevant flags for testing
	testCmd.Flags().String(flags.FlagNetwork, "", "network flag")
	testCmd.Flags().String(cosmosflags.FlagGRPC, "", "grpc addr flag")
	testCmd.Flags().String(cosmosflags.FlagGRPCInsecure, "", "grpc insecure flag")
	testCmd.Flags().String(flags.FlagFaucetBaseURL, "", "faucet base url flag")
	cosmosflags.AddTxFlagsToCmd(testCmd)

	// Pre-set some flags to custom values
	customChainID := "custom-chain-123"
	customGRPCAddr := "custom-grpc:9090"

	err := testCmd.Flag(cosmosflags.FlagChainID).Value.Set(customChainID)
	require.NoError(t, err)
	// Simulate flag being set via command line by marking it as changed
	testCmd.Flags().Set(cosmosflags.FlagChainID, customChainID)

	err = testCmd.Flag(cosmosflags.FlagGRPC).Value.Set(customGRPCAddr)
	require.NoError(t, err)
	// Simulate flag being set via command line by marking it as changed
	testCmd.Flags().Set(cosmosflags.FlagGRPC, customGRPCAddr)

	// Execute with LocalNet network flag
	testCmd.SetArgs([]string{
		fmt.Sprintf("--%s=%s", flags.FlagNetwork, flags.LocalNetworkName),
	})

	err = testCmd.ExecuteContext(context.Background())
	require.NoError(t, err)

	// Verify pre-set flags were NOT overridden
	chainIDFlag := testCmd.Flag(cosmosflags.FlagChainID)
	require.Equal(t, customChainID, chainIDFlag.Value.String(),
		"pre-set chain-id should not be overridden")

	grpcFlag := testCmd.Flag(cosmosflags.FlagGRPC)
	require.Equal(t, customGRPCAddr, grpcFlag.Value.String(),
		"pre-set grpc address should not be overridden")

	// Verify flags that weren't pre-set WERE set by network logic
	nodeFlag := testCmd.Flag(cosmosflags.FlagNode)
	require.Equal(t, pocket.LocalNetRPCURL, nodeFlag.Value.String(),
		"node flag should be set by network logic")
}
