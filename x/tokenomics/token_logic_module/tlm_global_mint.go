package token_logic_module

import (
	"context"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
	"github.com/pokt-network/poktroll/telemetry"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ TokenLogicModule = (*tlmGlobalMint)(nil)

type tlmGlobalMint struct {
	ctx    context.Context
	logger cosmoslog.Logger
	tlmCtx *TLMContext
}

// NewGlobalMintTLM creates a new instance of the GlobalMint TLM.
func NewGlobalMintTLM() TokenLogicModule {
	return &tlmGlobalMint{}
}

func (tlmgm *tlmGlobalMint) GetId() TokenLogicModuleId {
	return TLMGlobalMint
}

// Process processes the business logic for the GlobalMint TLM.
//
// The GlobalMint TLM is responsible for minting new tokens based on the global
// inflation rate and distributing them to various network participants.
// It enables:
//  1. Sustainable network growth through controlled inflation
//  2. Incentive alignment for all network participants
//  3. Decentralized reward distribution
func (tlmgm *tlmGlobalMint) Process(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	tlmgm.ctx = ctx
	tlmgm.logger = logger
	tlmgm.tlmCtx = &tlmCtx

	logger = logger.With(
		"tlm", "TLMGlobalMint",
		"method", "Process",
		"session_id", tlmCtx.Result.GetSessionId(),
	)

	// Mint new tokens based on global inflation
	newMintCoin, err := tlmgm.processInflationMint()
	if err != nil {
		logger.Error(fmt.Sprintf("error processing inflation mint: %v", err))
		return err
	}
	if newMintCoin.IsZero() {
		logger.Debug("newMintCoin is zero. Skipping Global Mint TLM.")
		return nil
	}

	// Distribute minted tokens according to configured percentages
	if err := tlmgm.processMintDistribution(newMintCoin); err != nil {
		logger.Error(fmt.Sprintf("error processing mint distribution: %v", err))
		return err
	}

	return nil
}

// processInflationMint calculates and mints new tokens based on the global inflation rate.
// The amount minted is proportional to the settlement amount and the configured inflation percentage.
func (tlmgm *tlmGlobalMint) processInflationMint() (cosmostypes.Coin, error) {
	// === PARAMETER EXTRACTION ===

	// Retrieve the global inflation per claim
	globalInflationPerClaim := tlmgm.tlmCtx.TokenomicsParams.GetGlobalInflationPerClaim()
	if globalInflationPerClaim == 0 {
		tlmgm.logger.Warn("global inflation is set to zero. Skipping Global Mint TLM.")
		return cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0), nil
	}

	// Convert to rat for safe numeric operators
	globalInflationPerClaimRat, err := encoding.Float64ToRat(globalInflationPerClaim)
	if err != nil {
		tlmgm.logger.Error(fmt.Sprintf("error converting global inflation per claim due to: %v", err))
		return cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0), err
	}

	// === MINT CALCULATION ===

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin := CalculateGlobalPerClaimMintInflationFromSettlementAmount(tlmgm.tlmCtx.SettlementCoin, globalInflationPerClaimRat)
	if newMintCoin.IsZero() {
		return cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0), tokenomicstypes.ErrTokenomicsCoinIsZero.Wrapf("newMintCoin cannot be zero, TLMContext: %+v", tlmgm.tlmCtx)
	}

	// === MINT OPERATION ===

	// Mint new uPOKT to the tokenomics module account from which the rewards will be distributed.
	tlmgm.tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION,
		DestinationModule: tokenomicstypes.ModuleName,
		Coin:              newMintCoin,
	})
	telemetry.MintedTokensFromModule(tokenomicstypes.ModuleName, float32(tlmgm.tlmCtx.SettlementCoin.Amount.Int64()))
	tlmgm.logger.Info(fmt.Sprintf("operation queued: mint (%s) to the tokenomics module account", newMintCoin))

	return newMintCoin, nil
}

// processMintDistribution handles the distribution of newly minted tokens to network participants.
// This function distributes the minted amount according to mint allocation percentages
// configured in the tokenomics parameters. The distribution includes rewards for suppliers,
// applications, service source owners, block proposers, and the DAO.
func (tlmgm *tlmGlobalMint) processMintDistribution(newMintCoin cosmostypes.Coin) error {
	tlmgm.logger.Info("Distributing newly minted tokens according to mint allocation percentages")

	// === PARAMETER EXTRACTION ===

	// Get the mint allocation percentages from tokenomics parameters
	mintAllocationPercentages := tlmgm.tlmCtx.TokenomicsParams.GetMintAllocationPercentages()

	// === ALLOCATION CALCULATIONS ===
	// Calculate how much each participant gets from the newly minted amount

	// Calculate supplier allocation
	supplierMintAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.Supplier)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting supplier mint allocation percentage: %v", err)
	}
	supplierCoinsToShareAmt := calculateAllocationAmount(newMintCoin.Amount, supplierMintAllocationRat)

	// Calculate application allocation
	appMintAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.Application)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting application mint allocation percentage: %v", err)
	}
	appAmount := calculateAllocationAmount(newMintCoin.Amount, appMintAllocationRat)

	// Calculate source owner allocation
	sourceOwnerMintAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.SourceOwner)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting source owner mint allocation percentage: %v", err)
	}
	sourceOwnerAmount := calculateAllocationAmount(newMintCoin.Amount, sourceOwnerMintAllocationRat)

	// Calculate proposer allocation
	proposerMintAllocationRat, err := encoding.Float64ToRat(mintAllocationPercentages.Proposer)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting proposer mint allocation percentage: %v", err)
	}
	proposerAmount := calculateAllocationAmount(newMintCoin.Amount, proposerMintAllocationRat)

	// === REWARD DISTRIBUTION ===
	// Distribute newly minted tokens to each participant according to allocation percentages

	// Distribute to supplier and their shareholders
	if !supplierCoinsToShareAmt.IsZero() {
		supplierCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, supplierCoinsToShareAmt)

		// Transfer from tokenomics module to supplier module
		tlmgm.tlmCtx.Result.AppendModToModTransfer(tokenomicstypes.ModToModTransfer{
			OpReason:        tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_MODULE_TRANSFER,
			SenderModule:    tokenomicstypes.ModuleName,
			RecipientModule: suppliertypes.ModuleName,
			Coin:            supplierCoin,
		})

		// Distribute to supplier's shareholders based on revenue share percentage
		if err := distributeSupplierRewardsToShareholders(
			tlmgm.logger,
			tlmgm.tlmCtx.Result,
			tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
			tlmgm.tlmCtx.Supplier,
			tlmgm.tlmCtx.Service.Id,
			supplierCoinsToShareAmt,
		); err != nil {
			return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
				"queueing operation: distributing rewards to supplier with operator address %s shareholders: %v",
				tlmgm.tlmCtx.Supplier.OperatorAddress,
				err,
			)
		}
		tlmgm.logger.Info(fmt.Sprintf("operation queued: distribute (%v) to supplier shareholders", supplierCoin))
	}

	// Distribute to application
	if !appAmount.IsZero() {
		appCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, appAmount)
		tlmgm.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_APPLICATION_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: tlmgm.tlmCtx.Application.GetAddress(),
			Coin:             appCoin,
		})
		tlmgm.logger.Info(fmt.Sprintf("operation queued: send (%v) to application %s", appCoin, tlmgm.tlmCtx.Application.GetAddress()))
	}

	// Distribute to service source owner
	if !sourceOwnerAmount.IsZero() {
		sourceOwnerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, sourceOwnerAmount)
		tlmgm.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: tlmgm.tlmCtx.Service.OwnerAddress,
			Coin:             sourceOwnerCoin,
		})
		tlmgm.logger.Info(fmt.Sprintf("operation queued: send (%v) to source owner %s", sourceOwnerCoin, tlmgm.tlmCtx.Service.OwnerAddress))
	}

	// Distribute to all validators based on stake weight
	if !proposerAmount.IsZero() {
		proposerCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, proposerAmount)
		// Distribute to all validators and their delegators proportionally based on stake weight
		if err := distributeValidatorRewards(
			tlmgm.ctx,
			tlmgm.logger,
			tlmgm.tlmCtx.Result,
			tlmgm.tlmCtx.StakingKeeper,
			proposerCoin,
			tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		); err != nil {
			tlmgm.logger.Error(fmt.Sprintf("error distributing validator rewards: %v", err))
			return err
		}
		tlmgm.logger.Info(fmt.Sprintf("operation queued: distribute %s to all validators by stake weight", proposerCoin))
	}

	// Distribute to DAO
	// DAO gets the remainder to ensure all minted tokens are distributed
	daoAmount := newMintCoin.Amount.Sub(supplierCoinsToShareAmt).Sub(appAmount).Sub(sourceOwnerAmount).Sub(proposerAmount)
	if !daoAmount.IsZero() {
		daoCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, daoAmount)
		daoRewardAddress := tlmgm.tlmCtx.TokenomicsParams.GetDaoRewardAddress()
		tlmgm.tlmCtx.Result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: daoRewardAddress,
			Coin:             daoCoin,
		})
		tlmgm.logger.Info(fmt.Sprintf("operation queued: send (%v) to DAO %s", daoCoin, daoRewardAddress))
	}

	// === VALIDATION ===

	// Verify all minted coins are distributed
	totalDistributed := supplierCoinsToShareAmt.Add(appAmount).Add(sourceOwnerAmount).Add(proposerAmount).Add(daoAmount)
	if !totalDistributed.Equal(newMintCoin.Amount) {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"total distributed amount (%s) does not equal minted amount (%s)",
			totalDistributed, newMintCoin.Amount,
		)
	}
	tlmgm.logger.Info(fmt.Sprintf("operation queued: distributed (%v) total minted coins to all participants", totalDistributed))

	return nil
}

// CalculateGlobalPerClaimMintInflationFromSettlementAmount calculates the amount
// of uPOKT to mint based on the global per claim inflation rate as a function of
// the settlement amount for a particular claim(s) or session(s).
func CalculateGlobalPerClaimMintInflationFromSettlementAmount(
	settlementCoin cosmostypes.Coin,
	globalInflationPerClaimRat *big.Rat,
) cosmostypes.Coin {
	// Determine how much new uPOKT to mint based on global per claim inflation.
	settlementAmtRat := new(big.Rat).SetInt(settlementCoin.Amount.BigInt())
	newMintAmtRat := new(big.Rat).Mul(settlementAmtRat, globalInflationPerClaimRat)
	// Always ceil the new mint amount.
	// DEV_NOTE: Since settlementCoin is never zero and the mint amount is ceiled,
	// mintAmtCoin will always be greater than zero.
	newMintRem := new(big.Int)
	newMintAmt, newMintRem := new(big.Int).QuoRem(newMintAmtRat.Num(), newMintAmtRat.Denom(), newMintRem)
	// If there is a remainder, add one to the mint amount to ceil the value.
	if newMintRem.Cmp(big.NewInt(0)) > 0 {
		newMintAmt.Add(newMintAmt, big.NewInt(1))
	}
	mintAmtCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewIntFromBigInt(newMintAmt))
	return mintAmtCoin
}

// calculateAllocationAmount does big.Rat arithmetic to determine the absolute
// amount from amountInt based on the allocation percentage provided.
func calculateAllocationAmount(
	amountInt math.Int,
	allocationPercentageRat *big.Rat,
) math.Int {
	amountRat := new(big.Rat).SetInt(amountInt.BigInt())

	allocationRat := new(big.Rat).Mul(amountRat, allocationPercentageRat)
	allocationAmtInt := new(big.Int).Quo(allocationRat.Num(), allocationRat.Denom())

	return math.NewIntFromBigInt(allocationAmtInt)
}
