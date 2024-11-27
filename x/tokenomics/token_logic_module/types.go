package token_logic_module

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type TokenLogicModuleId int

const (
	// UnspecifiedTLM is the default value for TokenLogicModuleId, it is used as a field
	// type for objects which need to distinguish whether a TLM has handled it or not.
	UnspecifiedTLM TokenLogicModuleId = iota

	// TLMRelayBurnEqualsMint is the token logic module that burns the application's
	// stake balance based on the amount of work done by the supplier.
	// The same amount of tokens is minted and added to the supplier account balance.
	// When the network achieves maturity in the far future, this is theoretically
	// the only TLM that will be necessary.
	TLMRelayBurnEqualsMint

	// TLMGlobalMint is the token logic module that mints new tokens based on the
	// global governance parameters in order to reward the participants providing
	// services while keeping inflation in check.
	TLMGlobalMint

	// TLMGlobalMintReimbursementRequest is the token logic module that complements
	// TLMGlobalMint to enable permissionless demand.
	// In order to prevent self-dealing attacks, applications will be overcharged by
	// the amount equal to global inflation, those funds will be sent to the DAO/PNF,
	// and an event will be emitted to track and send reimbursements; managed offchain by PNF.
	// TODO_POST_MAINNET: Introduce proper tokenomics based on the research done by @rawthil and @shane.
	TLMGlobalMintReimbursementRequest
)

var tokenLogicModuleStrings = [...]string{
	"UnspecifiedTLM",
	"TLMRelayBurnEqualsMint",
	"TLMGlobalMint",
	"TLMGlobalMintReimbursementRequest",
}

func (tlm TokenLogicModuleId) String() string {
	return tokenLogicModuleStrings[tlm]
}

func (tlm TokenLogicModuleId) EnumIndex() int {
	return int(tlm)
}

// TokenLogicModule is an interface that all token logic modules are expected to implement.
// IMPORTANT_SIDE_EFFECTS: Please note that TLMs may update the application and supplier objects,
// which is why they are passed in as pointers. NOTE: TLMs CANNOT persist any state changes.
// Persistence of updated application and supplier to the keeper is currently done by the
// tokenomics keeper in `ProcessTokenLogicModules()`. This design and separation of concerns
// may change in the future.
// DEV_NOTE: As of writing this, this is only in anticipation of potentially unstaking
// actors if their stake falls below a certain threshold.
type TokenLogicModule interface {
	GetId() TokenLogicModuleId
	// Process executes the token logic modules business logic given the input/output
	// parameters encapsulated by the TLMContext.
	// IT DOES NOT modify network state directly.
	Process(context.Context, cosmoslog.Logger, TLMContext) error
}

// TLMContext holds all inputs and outputs necessary for token logic module processing,
// allowing TLMs to remain isolated from the tokenomics keeper and each other while still
// permitting shared memory access (prior to an atomic state transition).
type TLMContext struct {
	TokenomicsParams      tokenomicstypes.Params
	SettlementCoin        cosmostypes.Coin // This is the "actualSettlementCoin" rather than just the "claimCoin" because of how settlement functions; see ensureClaimAmountLimits for details.
	SessionHeader         *sessiontypes.SessionHeader
	Result                *tokenomicstypes.ClaimSettlementResult
	Service               *sharedtypes.Service
	Application           *apptypes.Application
	Supplier              *sharedtypes.Supplier
	RelayMiningDifficulty *servicetypes.RelayMiningDifficulty
}

// NewDefaultTokenLogicModules returns the default token logic module processors:
// - TLMRelayBurnEqualsMint
// - TLMGlobalMint
func NewDefaultTokenLogicModules() []TokenLogicModule {
	return []TokenLogicModule{
		NewRelayBurnEqualsMintTLM(),
		NewGlobalMintTLM(),
		NewGlobalMintReimbursementRequestTLM(),
	}
}
