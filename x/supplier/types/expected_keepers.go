//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/supplier/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,SessionKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/pocket/x/session/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// SharedKeeper defines the expected interface needed to retrieve shared information.
type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
	GetSessionEndHeight(ctx context.Context, queryHeight int64) int64
}

// ServiceKeeper defines the expected interface for the Service module.
type ServiceKeeper interface {
	GetService(ctx context.Context, serviceId string) (sharedtypes.Service, bool)
}

// SessionKeeper defines the expected interface for the Session module.
type SessionKeeper interface {
	GetSession(ctx context.Context, req *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
}
