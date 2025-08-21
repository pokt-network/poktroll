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
	// UnspecifiedTLM
	// - Default value for TokenLogicModuleId
	// - Used as a field type for objects to distinguish if a TLM has handled it or not
	UnspecifiedTLM TokenLogicModuleId = iota

	// TLMRelayBurnEqualsMint
	// - Burns the application's stake balance based on the amount of work done by the supplier
	// - Mints the same amount of tokens and adds them to the supplier's account balance
	// - When the network matures, this is theoretically the only TLM needed
	TLMRelayBurnEqualsMint

	// TLMGlobalMint
	// - Mints new tokens based on global governance parameters
	// - Rewards participants providing services
	// - Keeps inflation in check
	TLMGlobalMint

	// TLMGlobalMintReimbursementRequest
	// - Complements TLMGlobalMint to enable permissionless demand
	// - Prevents self-dealing attacks:
	//   - Applications are overcharged by an amount equal to global inflation
	//   - Overcharged funds are sent to the DAO/PNF
	//   - Event emitted to track and send reimbursements (managed offchain by PNF)
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

// TokenLogicModule interface
// - All token logic modules must implement this interface
//
// IMPORTANT_SIDE_EFFECTS:
// - TLMs may update the application and supplier objects (passed as pointers)
// - TLMs CANNOT persist any state changes
// - Persistence of updated application and supplier is handled by the tokenomics keeper in `ProcessTokenLogicModules()`
// - This design and separation of concerns may change in the future
//
// DEV_NOTE:
// - As of writing, this anticipates potentially unstaking actors if their stake falls below a threshold
type TokenLogicModule interface {
	GetId() TokenLogicModuleId
	// Process executes the token logic modules business logic given the input/output
	// parameters encapsulated by the TLMContext.
	// IT DOES NOT modify network state directly.
	Process(context.Context, cosmoslog.Logger, TLMContext) error
}

// TLMContext is responsible for processing & settling tokenomics for a given session.
// - Holds all inputs and outputs necessary for token logic module processing
// - Allows TLMs to remain isolated from the tokenomics keeper and each other
// - Permits shared memory access (prior to an atomic state transition)
type TLMContext struct {
	TokenomicsParams      tokenomicstypes.Params
	SettlementCoin        cosmostypes.Coin // This is the "actualSettlementCoin" rather than just the "claimCoin" because of how settlement functions; see ensureClaimAmountLimits for details.
	SessionHeader         *sessiontypes.SessionHeader
	Result                *tokenomicstypes.ClaimSettlementResult
	Service               *sharedtypes.Service
	Application           *apptypes.Application
	Supplier              *sharedtypes.Supplier
	RelayMiningDifficulty *servicetypes.RelayMiningDifficulty
	StakingKeeper         tokenomicstypes.StakingKeeper // Used for validator and delegation queries
}

// NewDefaultTokenLogicModules
// - Returns the default token logic module processors:
//   - TLMRelayBurnEqualsMint
//   - TLMGlobalMint
//   - TLMGlobalMintReimbursementRequest
func NewDefaultTokenLogicModules() []TokenLogicModule {
	return []TokenLogicModule{
		NewRelayBurnEqualsMintTLM(),
		NewGlobalMintTLM(),
		NewGlobalMintReimbursementRequestTLM(),
	}
}
