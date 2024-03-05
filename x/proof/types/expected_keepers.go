//go:generate mockgen -destination=../../../testutil/proof/mocks/expected_keepers_mock.go -package=mocks . BankKeeper,SessionKeeper,ApplicationKeeper,AccountKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type SessionKeeper interface {
	GetSession(context.Context, *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
}

type SupplierKeeper interface {
	SetSupplier(context.Context, sharedtypes.Supplier)
}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
	DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ApplicationKeeper defines the expected application keeper to retrieve applications
type ApplicationKeeper interface {
	GetApplication(ctx context.Context, address string) (app apptypes.Application, found bool)
	GetAllApplications(ctx context.Context) []apptypes.Application
	SetApplication(context.Context, apptypes.Application)
}
