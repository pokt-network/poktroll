package types

//go:generate mockgen -destination ../../../testutil/session/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,SupplierKeeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type ApplicationKeeper interface {
	GetApplication(ctx context.Context, address string) (app apptypes.Application, found bool)
}

type SupplierKeeper interface {
	GetAllSupplier(ctx context.Context) (suppliers []sharedtypes.Supplier)
}
