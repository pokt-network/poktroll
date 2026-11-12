//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/supplier/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,SessionKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
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
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BlockedAddr(addr sdk.AccAddress) bool
}

// SharedKeeper defines the expected interface needed to retrieve shared information.
type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
	// GetParamsAtHeight returns the session params that were effective at the given height.
	// This is used for historical session queries to ensure correct session boundaries.
	GetParamsAtHeight(ctx context.Context, queryHeight int64) sharedtypes.Params
	GetSessionEndHeight(ctx context.Context, queryHeight int64) int64
	// IterateParamsHistoryReverse walks the params history from fromHeight back.
	IterateParamsHistoryReverse(
		ctx context.Context,
		fromHeight int64,
		fn func(effectiveHeight int64, params sharedtypes.Params) (stop bool),
	)
}

// ServiceKeeper defines the expected interface for the Service module.
type ServiceKeeper interface {
	GetService(ctx context.Context, serviceId string) (sharedtypes.Service, bool)
	// IsSupplierAllowed reports whether the supplier's owner or operator may serve
	// the service: always true for a service with an empty supplier allowlist.
	IsSupplierAllowed(ctx context.Context, serviceId, ownerAddress, operatorAddress string) bool
	// GetPendingAllowlistSweeps returns the services whose staked suppliers still
	// need to be checked against their (changed) supplier allowlist.
	GetPendingAllowlistSweeps(ctx context.Context) []servicetypes.AllowlistSweep
	SetAllowlistSweep(ctx context.Context, sweep servicetypes.AllowlistSweep)
	ClearAllowlistSweep(ctx context.Context, serviceId string)
}

// SessionKeeper defines the expected interface for the Session module.
type SessionKeeper interface {
	GetSession(ctx context.Context, req *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
}
