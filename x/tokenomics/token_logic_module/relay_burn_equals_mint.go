package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
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
	ctx context.Context,
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

	// First, maintain the original burn=mint behavior: mint full settlement amount to supplier
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
		tlmCtx.SettlementCoin.Amount,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
			"queueing operation: distributing rewards to supplier with operator address %s shareholders: %v",
			tlmCtx.Supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("operation queued: send (%v) from the supplier module to the supplier account with address %q", tlmCtx.SettlementCoin, tlmCtx.Supplier.OperatorAddress))

	// Add reward distribution logic for other participants when global inflation is disabled
	// This replaces the global mint TLM functionality when global_inflation_per_claim = 0
	// When global inflation is disabled, we still want to distribute additional rewards to
	// DAO, proposer, and source owner based on the mint allocation percentages, using the
	// settlement amount as the base for calculations.
	globalInflationPerClaim := tlmCtx.TokenomicsParams.GetGlobalInflationPerClaim()
	if globalInflationPerClaim == 0 {
		if err := tlm.processAdditionalRewardDistribution(ctx, logger, tlmCtx); err != nil {
			return err
		}
	}

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
		defer telemetry.BurnedTokensFromModule(apptypes.ModuleName, float32(tlmCtx.SettlementCoin.Amount.Int64()))
	}

	// Update the application's onchain stake
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

// processAdditionalRewardDistribution handles the reward distribution logic when global inflation is disabled.
// This function replaces the global mint TLM functionality by distributing additional rewards to
// DAO, proposer, and source owner based on the mint allocation percentages.
func (tlm tlmRelayBurnEqualsMint) processAdditionalRewardDistribution(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	logger.Info("Global inflation is disabled, applying reward distribution in RelayBurnEqualsMint TLM")

	// === PARAMETER EXTRACTION ===
	// Get the mint allocation percentages from tokenomics parameters
	mintAllocationPercentages := tlmCtx.TokenomicsParams.GetMintAllocationPercentages()
	settlementAmount := tlmCtx.SettlementCoin.Amount

	// === ALLOCATION CALCULATIONS ===
	// Calculate additional rewards for non-supplier participants
	// Note: we skip the supplier allocation since they already received the full settlement amount
	
	// Calculate proposer allocation
	proposerAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.Proposer)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting proposer allocation percentage: %v", err)
	}
	proposerAmount := calculateAllocationAmount(settlementAmount, proposerAllocationRat)

	// Calculate source owner allocation
	sourceOwnerAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.SourceOwner)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting source owner allocation percentage: %v", err)
	}
	sourceOwnerAmount := calculateAllocationAmount(settlementAmount, sourceOwnerAllocationRat)

	// Calculate DAO allocation
	daoAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.Dao)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting DAO allocation percentage: %v", err)
	}
	daoAmount := calculateAllocationAmount(settlementAmount, daoAllocationRat)

	// === MINTING OPERATIONS ===
	// Calculate total additional rewards to mint and mint them to the tokenomics module
	totalAdditionalRewards := proposerAmount.Add(sourceOwnerAmount).Add(daoAmount)
	
	if !totalAdditionalRewards.IsZero() {
		additionalRewardsCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, totalAdditionalRewards)
		
		// Mint additional rewards to the tokenomics module for distribution
		tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
			OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT,
			DestinationModule: tokenomicstypes.ModuleName,
			Coin:              additionalRewardsCoin,
		})
		logger.Info(fmt.Sprintf("operation queued: mint (%v) additional reward coins to the tokenomics module", additionalRewardsCoin))

		// === REWARD DISTRIBUTION ===
		// Distribute additional rewards to each participant
		
		// Distribute to block proposer
		if !proposerAmount.IsZero() {
			proposerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, proposerAmount)
			proposerAddr := cosmostypes.AccAddress(cosmostypes.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
			tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
				OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_PROPOSER_REWARD_DISTRIBUTION,
				SenderModule:     tokenomicstypes.ModuleName,
				RecipientAddress: proposerAddr,
				Coin:             proposerCoin,
			})
			logger.Info(fmt.Sprintf("operation queued: send (%v) to proposer %s", proposerCoin, proposerAddr))
		}

		// Distribute to service source owner
		if !sourceOwnerAmount.IsZero() {
			sourceOwnerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, sourceOwnerAmount)
			tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
				OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
				SenderModule:     tokenomicstypes.ModuleName,
				RecipientAddress: tlmCtx.Service.OwnerAddress,
				Coin:             sourceOwnerCoin,
			})
			logger.Info(fmt.Sprintf("operation queued: send (%v) to source owner %s", sourceOwnerCoin, tlmCtx.Service.OwnerAddress))
		}

		// Distribute to DAO
		if !daoAmount.IsZero() {
			daoCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, daoAmount)
			daoRewardAddress := tlmCtx.TokenomicsParams.GetDaoRewardAddress()
			tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
				OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION,
				SenderModule:     tokenomicstypes.ModuleName,
				RecipientAddress: daoRewardAddress,
				Coin:             daoCoin,
			})
			logger.Info(fmt.Sprintf("operation queued: send (%v) to DAO %s", daoCoin, daoRewardAddress))
		}
	}

	return nil
}
