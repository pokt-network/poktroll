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
		return setNetworkRelatedFlags(cmd, pocket.LocalNetChainId, pocket.LocalNetRPCURL, pocket.LocalNetGRPCAddr)

	// Shannon Alpha TestNet
	case flags.AlphaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.AlphaTestNetChainId, pocket.AlphaTestNetRPCURL, pocket.AlphaNetGRPCAddr)

	// Shannon Beta TestNet
	case flags.BetaNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.BetaTestNetChainId, pocket.BetaTestNetRPCURL, pocket.BetaNetGRPCAddr)

	// MainNet
	case flags.MainNetworkName:
		return setNetworkRelatedFlags(cmd, pocket.MainNetChainId, pocket.MainNetRPCURL, pocket.MainNetGRPCAddr)

	default:
		return fmt.Errorf("unknown --network specified %q", networkStr)
	}
}

// setNetworkRelatedFlags sets the following flags according to the given arguments
// ONLY if they have not already been set:
// * --chain-id
// * --node
// * --grpc-addr
//
// DEV_NOTE: --grpc-insecure is also set, but ONLY for LocalNet.
func setNetworkRelatedFlags(cmd *cobra.Command, chainId, nodeUrl, grpcAddr string) error {
	if !cmd.Flags().Changed(cosmosflags.FlagChainID) {
		if err := cmd.Flags().Set(cosmosflags.FlagChainID, chainId); err != nil {
			return err
		}
	}

	if !cmd.Flags().Changed(cosmosflags.FlagNode) {
		if err := cmd.Flags().Set(cosmosflags.FlagNode, nodeUrl); err != nil {
			return err
		}
	}

	// ONLY set --grpc-addr flag if it is registered on cmd.
	if grpcFlag := cmd.Flags().Lookup(cosmosflags.FlagGRPC); grpcFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagGRPC) {
			if err := cmd.Flags().Set(cosmosflags.FlagGRPC, grpcAddr); err != nil {
				return err
			}
		}
	}

	// TODO_TECHDEBT(@bryanchriswhite): This should only be set on transactions, not queries.
	// Also set --grpc-insecure flag, but ONLY for LocalNet.
	// if chainId == pocket.LocalNetChainId {
	// 	if !cmd.Flags().Changed(cosmosflags.FlagGRPCInsecure) {
	// 		if err := cmd.Flags().Set(cosmosflags.FlagGRPCInsecure, "true"); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	return nil
}
