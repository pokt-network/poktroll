package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"

	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
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
	tlmCtx TLMContext,
) error {
	logger = logger.With(
		"method", "TokenLogicModuleRelayBurnEqualsMint",
		"session_id", tlmCtx.Result.GetSessionId(),
	)

	// DEV_NOTE: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the application stake to the supplier balance in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint new uPOKT to the supplier module account.
	// These funds will be transferred to the supplier's shareholders below.
	// For reference, see operate/configs/supplier_staking_config.md.
	tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT,
		DestinationModule: suppliertypes.ModuleName,
		Coin:              tlmCtx.SettlementCoin,
	})

	logger.Info(fmt.Sprintf("operation queued: mint (%v) coins in the supplier module", tlmCtx.SettlementCoin))

	// Update telemetry information
	if tlmCtx.SettlementCoin.Amount.IsInt64() {
		defer telemetry.MintedTokensFromModule(suppliertypes.ModuleName, float32(tlmCtx.SettlementCoin.Amount.Int64()))
	}

	// Distribute the rewards to the supplier's shareholders based on the rev share percentage.
	if err := distributeSupplierRewardsToShareHolders(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
		tlmCtx.Supplier,
		tlmCtx.Service.Id,
		tlmCtx.SettlementCoin.Amount.Uint64(),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
			"queueing operation: distributing rewards to supplier with operator address %s shareholders: %v",
			tlmCtx.Supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("operation queued: send (%v) from the supplier module to the supplier account with address %q", tlmCtx.SettlementCoin, tlmCtx.Supplier.OperatorAddress))

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	tlmCtx.Result.AppendBurn(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN,
		DestinationModule: apptypes.ModuleName,
		Coin:              tlmCtx.SettlementCoin,
	})
	logger.Info(fmt.Sprintf("operation queued: burn (%v) from the application module account", tlmCtx.SettlementCoin))

	// Update telemetry information
	if tlmCtx.SettlementCoin.Amount.IsInt64() {
		defer telemetry.BurnedTokensFromModule(suppliertypes.ModuleName, float32(tlmCtx.SettlementCoin.Amount.Int64()))
	}

	// Update the application's on-chain stake
	newAppStake, err := tlmCtx.Application.Stake.SafeSub(tlmCtx.SettlementCoin)
	// DEV_NOTE: This should never occur because:
	//   1. Application overservicing SHOULD be mitigated by the protocol.
	//   2. tokenomicskeeper.Keeper#ensureClaimAmountLimits deducts the
	//      overserviced amount from the claimable amount ("free work").
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", tlmCtx.Application.Address, newAppStake)
	}
	tlmCtx.Application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("operation scheduled: update application %q stake to %v", tlmCtx.Application.Address, newAppStake))

	return nil
}
