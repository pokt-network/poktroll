//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/tokenomics/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper,ApplicationKeeper,ProofKeeper,SharedKeeper,SessionKeeper,SupplierKeeper,ServiceKeeper,StakingKeeper

package types

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	// Getters
	GetAccount(ctx context.Context, addr cosmostypes.AccAddress) cosmostypes.AccountI
	NewAccountWithAddress(context.Context, cosmostypes.AccAddress) cosmostypes.AccountI
	NextAccountNumber(context.Context) uint64

	// Setters
	SetAccount(context.Context, cosmostypes.AccountI)
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	// Setters
	MintCoins(ctx context.Context, moduleName string, amt cosmostypes.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt cosmostypes.Coins) error

	// Getters
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr cosmostypes.AccAddress, amt cosmostypes.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt cosmostypes.Coins) error
	Balance(context.Context, *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error)
}

type ApplicationKeeper interface {
	// Getters
	GetApplication(ctx context.Context, appAddr string) (app apptypes.Application, found bool)
	GetAllApplications(ctx context.Context) []apptypes.Application
	GetParams(ctx context.Context) (params apptypes.Params)

	// Setters
	SetApplication(ctx context.Context, app apptypes.Application)
	UnbondApplication(ctx context.Context, app *apptypes.Application) error
	EndBlockerUnbondApplications(ctx context.Context) error
}

type ProofKeeper interface {
	// Getters
	GetAllClaims(ctx context.Context) []prooftypes.Claim
	GetProof(ctx context.Context, sessionId, supplierOperatorAddr string) (proof prooftypes.Proof, isProofFound bool)
	GetSessionEndHeightClaimsIterator(ctx context.Context, sessionEndHeight int64) sharedtypes.RecordIterator[prooftypes.Claim]
	ProofRequirementForClaim(ctx context.Context, claim *prooftypes.Claim) (prooftypes.ProofRequirementReason, error)
	GetAllProofs(ctx context.Context) []prooftypes.Proof
	GetParams(ctx context.Context) prooftypes.Params

	// Setters
	RemoveClaim(ctx context.Context, sessionId, supplierOperatorAddr string)
	RemoveProof(ctx context.Context, sessionId, supplierOperatorAddr string)
	UpsertClaim(ctx context.Context, claim prooftypes.Claim)
	UpsertProof(ctx context.Context, claim prooftypes.Proof)
	SetParams(ctx context.Context, params prooftypes.Params) error

	// Only used for testing & simulation
	ValidateSubmittedProofs(ctx cosmostypes.Context) (numValidProofs, numInvalidProofs uint64, err error)
}

type SharedKeeper interface {
	// Getters
	GetParams(ctx context.Context) sharedtypes.Params
	GetSessionEndHeight(ctx context.Context, queryHeight int64) int64
	GetProofWindowCloseHeight(ctx context.Context, queryHeight int64) int64

	// Setters
	SetParams(ctx context.Context, params sharedtypes.Params) error
}

type SessionKeeper interface {
	// Getters
	GetSession(context.Context, *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error)
	GetBlockHash(ctx context.Context, height int64) []byte
	GetParams(ctx context.Context) sessiontypes.Params

	// Setters
	StoreBlockHash(ctx context.Context)
}

type SupplierKeeper interface {
	// Getters
	GetParams(ctx context.Context) suppliertypes.Params
	GetSupplier(ctx context.Context, supplierOperatorAddr string) (supplier sharedtypes.Supplier, found bool)
	GetDehydratedSupplier(ctx context.Context, supplierOperatorAddr string) (supplier sharedtypes.Supplier, found bool)
	GetSupplierActiveServiceConfig(ctx context.Context, supplier *sharedtypes.Supplier, serviceId string) (activeServiceConfigs []*sharedtypes.SupplierServiceConfig)

	// Setters
	SetAndIndexDehydratedSupplier(ctx context.Context, supplier sharedtypes.Supplier)
	SetDehydratedSupplier(ctx context.Context, supplier sharedtypes.Supplier)
}

type ServiceKeeper interface {
	// Getters
	GetService(ctx context.Context, serviceID string) (sharedtypes.Service, bool)
	GetRelayMiningDifficulty(ctx context.Context, serviceID string) (servicetypes.RelayMiningDifficulty, bool)
	GetParams(ctx context.Context) servicetypes.Params

	// Setters
	UpdateRelayMiningDifficulty(ctx context.Context, relaysPerServiceMap map[string]uint64) (map[string]servicetypes.RelayMiningDifficulty, error)
	SetService(ctx context.Context, service sharedtypes.Service)
	SetParams(ctx context.Context, params servicetypes.Params) error
}

type MigrationKeeper interface {
	// Setters
	ImportFromMorseAccountState(ctx context.Context, morseAccountState *migrationtypes.MorseAccountState)

	// Getters
	GetMorseClaimableAccount(ctx context.Context, morseHexAddress string) (morseAccount migrationtypes.MorseClaimableAccount, isFound bool)
	GetAllMorseClaimableAccounts(ctx context.Context) (morseAccounts []migrationtypes.MorseClaimableAccount)
}

// StakingKeeper defines the expected interface for the Staking module.
type StakingKeeper interface {
	// GetValidatorByConsAddr gets a validator by consensus address
	GetValidatorByConsAddr(ctx context.Context, consAddr cosmostypes.ConsAddress) (stakingtypes.Validator, error)
	// Validator gets a validator by operator address
	Validator(ctx context.Context, addr cosmostypes.ValAddress) (stakingtypes.ValidatorI, error)
	// GetDelegatorDelegations gets all delegations for a delegator
	GetDelegatorDelegations(ctx context.Context, delegator cosmostypes.AccAddress, maxRetrieve uint16) ([]stakingtypes.Delegation, error)
	// GetValidatorDelegations gets all delegations to a validator
	GetValidatorDelegations(ctx context.Context, valAddr cosmostypes.ValAddress) ([]stakingtypes.Delegation, error)
	// GetBondedValidatorsByPower gets all bonded validators sorted by voting power
	GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error)
	// SetValidator sets the main record holding validator details
	SetValidator(ctx context.Context, validator stakingtypes.Validator) error
	// SetValidatorByConsAddr sets a validator by consensus address
	SetValidatorByConsAddr(ctx context.Context, validator stakingtypes.Validator) error
}
