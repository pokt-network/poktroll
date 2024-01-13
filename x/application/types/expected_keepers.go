package types

//go:generate mockgen -destination ../../../testutil/application/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,GatewayKeeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	DelegateCoinsFromAccountToModule(
		ctx sdk.Context,
		senderAddr sdk.AccAddress,
		recipientModule string,
		amt sdk.Coins,
	) error
	UndelegateCoinsFromModuleToAccount(
		ctx sdk.Context,
		senderModule string,
		recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
}

// GatewayKeeper defines the expected interface needed to retrieve gateway information.
type GatewayKeeper interface {
	GetGateway(ctx sdk.Context, addr string) (gatewaytypes.Gateway, bool)
}
