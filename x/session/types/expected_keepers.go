//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/session/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,SupplierKeeper,SharedKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/pocket/x/application/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI // only used for simulation
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
}

// ApplicationKeeper defines the expected application keeper to retrieve applications
type ApplicationKeeper interface {
	GetApplication(ctx context.Context, address string) (app apptypes.Application, found bool)
}

// SupplierKeeper defines the expected interface needed to retrieve suppliers
type SupplierKeeper interface {
	GetAllSuppliers(ctx context.Context) (suppliers []sharedtypes.Supplier)
}

// SharedKeeper defines the expected interface needed to retrieve shared parameters
type SharedKeeper interface {
	GetParams(ctx context.Context) (params sharedtypes.Params)
}
