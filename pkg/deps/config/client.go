package config

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/tx"
)

// GetTxClientGasAndFeesOptions returns a slice of TxClientOptions which
// are derived from the provided command flags and/or config, using the
// following precedence:
// 1. If a fee is specified, it overrides all gas settings, only returns:
//   - WithFeeAmount
//
// 2. Otherwise, the following gas related options are returned:
//   - WithGasPrices
//   - WithGasAdjustment
//   - WithGasSetting
func GetTxClientGasAndFeesOptions(cmd *cobra.Command) ([]client.TxClientOption, error) {
	feesStr, err := cmd.Flags().GetString(flags.FlagFees)
	if err != nil {
		return nil, err
	}

	// If a fee is specified, it overrides all gas settings.
	if feesStr != "" {
		feeAmount, err := types.ParseDecCoins(feesStr)
		if err != nil {
			return nil, err
		}

		return []client.TxClientOption{
			tx.WithFeeAmount(&feeAmount),
		}, nil
	}

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
	gasSetting, err := flags.ParseGasSetting("auto")
	if err != nil {
		return nil, err
	}

	// Ensure that the gas prices include upokt
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
