package types

//go:generate mockgen -destination ../../../testutil/session/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,SupplierKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface{}

// ApplicationKeeper defines the expected application keeper to retrieve applications
type ApplicationKeeper interface {
	GetApplication(ctx sdk.Context, address string) (app apptypes.Application, found bool)
}

// SupplierKeeper defines the expected supplier keeper to retrieve suppliers
type SupplierKeeper interface {
	GetAllSupplier(ctx sdk.Context) (suppliers []sharedtypes.Supplier)
}
