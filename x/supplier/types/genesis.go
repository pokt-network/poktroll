package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		SupplierList: []sharedtypes.Supplier{},
		ClaimList: []Claim{},
// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in supplier
	supplierIndexMap := make(map[string]struct{})
	for _, supplier := range gs.SupplierList {
		index := string(SupplierKey(supplier.Address))
		if _, ok := supplierIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for supplier")
		}
		supplierIndexMap[index] = struct{}{}
	}

	// Check that the stake value for the suppliers is valid
	for _, supplier := range gs.SupplierList {
		// TODO_TECHDEBT: Consider creating shared helpers across the board for stake validation,
		// similar to how we have `ValidateAppServiceConfigs` below
		if supplier.Stake == nil {
			return sdkerrors.Wrapf(ErrSupplierInvalidStake, "nil stake amount for supplier")
		}
		stake, err := sdk.ParseCoinNormalized(supplier.Stake.String())
		if !stake.IsValid() {
			return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid stake amount for supplier %v; (%v)", supplier.Stake, stake.Validate())
		}
		if err != nil {
			return sdkerrors.Wrapf(ErrSupplierInvalidStake, "cannot parse stake amount for supplier %v; (%v)", supplier.Stake, err)
		}
		if stake.IsZero() || stake.IsNegative() {
			return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid stake amount for supplier: %v <= 0", supplier.Stake)
		}
		if stake.Denom != "upokt" {
			return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid stake amount denom for supplier %v", supplier.Stake)
		}

		// Valid the application service configs
		// Validate the application service configs
		if err := servicehelpers.ValidateSupplierServiceConfigs(supplier.Services); err != nil {
			return sdkerrors.Wrapf(ErrSupplierInvalidServiceConfig, err.Error())
		}
	}

	// Check for duplicated index in claim
claimIndexMap := make(map[string]struct{})

for _, elem := range gs.ClaimList {
	index := string(ClaimKey(elem.Index))
	if _, ok := claimIndexMap[index]; ok {
		return fmt.Errorf("duplicated index for claim")
	}
	claimIndexMap[index] = struct{}{}
}
// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
