//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/migration/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,SharedKeeper,GatewayKeeper,ApplicationKeeper,SupplierKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/gateway/types"
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
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type GatewayKeeper interface {
	GetGateway(ctx context.Context, address string) (gateway types.Gateway, found bool)
	SetGateway(ctx context.Context, gateway types.Gateway)
}

type ApplicationKeeper interface {
	GetApplication(ctx context.Context, appAddr string) (app apptypes.Application, found bool)
	SetApplication(ctx context.Context, application apptypes.Application)
}
type SupplierKeeper interface {
	GetSupplier(ctx context.Context, supplierOperatorAddr string) (supplier sharedtypes.Supplier, found bool)
	SetSupplier(ctx context.Context, supplier sharedtypes.Supplier)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
}
