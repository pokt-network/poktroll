//go:generate mockgen -destination ../../../testutil/tokenomics/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,ProofKeeper,SharedKeeper,SessionKeeper,SupplierKeeper,ServiceKeeper

package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	// Only used for testing & simulation
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(context.Context, types.AccountI)
	// Return a new account with the next account number and the specified address. Does not save the new account to the store.
	NewAccountWithAddress(context.Context, sdk.AccAddress) sdk.AccountI
	// Fetch the next account number, and increment the internal counter.
	NextAccountNumber(context.Context) uint64
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
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	Balance(context.Context, *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error)
}

type ApplicationKeeper interface {
	GetApplication(ctx context.Context, appAddr string) (app apptypes.Application, found bool)
	SetApplication(ctx context.Context, app apptypes.Application)
	GetAllApplications(ctx context.Context) []apptypes.Application
	UnbondApplication(ctx context.Context, app *apptypes.Application) error
	EndBlockerUnbondApplications(ctx context.Context) error
}

type ProofKeeper interface {
	GetAllClaims(ctx context.Context) []prooftypes.Claim
	RemoveClaim(ctx context.Context, sessionId, supplierOperatorAddr string)
	GetProof(ctx context.Context, sessionId, supplierOperatorAddr string) (proof prooftypes.Proof, isProofFound bool)
	RemoveProof(ctx context.Context, sessionId, supplierOperatorAddr string)

	AllClaims(ctx context.Context, req *prooftypes.QueryAllClaimsRequest) (*prooftypes.QueryAllClaimsResponse, error)
	EnsureValidProof(ctx context.Context, proof *prooftypes.Proof) error
	ProofRequirementForClaim(ctx context.Context, claim *prooftypes.Claim) (prooftypes.ProofRequirementReason, error)

	// Only used for testing & simulation
	GetAllProofs(ctx context.Context) []prooftypes.Proof
	UpsertClaim(ctx context.Context, claim prooftypes.Claim)
	UpsertProof(ctx context.Context, claim prooftypes.Proof)
	GetParams(ctx context.Context) prooftypes.Params
	SetParams(ctx context.Context, params prooftypes.Params) error
}

type SharedKeeper interface {
	GetParams(ctx context.Context) sharedtypes.Params
	SetParams(ctx context.Context, params sharedtypes.Params) error

	GetSessionEndHeight(ctx context.Context, queryHeight int64) int64
	GetProofWindowCloseHeight(ctx context.Context, queryHeight int64) int64
}

type SessionKeeper interface {
	GetSession(context.Context, *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
	StoreBlockHash(ctx context.Context)
	GetParams(ctx context.Context) sessiontypes.Params
}

type SupplierKeeper interface {
	GetSupplier(ctx context.Context, supplierOperatorAddr string) (supplier sharedtypes.Supplier, found bool)
	GetAllSuppliers(ctx context.Context) (suppliers []sharedtypes.Supplier)
	SetSupplier(ctx context.Context, supplier sharedtypes.Supplier)
	GetParams(ctx context.Context) suppliertypes.Params
}

type ServiceKeeper interface {
	GetService(ctx context.Context, serviceID string) (sharedtypes.Service, bool)
	GetRelayMiningDifficulty(ctx context.Context, serviceID string) (servicetypes.RelayMiningDifficulty, bool)
	UpdateRelayMiningDifficulty(ctx context.Context, relaysPerServiceMap map[string]uint64) (map[string]servicetypes.RelayMiningDifficulty, error)
	// Only used for testing & simulation
	SetService(ctx context.Context, service sharedtypes.Service)
}
