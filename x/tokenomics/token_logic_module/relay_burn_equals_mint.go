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

type tlmRelayBurnEqualsMint struct {
	ctx    context.Context
	logger cosmoslog.Logger
	tlmCtx *TLMContext
}

// NewRelayBurnEqualsMintTLM returns a new RelayBurnEqualsMint TLM.
func NewRelayBurnEqualsMintTLM() TokenLogicModule {
	return &tlmRelayBurnEqualsMint{}
}

func (tlmbem *tlmRelayBurnEqualsMint) GetId() TokenLogicModuleId {
	return TLMRelayBurnEqualsMint
}

// Process processes the business logic for the RelayBurnEqualsMint TLM.
func (tlmbem *tlmRelayBurnEqualsMint) Process(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	tlmbem.ctx = ctx
	tlmbem.logger = logger
	tlmbem.tlmCtx = &tlmCtx

	logger = logger.With(
		"method", "TokenLogicModuleRelayBurnEqualsMint",
		"session_id", tlmCtx.Result.GetSessionId(),
	)

	// DEV_NOTE: Mint & burn involves minting & burning to module accounts as interim steps.
	// This enables:
	// 1. Second order economic effects with more optionality.
	// 2. More complex tokenomics distributions in the future.
	// 3. Centralized tracking of token transfers

	if err := tlmbem.processApplicationBurn(); err != nil {
		logger.Error(fmt.Sprintf("error processing application burn: %v", err))
		return err
	}

	// Check if global inflation is disabled
	globalInflationPerClaim := tlmCtx.TokenomicsParams.GetGlobalInflationPerClaim()

	if globalInflationPerClaim == 0 {
		// When global inflation is disabled, distribute the settlement amount
		// according to claim settlement distribution percentages
		if err := tlmbem.processDistributedRewardSettlement(); err != nil {
			return err
		}
	} else {
		// Original behavior: mint full settlement amount to supplier and distribute to shareholders
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
	}

	return nil
}

func (tlmbem *tlmRelayBurnEqualsMint) processApplicationBurn() error {
	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	tlmbem.tlmCtx.Result.AppendBurn(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN,
		DestinationModule: apptypes.ModuleName,
		Coin:              tlmbem.tlmCtx.SettlementCoin,
	})
	tlmbem.logger.Info(fmt.Sprintf("operation queued: burn (%v) from the application module account", tlmbem.tlmCtx.SettlementCoin))

	// Update telemetry information
	if tlmbem.tlmCtx.SettlementCoin.Amount.IsInt64() {
		defer telemetry.BurnedTokensFromModule(apptypes.ModuleName, float32(tlmbem.tlmCtx.SettlementCoin.Amount.Int64()))
	}

	// Update the application's onchain stake
	newAppStake, err := tlmbem.tlmCtx.Application.Stake.SafeSub(tlmbem.tlmCtx.SettlementCoin)
	// DEV_NOTE: This should never occur because:
	//   1. Application over-servicing SHOULD be mitigated by the protocol.
	//   2. tokenomicskeeper.Keeper#ensureClaimAmountLimits deducts the
	//      overserviced amount from the claimable amount ("free work").
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", tlmbem.tlmCtx.Application.Address, newAppStake)
	}
	tlmbem.tlmCtx.Application.Stake = &newAppStake
	tlmbem.logger.Info(fmt.Sprintf("operation scheduled: update application %q stake to %v", tlmbem.tlmCtx.Application.Address, newAppStake))

	return nil
}

// processDistributedRewardSettlement handles the reward distribution logic when global inflation is disabled.
// This function distributes the settlement amount according to claim settlement distribution percentages
// instead of giving the full amount to the supplier. The total amount minted equals the settlement
// amount, but it's distributed among supplier, DAO, proposer, source owner, and application based on the configured percentages.
func (tlmbem *tlmRelayBurnEqualsMint) processDistributedRewardSettlement() error {
	tlmbem.logger.Info("Global inflation is disabled, distributing settlement amount according to claim settlement distribution percentages")

	// === PARAMETER EXTRACTION ===
	// Get the claim settlement distribution from tokenomics parameters
	claimSettlementDistribution := tlmbem.tlmCtx.TokenomicsParams.GetClaimSettlementDistribution()
	settlementAmount := tlmbem.tlmCtx.SettlementCoin.Amount

	// === ALLOCATION CALCULATIONS ===
	// Calculate how much each participant gets from the settlement amount

	// Calculate supplier allocation
	supplierAllocationRat, err := encoding.Float64ToRat(claimSettlementDistribution.Supplier)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting supplier allocation percentage: %v", err)
	}
	supplierAmount := calculateAllocationAmount(settlementAmount, supplierAllocationRat)

	// Calculate proposer allocation
	proposerAllocationRat, err := encoding.Float64ToRat(claimSettlementDistribution.Proposer)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting proposer allocation percentage: %v", err)
	}
	proposerAmount := calculateAllocationAmount(settlementAmount, proposerAllocationRat)

	// Calculate source owner allocation
	sourceOwnerAllocationRat, err := encoding.Float64ToRat(claimSettlementDistribution.SourceOwner)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting source owner allocation percentage: %v", err)
	}
	sourceOwnerAmount := calculateAllocationAmount(settlementAmount, sourceOwnerAllocationRat)

	// Calculate application allocation
	applicationAllocationRat, err := encoding.Float64ToRat(claimSettlementDistribution.Application)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting application allocation percentage: %v", err)
	}
	applicationAmount := calculateAllocationAmount(settlementAmount, applicationAllocationRat)

	// DAO gets the remainder to ensure all settlement tokens are distributed
	daoAmount := settlementAmount.Sub(supplierAmount).Sub(proposerAmount).Sub(sourceOwnerAmount).Sub(applicationAmount)

	// === MINTING OPERATIONS ===
	// Mint the total settlement amount to the tokenomics module for distribution
	tlmbem.tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT,
		DestinationModule: tokenomicstypes.ModuleName,
		Coin:              tlmbem.tlmCtx.SettlementCoin,
	})
	tlmbem.logger.Info(fmt.Sprintf("operation queued: mint (%v) coins to tokenomics module for distributed settlement", tlmbem.tlmCtx.SettlementCoin))

	// Update telemetry information
	if tlmbem.tlmCtx.SettlementCoin.Amount.IsInt64() {
		defer telemetry.MintedTokensFromModule(tokenomicstypes.ModuleName, float32(tlmbem.tlmCtx.SettlementCoin.Amount.Int64()))
	}

	// === REWARD DISTRIBUTION ===
	// Distribute settlement amount to each participant according to allocation percentages

	// Distribute to supplier and their shareholders
	if !supplierAmount.IsZero() {
		supplierCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierAmount)

		// Transfer from tokenomics module to supplier module
		tlmbem.tlmCtx.Result.AppendModToModTransfer(tokenomicstypes.ModToModTransfer{
			OpReason:        tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
			SenderModule:    tokenomicstypes.ModuleName,
			RecipientModule: suppliertypes.ModuleName,
			Coin:            supplierCoin,
		})

		// Distribute to supplier's shareholders based on revenue share percentage
		if err := distributeSupplierRewardsToShareHolders(
			tlmbem.logger,
			tlmbem.tlmCtx.Result,
			tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
			tlmbem.tlmCtx.Supplier,
			tlmbem.tlmCtx.Service.Id,
			supplierAmount,
		); err != nil {
			return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
				"queueing operation: distributing rewards to supplier with operator address %s shareholders: %v",
				tlmbem.tlmCtx.Supplier.OperatorAddress,
				err,
			)
		}
		tlmbem.logger.Info(fmt.Sprintf("operation queued: distribute (%v) to supplier shareholders", supplierCoin))
	}

	// Distribute to block proposer
	if !proposerAmount.IsZero() {
		proposerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, proposerAmount)
		proposerAddr := cosmostypes.AccAddress(cosmostypes.UnwrapSDKContext(tlmbem.ctx).BlockHeader().ProposerAddress).String()
		tlmbem.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_PROPOSER_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: proposerAddr,
			Coin:             proposerCoin,
		})
		tlmbem.logger.Info(fmt.Sprintf("operation queued: send (%v) to proposer %s", proposerCoin, proposerAddr))
	}

	// Distribute to service source owner
	if !sourceOwnerAmount.IsZero() {
		sourceOwnerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, sourceOwnerAmount)
		tlmbem.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: tlmbem.tlmCtx.Service.OwnerAddress,
			Coin:             sourceOwnerCoin,
		})
		tlmbem.logger.Info(fmt.Sprintf("operation queued: send (%v) to source owner %s", sourceOwnerCoin, tlmbem.tlmCtx.Service.OwnerAddress))
	}

	// Distribute to DAO
	if !daoAmount.IsZero() {
		daoCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, daoAmount)
		daoRewardAddress := tlmbem.tlmCtx.TokenomicsParams.GetDaoRewardAddress()
		tlmbem.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: daoRewardAddress,
			Coin:             daoCoin,
		})
		tlmbem.logger.Info(fmt.Sprintf("operation queued: send (%v) to DAO %s", daoCoin, daoRewardAddress))
	}

	// Distribute to application
	if !applicationAmount.IsZero() {
		applicationCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, applicationAmount)
		tlmbem.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: tlmbem.tlmCtx.Application.Address,
			Coin:             applicationCoin,
		})
		tlmbem.logger.Info(fmt.Sprintf("operation queued: send (%v) to application %s", applicationCoin, tlmbem.tlmCtx.Application.Address))
	}

	// === VALIDATION ===
	// Verify all settlement coins are distributed
	totalDistributed := supplierAmount.Add(proposerAmount).Add(sourceOwnerAmount).Add(daoAmount).Add(applicationAmount)
	if !totalDistributed.Equal(settlementAmount) {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"total distributed amount (%s) does not equal settlement amount (%s)",
			totalDistributed, settlementAmount,
		)
	}
	tlmbem.logger.Info(fmt.Sprintf("operation queued: distributed (%v) total settlement coins to all participants", totalDistributed))

	return nil
}
