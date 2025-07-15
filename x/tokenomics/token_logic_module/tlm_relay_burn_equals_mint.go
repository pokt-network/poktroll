package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
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
//
// DEV_NOTE: Mint & burn involves minting & burning to module accounts as interim steps.
// This enables:
//  1. Second order economic effects with more optionality.
//  2. More complex tokenomics distributions in the future.
//  3. Centralized tracking of token transfers
func (tlmbem *tlmRelayBurnEqualsMint) Process(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	tlmbem.ctx = ctx
	tlmbem.logger = logger
	tlmbem.tlmCtx = &tlmCtx

	logger = logger.With(
		"tlm", "TLMRelayBurnEqualsMint",
		"method", "Process",
		"session_id", tlmCtx.Result.GetSessionId(),
	)

	// Burn the corresponding tokens from the application's stake.
	if err := tlmbem.processApplicationBurn(); err != nil {
		logger.Error(fmt.Sprintf("error processing application burn: %v", err))
		return err
	}

	// Mint new uPOKT to the tokenomics module for distribution
	if err := tlmbem.processTokenomicsMint(); err != nil {
		logger.Error(fmt.Sprintf("error processing tokenomics mint: %v", err))
		return err
	}

	// Distribute the settlement amount
	if err := tlmbem.processRewardDistribution(); err != nil {
		logger.Error(fmt.Sprintf("error processing reward distribution: %v", err))
		return err
	}

	return nil
}

// processApplicationBurn reduces the application's stake (tokens in escrow) to pay for the services consumed.
// It includes (but not limited to) the following steps:
// 1. Queue a burn operation on the application module's funds
// 2. Reduce the application's stake (does not include a flush to disk)
func (tlmbem *tlmRelayBurnEqualsMint) processApplicationBurn() error {
	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	tlmbem.tlmCtx.Result.AppendBurn(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN,
		DestinationModule: apptypes.ModuleName,
		Coin:              tlmbem.tlmCtx.SettlementCoin,
	})
	tlmbem.logger.Info(fmt.Sprintf("operation queued: burn (%v) from the application module account", tlmbem.tlmCtx.SettlementCoin))

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

// processTokenomicsMint mints new tokens to the tokenomics module.
// It is equivalent to the amount being burnt from the application stake.
// These funds will be distributed to the supplier and other actors.
func (tlmbem *tlmRelayBurnEqualsMint) processTokenomicsMint() error {
	// Mint the total settlement amount to the tokenomics module for distribution
	tlmbem.tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT,
		DestinationModule: tokenomicstypes.ModuleName,
		Coin:              tlmbem.tlmCtx.SettlementCoin,
	})
	tlmbem.logger.Info(fmt.Sprintf("operation queued: mint (%v) coins to tokenomics module for distributed settlement", tlmbem.tlmCtx.SettlementCoin))

	return nil
}

// processRewardDistribution handles the mint=burn reward distribution.
//
// This function distributes the settlement amount according to mint=burn claim
// distribution percentages instead of giving the full amount to the supplier.
//
// The total amount minted equals the settlement amount, but it's distributed among
// supplier, DAO, proposer, source owner, and application based on the configured percentages.
func (tlmbem *tlmRelayBurnEqualsMint) processRewardDistribution() error {
	tlmbem.logger.Info("Mint=burn TLM: distributing settlement amount according to mint equals burn claim distribution percentages")

	// === PARAMETER EXTRACTION ===

	// Get the mint equals burn claim distribution from tokenomics parameters
	mintEqualsBurnClaimDistribution := tlmbem.tlmCtx.TokenomicsParams.GetMintEqualsBurnClaimDistribution()
	settlementAmount := tlmbem.tlmCtx.SettlementCoin.Amount

	// === ALLOCATION CALCULATIONS ===
	// Calculate how much each participant gets from the settlement amount

	// Calculate supplier allocation
	supplierAllocationRat, err := encoding.Float64ToRat(mintEqualsBurnClaimDistribution.Supplier)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting supplier allocation percentage: %v", err)
	}
	supplierAmount := calculateAllocationAmount(settlementAmount, supplierAllocationRat)

	// Calculate proposer allocation
	proposerAllocationRat, err := encoding.Float64ToRat(mintEqualsBurnClaimDistribution.Proposer)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting proposer allocation percentage: %v", err)
	}
	proposerAmount := calculateAllocationAmount(settlementAmount, proposerAllocationRat)

	// Calculate source owner allocation
	sourceOwnerAllocationRat, err := encoding.Float64ToRat(mintEqualsBurnClaimDistribution.SourceOwner)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting source owner allocation percentage: %v", err)
	}
	sourceOwnerAmount := calculateAllocationAmount(settlementAmount, sourceOwnerAllocationRat)

	// Calculate application allocation
	applicationAllocationRat, err := encoding.Float64ToRat(mintEqualsBurnClaimDistribution.Application)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting application allocation percentage: %v", err)
	}
	applicationAmount := calculateAllocationAmount(settlementAmount, applicationAllocationRat)

	// DAO gets the remainder to ensure all settlement tokens are distributed
	daoAmount := settlementAmount.Sub(supplierAmount).Sub(proposerAmount).Sub(sourceOwnerAmount).Sub(applicationAmount)

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
