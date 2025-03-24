package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		SupplierList: []sharedtypes.Supplier{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in supplier
	supplierOperatorAddrMap := make(map[string]struct{})
	for _, supplier := range gs.SupplierList {
		supplierOperatorAddr := string(SupplierOperatorKey(supplier.OperatorAddress))
		if _, ok := supplierOperatorAddrMap[supplierOperatorAddr]; ok {
			return fmt.Errorf("duplicated index for supplier")
		}
		supplierOperatorAddrMap[supplierOperatorAddr] = struct{}{}
	}

	// Check that the stake value for the suppliers is valid
	for _, supplier := range gs.SupplierList {
		// TODO_TECHDEBT: Consider creating shared helpers across the board for stake validation,
		// similar to how we have `ValidateAppServiceConfigs` below
		if supplier.Stake == nil {
			return ErrSupplierInvalidStake.Wrapf("nil stake amount for supplier")
		}
		stake, err := sdk.ParseCoinNormalized(supplier.Stake.String())
		if !stake.IsValid() {
			return ErrSupplierInvalidStake.Wrapf("invalid stake amount for supplier %v; (%v)", supplier.Stake, stake.Validate())
		}
		if err != nil {
			return ErrSupplierInvalidStake.Wrapf("cannot parse stake amount for supplier %v; (%v)", supplier.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return ErrSupplierInvalidStake.Wrapf("invalid stake amount for supplier: %v <= 0", supplier.Stake)
		}
		if stake.Denom != "upokt" {
			return ErrSupplierInvalidStake.Wrapf("invalid stake amount denom for supplier %v", supplier.Stake)
		}

		// Validate the application service configs
		if err := sharedtypes.ValidateSupplierServiceConfigs(supplier.Services); err != nil {
			return ErrSupplierInvalidServiceConfig.Wrapf("%s", err.Error())
		}

		// Validate the supplier owner address
		if _, err := sdk.AccAddressFromBech32(supplier.OwnerAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid supplier owner address %v", err.Error())
		}

		// Validate the supplier operator address
		if _, err := sdk.AccAddressFromBech32(supplier.OperatorAddress); err != nil {
			return ErrSupplierInvalidAddress.Wrapf("invalid supplier operator address %v", err.Error())
		}

	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
