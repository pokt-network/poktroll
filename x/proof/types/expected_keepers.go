//go:generate mockgen -destination=../../../testutil/proof/mocks/expected_keepers_mock.go -package=mocks . BankKeeper,SessionKeeper,ApplicationKeeper,AccountKeeper,SharedKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
)

type SessionKeeper interface {
	GetSession(context.Context, *session.QueryGetSessionRequest) (*session.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
	StoreBlockHash(ctx context.Context)
}

type SupplierKeeper interface {
	SetSupplier(context.Context, shared.Supplier)
}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
	SetAccount(context.Context, sdk.AccountI)
	// Return a new account with the next account number and the specified address. Does not save the new account to the store.
	NewAccountWithAddress(context.Context, sdk.AccAddress) sdk.AccountI
	// Fetch the next account number, and increment the internal counter.
	NextAccountNumber(context.Context) uint64
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	// We use the bankkeeper SendXXX instead of DelegateXX methods
	// because their purpose is to "escrow" funds on behalf of an account rather
	// than "delegate" funds from one account to another which is more closely
	// linked to staking.
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ApplicationKeeper defines the expected application keeper to retrieve applications
type ApplicationKeeper interface {
	GetApplication(ctx context.Context, address string) (app application.Application, found bool)
	GetAllApplications(ctx context.Context) []application.Application
	SetApplication(context.Context, application.Application)
}

// SharedKeeper defines the expected interface needed to retrieve shared information.
type SharedKeeper interface {
	GetParams(ctx context.Context) shared.Params
}
