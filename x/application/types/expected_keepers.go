//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/application/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,GatewayKeeper,SharedKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/pocket/x/gateway/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
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

// GatewayKeeper defines the expected interface needed to retrieve gateway information.
type GatewayKeeper interface {
	GetGateway(ctx context.Context, addr string) (gatewaytypes.Gateway, bool)
	GetAllGateways(ctx context.Context) []gatewaytypes.Gateway
}

// SharedKeeper defines the expected interface needed to retrieve shared information.
type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
	GetSessionEndHeight(ctx context.Context, queryHeight int64) int64
}
