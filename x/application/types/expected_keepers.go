//go:generate mockgen -destination ../../../testutil/application/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,GatewayKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// GatewayKeeper defines the expected interface needed to retrieve gateway information.
type GatewayKeeper interface {
	GetGateway(ctx context.Context, addr string) (gatewaytypes.Gateway, bool)
}
