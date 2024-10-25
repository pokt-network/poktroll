package token_logic_module

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type TokenLogicModuleId int

const (
	// TLMRelayBurnEqualsMint is the token logic module that burns the application's
	// stake balance based on the amount of work done by the supplier.
	// The same amount of tokens is minted and added to the supplier account balance.
	// When the network achieves maturity in the far future, this is theoretically
	// the only TLM that will be necessary.
	TLMRelayBurnEqualsMint TokenLogicModuleId = iota

	// TLMGlobalMint is the token logic module that mints new tokens based on the
	// global governance parameters in order to reward the participants providing
	// services while keeping inflation in check.
	TLMGlobalMint
)

var tokenLogicModuleStrings = [...]string{
	"TLMRelayBurnEqualsMint",
	"TLMGlobalMint",
}

func (tlm TokenLogicModuleId) String() string {
	return tokenLogicModuleStrings[tlm]
}

func (tlm TokenLogicModuleId) EnumIndex() int {
	return int(tlm)
}

// TODO_IN_THIS_COMMIT: update after renaming TokenLogicModule to TokenLogicModuleId
// and TokenLogicModuleProcessor to TokenLogicModule.
//
// TokenLogicModuleProcessor is an interface that all token logic modules are
// expected to implement.
// IMPORTANT_SIDE_EFFECTS: Please note that TLMs may update the application and supplier objects,
// which is why they are passed in as pointers. NOTE: TLMs CANNOT persist any state changes.
// Persistence of updated application and supplier to the keeper is currently done by the TLM
// processor in `ProcessTokenLogicModules()`. This design and separation of concerns may change
// in the future.
// DEV_NOTE: As of writing this, this is only in anticipation of potentially unstaking
// actors if their stake falls below a certain threshold.
type TokenLogicModule interface {
	GetId() TokenLogicModuleId
	Process(
		context.Context,
		cosmoslog.Logger,
		*PendingSettlementResult,
		*sharedtypes.Service,
		*sessiontypes.SessionHeader,
		*apptypes.Application,
		*sharedtypes.Supplier,
		cosmostypes.Coin, // This is the "actualSettlementCoin" rather than just the "claimCoin" because of how settlement functions; see ensureClaimAmountLimits for details.
		*servicetypes.RelayMiningDifficulty,
	) error
}

// NewDefaultTokenLogicModules returns the default token logic module processors:
// - TLMRelayBurnEqualsMint
// - TLMGlobalMint
func NewDefaultTokenLogicModules(authorityRewardAddr string) []TokenLogicModule {
	return []TokenLogicModule{
		NewRelayBurnEqualsMintTLM(),
		// TODO_TECHDEBT: Replace authorityRewardAddr with the tokenomics module
		// params once it's refactored as a param.
		NewGlobalMintTLM(authorityRewardAddr),
	}
}
