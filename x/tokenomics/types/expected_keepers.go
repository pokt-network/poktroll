//go:generate mockgen -destination ../../../testutil/tokenomics/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,ProofKeeper,SharedKeeper,SessionKeeper

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
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
	GetApplication(ctx context.Context, appAddr string) (app application.Application, found bool)
	SetApplication(ctx context.Context, app application.Application)
}

type ProofKeeper interface {
	GetAllClaims(ctx context.Context) []proof.Claim
	RemoveClaim(ctx context.Context, sessionId, supplierAddr string)
	GetProof(ctx context.Context, sessionId, supplierAddr string) (proof proof.Proof, isProofFound bool)
	RemoveProof(ctx context.Context, sessionId, supplierAddr string)

	AllClaims(ctx context.Context, req *proof.QueryAllClaimsRequest) (*proof.QueryAllClaimsResponse, error)

	// Only used for testing & simulation
	UpsertClaim(ctx context.Context, claim proof.Claim)
	UpsertProof(ctx context.Context, claim proof.Proof)

	GetParams(ctx context.Context) proof.Params
	SetParams(ctx context.Context, params proof.Params) error
}

type SharedKeeper interface {
	GetParams(ctx context.Context) shared.Params
	SetParams(ctx context.Context, params shared.Params) error

	GetProofWindowCloseHeight(ctx context.Context, queryHeight int64) int64
}

type SessionKeeper interface {
	GetSession(context.Context, *session.QueryGetSessionRequest) (*session.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
	StoreBlockHash(ctx context.Context)
}

type SupplierKeeper interface {
	GetSupplier(ctx context.Context, supplierAddr string) (supplier shared.Supplier, found bool)
	SetSupplier(ctx context.Context, supplier shared.Supplier)
}
