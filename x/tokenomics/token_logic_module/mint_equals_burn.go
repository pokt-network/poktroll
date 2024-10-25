package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ TokenLogicModule = (*tlmRelayBurnEqualsMint)(nil)

type tlmRelayBurnEqualsMint struct{}

// NewRelayBurnEqualsMintTLM returns a new RelayBurnEqualsMint TLM.
func NewRelayBurnEqualsMintTLM() TokenLogicModule {
	return &tlmRelayBurnEqualsMint{}
}

func (tlm tlmRelayBurnEqualsMint) GetId() TokenLogicModuleId {
	return TLMRelayBurnEqualsMint
}

// Process processes the business logic for the RelayBurnEqualsMint TLM.
func (tlm tlmRelayBurnEqualsMint) Process(
	_ context.Context,
	logger cosmoslog.Logger,
	pendingResult *PendingSettlementResult,
	service *sharedtypes.Service,
	_ *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoin cosmostypes.Coin,
	_ *servicetypes.RelayMiningDifficulty,
) error {
	logger = logger.With("method", "TokenLogicModuleRelayBurnEqualsMint")

	// DEV_NOTE: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the application stake to the supplier balance in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint new uPOKT to the supplier module account.
	// These funds will be transferred to the supplier's shareholders below.
	// For reference, see operate/configs/supplier_staking_config.md.
	pendingResult.AppendMint(MintBurn{
		TLM:               TLMRelayBurnEqualsMint,
		DestinationModule: suppliertypes.ModuleName,
		Coin:              settlementCoin,
	})

	logger.Info(fmt.Sprintf("operation scheduled: mint (%v) coins in the supplier module", settlementCoin))

	// Distribute the rewards to the supplier's shareholders based on the rev share percentage.
	if err := distributeSupplierRewardsToShareHolders(
		logger,
		pendingResult,
		TLMRelayBurnEqualsMint,
		supplier,
		service.Id,
		settlementCoin.Amount.Uint64(),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsModuleMint.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent (%v) from the supplier module to the supplier account with address %q", settlementCoin, supplier.OperatorAddress))

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	pendingResult.AppendBurn(MintBurn{
		TLM:               TLMRelayBurnEqualsMint,
		DestinationModule: apptypes.ModuleName,
		Coin:              settlementCoin,
	})
	logger.Info(fmt.Sprintf("operation scheduled: burn (%v) from the application module account", settlementCoin))

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(settlementCoin)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", application.Address, newAppStake)
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %v", application.Address, newAppStake))

	return nil
}
