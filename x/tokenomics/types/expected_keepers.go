//go:generate mockgen -destination ../../../testutil/tokenomics/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,ProofKeeper,SharedKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI // only used for simulation
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	// We use the bankkeeper SendXXX instead of DelegateXX methods
	// because their purpose is to "escrow" funds on behalf of an account rather
	// than "delegate" funds from one account to another which is more closely
	// linked to staking.
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	Balance(context.Context, *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error)
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
	UpsertProof(ctx context.Context, claim prooftypes.Proof)

	GetParams(ctx context.Context) prooftypes.Params
	SetParams(ctx context.Context, params prooftypes.Params) error
}

type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
	SetParams(ctx context.Context, params sharedtypes.Params) error

	GetProofWindowCloseHeight(ctx context.Context, queryHeight int64) int64
}

type SupplierKeeper interface {
	GetSupplier(ctx context.Context, supplierAddr string) (supplier sharedtypes.Supplier, found bool)
	SetSupplier(ctx context.Context, supplier sharedtypes.Supplier)
}
