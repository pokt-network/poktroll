package config

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
)

// GetTxClientGasAndFeesOptionsFromFlags returns a slice of TxClientOptions which
// are derived from the provided command flags and/or config, using the
// following precedence:
//
// 1. If a fee is specified, it overrides all gas settings, only returns:
//   - WithFeeAmount
//     In this case, the fee amount is explicitly set (by the tx author);
//     therefore, while the gas limit will still be respected by the CheckTx
//     ABCI method, the gas limit, adjustment, and prices are NOT used to
//     calculate the fee.
//
// 2. Otherwise, the following gas related options are returned:
//   - WithGasPrices
//   - WithGasAdjustment
//   - WithGasSetting
//     In this case, the fee is calculated, given by: `fees = gas_limit * gas_adjustment * gas_prices`.
//
// gasSettingStr is expected to be either "auto", empty string, or a string integer:
//   - "auto": The gas limit will be determined by simulating the transaction
//     and multiplying by the gas adjustment.
//   - "<integer>": The gas limit will be set to the integer amount represented
//     by the string.
//   - "": The gas limit will be set to flags.DefaultGasLimit (200000).
func GetTxClientGasAndFeesOptionsFromFlags(cmd *cobra.Command, gasSettingStr string) ([]client.TxClientOption, error) {
	// Retrieve the explicitly specified fee amount from the command flags.
	feesStr, err := cmd.Flags().GetString(flags.FlagFees)
	if err != nil {
		if !strings.Contains(err.Error(), "flag accessed but not defined") {
			return nil, err
		}

		// This error indicates that the fees flag not registered and can be safely ignored.
		// Explicitly setting feesStr to an empty string to guarantee correct conditional branching.
		feesStr = ""
	}

	// If a fee is specified, it overrides all gas settings and returns immediately.
	if feesStr != "" {
		feeAmount, parseErr := types.ParseDecCoins(feesStr)
		if parseErr != nil {
			return nil, err
		}

		return []client.TxClientOption{
			tx.WithFeeAmount(&feeAmount),
		}, nil
	}

	// Retrieve all gas related options from the command flags.
	gasPriceStr, err := cmd.Flags().GetString(flags.FlagGasPrices)
	if err != nil {
		return nil, err
	}
	gasPrices, err := types.ParseDecCoins(gasPriceStr)
	if err != nil {
		return nil, err
	}
	gasAdjustment, err := cmd.Flags().GetFloat64(flags.FlagGasAdjustment)
	if err != nil {
		return nil, err
	}

	// The RelayMiner always uses tx simulation to estimate the gas since this
	// will be variable depending on the tx being sent.
	// Always use the "auto" gas setting for the RelayMiner.
	gasSetting, err := flags.ParseGasSetting(gasSettingStr)
	if err != nil {
		return nil, err
	}

	// Onchain fees (i.e. gas) can only be paid in upokt.
	for _, gasPrice := range gasPrices {
		if gasPrice.Denom != volatile.DenomuPOKT {
			// TODO_TECHDEBT(red-0ne): Allow other gas prices denominations once supported (e.g. mPOKT, POKT)
			// See https://docs.cosmos.network/main/build/architecture/adr-024-coin-metadata#decision
			return nil, fmt.Errorf("only gas prices with %s denom are supported", volatile.DenomuPOKT)
		}
	}

	// Return the gas and fee related options.
	return []client.TxClientOption{
		tx.WithGasPrices(&gasPrices),
		tx.WithGasAdjustment(gasAdjustment),
		tx.WithGasSetting(&gasSetting),
	}, nil
}
