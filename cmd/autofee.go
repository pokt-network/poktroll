package cmd

// Auto-Fee Implementation.
//
// This file implements an --auto-fee flag that works across all transaction commands in pocketd.
// The implementation follows a clear priority hierarchy to respect user intent while providing convenient automation.
//
// PRIORITY HIERARCHY (Highest to Lowest):
// 1. Explicit --fees flag (highest priority, overrides everything)
// 2. Manual --gas (not "auto") + --gas-prices combination
// 3. --auto-fee logic (when enabled and no explicit fees/gas set)
// 4. --gas-adjustment (always respected, defaults to 1.2 for auto-fee)
//
// USAGE EXAMPLES:
//   # Auto-fee with default settings
//   pocketd tx send ... --auto-fee
//
//   # Auto-fee with custom gas adjustment
//   pocketd tx send ... --auto-fee --gas-adjustment 1.5
//
//   # Explicit fees override auto-fee (auto-fee ignored with warning)
//   pocketd tx send ... --auto-fee --fees 1000upokt
//
//   # Manual gas calculation overrides auto-fee
//   pocketd tx send ... --auto-fee --gas 200000 --gas-prices 0.1upokt
//
// Usage & Integration:
// 1. Call AddAutoFeeFlag() on your root command during initialization
// 2. Call WrapTxCommand() on all transaction subcommands
// 3. Optionally set default gas prices via environment variable COSMOS_DEFAULT_GAS_PRICES

import (
	"errors"
	"fmt"

	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
)

const (
	// Default gas adjustment multiplier for auto-fee (50% buffer)
	DefaultAutoFeeGasAdjustment = 1.5

	// https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/app.toml
	DefaultGasPrices = "0.001upokt"
)

// AddAutoFeeFlag adds the --auto-fee flag to the root command
// Call this during your app's root command initialization
func AddAutoFeeFlag(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().Bool(
		flags.FlagAutoFee,
		false,
		"Automatically calculate and set transaction fees using gas simulation",
	)
}

// WrapTxCommand wraps a transaction command to handle auto-fee logic.
// Call this on all transaction subcommands during initialization.
func WrapTxCommand(cmd *cobra.Command) {
	if cmd.RunE == nil {
		return
	}

	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Apply auto-fee logic before executing the command
		if err := handleAutoFee(cmd); err != nil {
			return fmt.Errorf("auto-fee error: %w", err)
		}

		// Execute the original command
		return originalRunE(cmd, args)
	}
}

// WrapAllTxCommands recursively wraps all transaction commands in a command tree.
// Convenience function to wrap an entire tx command subtree.
func WrapAllTxCommands(txCmd *cobra.Command) {
	WrapTxCommand(txCmd)

	for _, subCmd := range txCmd.Commands() {
		WrapAllTxCommands(subCmd)
	}
}

// handleAutoFee implements the core auto-fee logic following the priority hierarchy.
func handleAutoFee(cmd *cobra.Command) error {
	// Validate flag combinations and show warnings
	if err := validateAndWarnFeeFlags(cmd); err != nil {
		return err
	}

	// Check if auto-fee is enabled
	autoFee, err := cmd.Flags().GetBool(flags.FlagAutoFee)
	if err != nil || !autoFee {
		return nil // Auto-fee not enabled, nothing to do
	}

	// Priority 1: Respect explicit --fees (highest priority)
	if fees, _ := cmd.Flags().GetString(cosmosflags.FlagFees); fees != "" {
		return nil // Explicit fees set, don't override
	}

	// Priority 2: Respect manual --gas + --gas-prices combination
	// Intentionally ignore errors here, as we default to auto-fee if flags are not set
	gas, _ := cmd.Flags().GetString(cosmosflags.FlagGas)
	gasPrices, _ := cmd.Flags().GetString(cosmosflags.FlagGasPrices)

	if gas != "" && gas != "auto" && gasPrices != "" {
		return nil // Manual gas calculation, don't override
	}

	// Priority 3: Apply auto-fee logic
	return applyAutoFeeCalculation(cmd)
}

// applyAutoFeeCalculation applies the auto-fee calculation logic.
func applyAutoFeeCalculation(cmd *cobra.Command) error {
	// Set gas to "auto" to enable simulation
	if err := cmd.Flags().Set(cosmosflags.FlagGas, "auto"); err != nil {
		return fmt.Errorf("failed to set gas to auto: %w", err)
	}

	// Priority 4: Handle gas adjustment (provide default if not set)
	gasAdj, err := cmd.Flags().GetFloat64(cosmosflags.FlagGasAdjustment)
	if err != nil || gasAdj <= 0 {
		if err := cmd.Flags().Set(cosmosflags.FlagGasAdjustment, fmt.Sprintf("%.1f", DefaultAutoFeeGasAdjustment)); err != nil {
			return fmt.Errorf("failed to set default gas adjustment: %w", err)
		}
	}

	// Ensure gas prices are available for fee calculation
	// Intentionally ignore errors here, as we default to auto-fee if flags are not set
	gasPrices, _ := cmd.Flags().GetString(cosmosflags.FlagGasPrices)
	if gasPrices == "" {
		if err := cmd.Flags().Set(cosmosflags.FlagGasPrices, DefaultGasPrices); err != nil {
			return fmt.Errorf("failed to set default gas prices: %w", err)
		}
	}

	return nil
}

// validateAndWarnFeeFlags validates flag combinations and shows helpful warnings
func validateAndWarnFeeFlags(cmd *cobra.Command) error {
	fees, _ := cmd.Flags().GetString(cosmosflags.FlagFees)
	gas, _ := cmd.Flags().GetString(cosmosflags.FlagGas)
	gasPrices, _ := cmd.Flags().GetString(cosmosflags.FlagGasPrices)
	autoFee, _ := cmd.Flags().GetBool(flags.FlagAutoFee)
	gasAdj, _ := cmd.Flags().GetFloat64(cosmosflags.FlagGasAdjustment)

	// Validate gas adjustment value
	if gasAdj < 0 {
		return errors.New("gas adjustment must be non-negative")
	}

	// Show warnings for conflicting flags when explicit fees are set
	if fees != "" {
		if autoFee {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s ignored when --%s is explicitly set\n", flags.FlagAutoFee, cosmosflags.FlagFees)
		}
		if gasPrices != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s ignored when --%s is explicitly set\n", cosmosflags.FlagGasPrices, cosmosflags.FlagFees)
		}
		if gas != "" && gas != "auto" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s ignored when --%s is explicitly set\n", cosmosflags.FlagGas, cosmosflags.FlagGas)
		}
	}

	// Show warning when manual gas calculation would override auto-fee
	if autoFee && fees == "" && gas != "" && gas != "auto" && gasPrices != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s ignored due to manual gas calculation (--%s + --%s)\n",
			flags.FlagAutoFee, cosmosflags.FlagGas, cosmosflags.FlagGasPrices)
	}

	// Validate that auto-fee has the prerequisites
	if autoFee && fees == "" && (gas == "" || gas == "auto") && gasPrices == "" {
		return fmt.Errorf("--%s requires gas prices to be available via --%s flag", flags.FlagAutoFee, cosmosflags.FlagGasPrices)
	}

	return nil
}
