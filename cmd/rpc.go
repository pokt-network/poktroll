package cmd

import (
	"fmt"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/flags"
)

// ParseAndSetNetworkRelatedFlags checks if the --network flag is set (i.e. not empty-string).
// If so, set the following flags according to their hard-coded network-specific values:
// * --chain-id
// * --node
// * --grpc-addr
func ParseAndSetNetworkRelatedFlags(cmd *cobra.Command) error {
	networkStr, err := cmd.Flags().GetString(flags.FlagNetwork)
	if err != nil {
		return err
	}

	switch networkStr {
	case "":
		// No network flag was provided, so we don't need to set any flags.
		return nil

	// LocalNet
	case flags.LocalNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.LocalNetChainId, pocket.LocalNetRPCURL, pocket.LocalNetGRPCAddr, "true", pocket.LocalNetFaucetBaseURL)

	// Alpha TestNet
	case flags.AlphaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.AlphaTestNetChainId, pocket.AlphaTestNetRPCURL, pocket.AlphaNetGRPCAddr, "false", pocket.AlphaTestNetFaucetBaseURL)

	// Beta TestNet
	case flags.BetaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.BetaTestNetChainId, pocket.BetaTestNetRPCURL, pocket.BetaNetGRPCAddr, "false", pocket.BetaTestNetFaucetBaseURL)

	// MainNet
	case flags.MainNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.MainNetChainId, pocket.MainNetRPCURL, pocket.MainNetGRPCAddr, "false", pocket.MainNetFaucetBaseURL)

	default:
		return fmt.Errorf("unknown --network specified %q", networkStr)
	}
}

// setNetworkRelatedFlags sets the following flags according to the given arguments
// ONLY if they have not already been set AND are registered on the given command:
// * --chain-id
// * --node
// * --grpc-addr
// * --grpc-insecure
// * --faucet-base-url

func setNetworkRelatedFlags(cmd *cobra.Command, chainId, nodeUrl, grpcAddr, grpcInsecure, faucetBaseUrl string) error {
	// --chain-id flag
	if chainIDFlag := cmd.Flags().Lookup(cosmosflags.FlagChainID); chainIDFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagChainID) {
			if err := cmd.Flags().Set(cosmosflags.FlagChainID, chainId); err != nil {
				return err
			}
		}
	}

	// --node flag
	if nodeFlag := cmd.Flags().Lookup(cosmosflags.FlagNode); nodeFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagNode) {
			if err := cmd.Flags().Set(cosmosflags.FlagNode, nodeUrl); err != nil {
				return err
			}
		}
	}

	// --grpc-addr flag
	if grpcFlag := cmd.Flags().Lookup(cosmosflags.FlagGRPC); grpcFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagGRPC) {
			if err := cmd.Flags().Set(cosmosflags.FlagGRPC, grpcAddr); err != nil {
				return err
			}
		}
	}

	// --grpc-insecure flag
	if grpcInsecureFlag := cmd.Flags().Lookup(cosmosflags.FlagGRPCInsecure); grpcInsecureFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagGRPCInsecure) {
			if err := cmd.Flags().Set(cosmosflags.FlagGRPCInsecure, grpcInsecure); err != nil {
				return err
			}
		}
	}

	// --faucet-base-url flag
	if faucetBaseURLFlag := cmd.Flags().Lookup(flags.FlagFaucetBaseURL); faucetBaseURLFlag != nil {
		if !cmd.Flags().Changed(flags.FlagFaucetBaseURL) {
			if err := cmd.Flags().Set(flags.FlagFaucetBaseURL, faucetBaseUrl); err != nil {
				return err
			}
		}
	}

	return nil
}
