//go:generate mockgen -destination=../../../testutil/proof/mocks/expected_keepers_mock.go -package=mocks . BankKeeper,SessionKeeper,ApplicationKeeper,AccountKeeper,SharedKeeper,ServiceKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type SessionKeeper interface {
	GetSession(context.Context, *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
	StoreBlockHash(ctx context.Context)
}

type SupplierKeeper interface {
	SetSupplier(context.Context, sharedtypes.Supplier)
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
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// We use the bankkeeper SendXXX instead of DelegateXX methods
	// because their purpose is to "escrow" funds on behalf of an account rather
	// than "delegate" funds from one account to another which is more closely
	// linked to staking.
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ApplicationKeeper defines the expected application keeper to retrieve applications
type ApplicationKeeper interface {
	GetApplication(ctx context.Context, address string) (app apptypes.Application, found bool)
	GetAllApplications(ctx context.Context) []apptypes.Application
	SetApplication(context.Context, apptypes.Application)
}

// SharedKeeper defines the expected interface needed to retrieve shared information.
type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
}

// ServiceKeeper defines the expected interface for the Service module.
type ServiceKeeper interface {
	GetService(ctx context.Context, serviceID string) (sharedtypes.Service, bool)
	GetRelayMiningDifficulty(ctx context.Context, serviceID string) (servicetypes.RelayMiningDifficulty, bool)
	// Only used for testing & simulation
	SetService(ctx context.Context, service sharedtypes.Service)
	SetRelayMiningDifficulty(ctx context.Context, relayMiningDifficulty servicetypes.RelayMiningDifficulty)
}
