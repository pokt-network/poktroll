//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/migration/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,SharedKeeper,ApplicationKeeper,SupplierKeeper

package types

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	GetParams(context.Context) authtypes.Params
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type ApplicationKeeper interface {
	GetApplication(ctx context.Context, appAddr string) (app apptypes.Application, found bool)
	SetApplication(ctx context.Context, application apptypes.Application)
	GetParams(ctx context.Context) apptypes.Params
	StakeApplication(ctx context.Context, logger cosmoslog.Logger, msg *apptypes.MsgStakeApplication) (*apptypes.Application, error)
}
type SupplierKeeper interface {
	GetSupplier(ctx context.Context, supplierOperatorAddr string) (supplier sharedtypes.Supplier, found bool)
	SetSupplier(ctx context.Context, supplier sharedtypes.Supplier)
	StakeSupplier(ctx context.Context, logger cosmoslog.Logger, msg *suppliertypes.MsgStakeSupplier) (*sharedtypes.Supplier, error)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
}
