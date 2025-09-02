package cmd

import (
	"fmt"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/flags"
)

// ParseAndSetNetworkRelatedFlags checks if the --network flag is set (i.e. not empty-string).
// If so, sets the following flags according to their hard-coded network-specific values:
// • --chain-id
// • --node
// • --grpc-addr
// • --grpc-insecure
// • --faucet-base-url
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
		return setNetworkRelatedFlags(
			cmd,
			pocket.LocalNetChainId,
			pocket.LocalNetRPCURL,
			pocket.LocalNetGRPCAddr,
			flags.BooleanTrueValue,
			pocket.LocalNetFaucetBaseURL,
		)

	// Alpha TestNet
	case flags.AlphaNetworkName:
		return setNetworkRelatedFlags(
			cmd,
			pocket.AlphaTestNetChainId,
			pocket.AlphaTestNetRPCURL,
			pocket.AlphaNetGRPCAddr,
			flags.BooleanFalseValue,
			pocket.AlphaTestNetFaucetBaseURL,
		)

	// Beta TestNet
	case flags.BetaNetworkName:
		return setNetworkRelatedFlags(
			cmd,
			pocket.BetaTestNetChainId,
			pocket.BetaTestNetRPCURL,
			pocket.BetaNetGRPCAddr,
			flags.BooleanFalseValue,
			pocket.BetaTestNetFaucetBaseURL,
		)

	// MainNet
	case flags.MainNetworkName:
		return setNetworkRelatedFlags(
			cmd,
			pocket.MainNetChainId,
			pocket.MainNetRPCURL,
			pocket.MainNetGRPCAddr,
			flags.BooleanFalseValue,
			pocket.MainNetFaucetBaseURL,
		)

	default:
		return fmt.Errorf("unknown --network specified %q", networkStr)
	}
}

// setNetworkRelatedFlags sets network-specific flags if not already set and registered:
// • --chain-id: Blockchain network identifier
// • --node: RPC endpoint URL
// • --grpc-addr: gRPC endpoint address
// • --grpc-insecure: Whether to use insecure gRPC connection
// • --faucet-base-url: Faucet service base URL

func setNetworkRelatedFlags(cmd *cobra.Command, chainId, nodeUrl, grpcAddr, grpcInsecure, faucetBaseUrl string) error {
	// --chain-id flag
	if chainIDFlag := cmd.Flag(cosmosflags.FlagChainID); chainIDFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagChainID) {
			if err := chainIDFlag.Value.Set(chainId); err != nil {
				return err
			}
		}
	}

	// --node flag
	if nodeFlag := cmd.Flag(cosmosflags.FlagNode); nodeFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagNode) {
			if err := nodeFlag.Value.Set(nodeUrl); err != nil {
				return err
			}
		}
	}

	// --grpc-addr flag
	if grpcFlag := cmd.Flag(cosmosflags.FlagGRPC); grpcFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagGRPC) {
			if err := grpcFlag.Value.Set(grpcAddr); err != nil {
				return err
			}
		}
	}

	// --grpc-insecure flag
	if grpcInsecureFlag := cmd.Flag(cosmosflags.FlagGRPCInsecure); grpcInsecureFlag != nil {
		if !cmd.Flags().Changed(cosmosflags.FlagGRPCInsecure) {
			if err := grpcInsecureFlag.Value.Set(grpcInsecure); err != nil {
				return err
			}
		}
	}

	// --faucet-base-url flag
	if faucetBaseURLFlag := cmd.Flag(flags.FlagFaucetBaseURL); faucetBaseURLFlag != nil {
		if !cmd.Flags().Changed(flags.FlagFaucetBaseURL) {
			if err := faucetBaseURLFlag.Value.Set(faucetBaseUrl); err != nil {
				return err
			}
		}
	}

	return nil
}
