//go:generate mockgen -destination ../../../testutil/tokenomics/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,ProofKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI // only used for simulation
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	// TODO_IN_THIS_PR: Update all the comments on why we use Send instead of Delegate for all expected_keepers.
	SendCoinsFromModuleToAccount(
		ctx context.Context,
		senderModule string,
		recipientAddr sdk.AccAddress,
		amt sdk.Coins,
	) error
}

type ApplicationKeeper interface {
	GetApplication(ctx context.Context, appAddr string) (app apptypes.Application, found bool)
	SetApplication(ctx context.Context, app apptypes.Application)
}

type ProofKeeper interface {
	GetAllClaims(ctx context.Context) []prooftypes.Claim
	RemoveClaim(ctx context.Context, sessionId, supplierAddr string)
	GetProof(ctx context.Context, sessionId, supplierAddr string) (proof prooftypes.Proof, isProofFound bool)
	RemoveProof(ctx context.Context, sessionId, supplierAddr string)

	// Only used for testing & simulation
	UpsertClaim(ctx context.Context, claim prooftypes.Claim)
}
