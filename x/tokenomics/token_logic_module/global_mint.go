package token_logic_module

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ TokenLogicModule = (*tlmGlobalMint)(nil)

type tlmGlobalMint struct{}

// NewGlobalMintTLM creates a new instance of the GlobalMint TLM.
func NewGlobalMintTLM() TokenLogicModule {
	return &tlmGlobalMint{}
}

func (tlm tlmGlobalMint) GetId() TokenLogicModuleId {
	return TLMGlobalMint
}

// Process processes the business logic for the GlobalMint TLM.
func (tlm tlmGlobalMint) Process(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	logger = logger.With(
		"method", "tlmGlobalMint#Process",
		"session_id", tlmCtx.Result.GetSessionId(),
	)

	globalInflationPerClaim := tlmCtx.TokenomicsParams.GetGlobalInflationPerClaim()
	globalInflationPerClaimRat, err := Float64ToRat(globalInflationPerClaim)
	if err != nil {
		logger.Error(fmt.Sprintf("error converting global inflation per claim due to: %v", err))
		return err
	}

	if globalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin := CalculateGlobalPerClaimMintInflationFromSettlementAmount(tlmCtx.SettlementCoin, globalInflationPerClaimRat)
	if newMintCoin.IsZero() {
		return tokenomicstypes.ErrTokenomicsCoinIsZero.Wrapf("newMintCoin cannot be zero, TLMContext: %+v", tlmCtx)
	}

	// Mint new uPOKT to the tokenomics module account
	tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION,
		DestinationModule: tokenomicstypes.ModuleName,
		Coin:              newMintCoin,
	})
	logger.Info(fmt.Sprintf("operation queued: mint (%s) to the tokenomics module account", newMintCoin))

	mintAllocationPercentages := tlmCtx.TokenomicsParams.GetMintAllocationPercentages()

	// Send a portion of the rewards to the application
	appMintAllocationRat, err := Float64ToRat(mintAllocationPercentages.Application)
	if err != nil {
		logger.Error(fmt.Sprintf("error converting application mint allocation percentage due to: %v", err))
		return err
	}

	appCoin := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_APPLICATION_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		tlmCtx.Application.GetAddress(),
		appMintAllocationRat,
		newMintCoin,
	)
	logMsg := fmt.Sprintf("operation queued: send (%v) newley minted coins from the tokenomics module to the application with address %q", appCoin, tlmCtx.Application.GetAddress())
	logRewardOperation(logger, logMsg, &appCoin)

	// Send a portion of the rewards to the supplier shareholders.
	supplierMintAllocationRat, err := Float64ToRat(mintAllocationPercentages.Supplier)
	if err != nil {
		logger.Error(fmt.Sprintf("error converting supplier mint allocation percentage due to: %v", err))
		return err
	}

	supplierCoinsToShareAmt := calculateAllocationAmount(newMintCoin.Amount, supplierMintAllocationRat)
	supplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, supplierCoinsToShareAmt)
	// Send funds from the tokenomics module to the supplier module account
	tlmCtx.Result.AppendModToModTransfer(tokenomicstypes.ModToModTransfer{
		OpReason:        tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_MODULE_TRANSFER,
		SenderModule:    tokenomicstypes.ModuleName,
		RecipientModule: suppliertypes.ModuleName,
		Coin:            supplierCoin,
	})
	// Distribute the rewards from within the supplier's module account.
	if err = distributeSupplierRewardsToShareHolders(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
		tlmCtx.Supplier,
		tlmCtx.Service.Id,
		supplierCoinsToShareAmt,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
			"queueing operation: distributing rewards to supplier with operator address %s shareholders: %v",
			tlmCtx.Supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("operation queued: send (%v) newley minted coins from the tokenomics module to the supplier with address %q", supplierCoin, tlmCtx.Supplier.OperatorAddress))

	// Send a portion of the rewards to the source owner
	sourceOwnerMintAllocationRat, err := Float64ToRat(mintAllocationPercentages.SourceOwner)
	if err != nil {
		logger.Error(fmt.Sprintf("error converting source owner mint allocation percentage due to: %v", err))
		return err
	}

	serviceCoin := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		tlmCtx.Service.OwnerAddress,
		sourceOwnerMintAllocationRat,
		newMintCoin,
	)
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the source owner with address %q", serviceCoin, tlmCtx.Service.OwnerAddress)
	logRewardOperation(logger, logMsg, &serviceCoin)

	// Send a portion of the rewards to the block proposer
	proposerAddr := cosmostypes.AccAddress(cosmostypes.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerMintAllocationRat, err := Float64ToRat(mintAllocationPercentages.Proposer)
	if err != nil {
		logger.Error(fmt.Sprintf("error converting proposer mint allocation percentage due to: %v", err))
		return err
	}
	proposerCoin := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_PROPOSER_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		proposerAddr,
		proposerMintAllocationRat,
		newMintCoin,
	)
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the proposer with address %q", proposerCoin, proposerAddr)
	logRewardOperation(logger, logMsg, &proposerCoin)

	// Send a portion of the rewards to the DAO
	daoRewardAddress := tlmCtx.TokenomicsParams.GetDaoRewardAddress()
	daoCoin := sendRewardsToDAOAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		daoRewardAddress,
		newMintCoin,
		appCoin, supplierCoin, proposerCoin, serviceCoin,
	)
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the DAO with address %q", daoCoin, daoRewardAddress)
	logRewardOperation(logger, logMsg, &daoCoin)

	// Check and log the total amount of coins distributed
	if err := ensureMintedCoinsAreDistributed(logger, appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin, newMintCoin); err != nil {
		return err
	}

	return nil
}

// ensureMintedCoinsAreDistributed checks whether the total amount of minted coins
// is correctly distributed to the application, supplier, DAO, source owner, and proposer.
// If the total distributed coins do not equal the amount of newly minted coins, an error
// is returned. If the discrepancy is within the allowable tolerance, a warning is logged
// and nil is returned.
func ensureMintedCoinsAreDistributed(
	logger cosmoslog.Logger,
	appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin, newMintCoin cosmostypes.Coin,
) error {
	// Compute the difference between the total distributed coins and the amount of newly minted coins
	totalMintDistributedCoin := appCoin.Add(supplierCoin).Add(daoCoin).Add(serviceCoin).Add(proposerCoin)

	coinDifference := totalMintDistributedCoin.Sub(newMintCoin)
	if !coinDifference.IsZero() {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"the total distributed coins (%s) do not equal the amount of newly minted coins (%s)"+
				"appCoin: %s, supplierCoin: %s, daoCoin: %s, serviceCoin: %s, proposerCoin: %s",
			totalMintDistributedCoin, newMintCoin, appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin)
	}

	logger.Info(fmt.Sprintf("operation queued: distribute (%v) coins to the application, supplier, DAO, source owner, and proposer", totalMintDistributedCoin))

	return nil
}

// sendRewardsToDAOAccount calculates and sends rewards to the DAO account.
// The DAO reward amount is computed as the difference between the total settlement amount
// and the sum of all other actors' rewards (application, supplier, proposer, and source owner).
// This difference-based approach, rather than using a fixed percentage, ensures that any
// rounding remainders from the integer division of other rewards are captured in the DAO's share.
// The tokenomics.Params validation guarantees that allocation percentages sum to 100%,
// ensuring this calculation's correctness.
func sendRewardsToDAOAccount(
	result *tokenomicstypes.ClaimSettlementResult,
	opReason tokenomicstypes.SettlementOpReason,
	senderModule string,
	recipientAddr string,
	settlementCoin, appRewards, supplierRewards, proposerRewards, sourceOwnerRewards cosmostypes.Coin,
) cosmostypes.Coin {
	coinToDAOAcc := settlementCoin.
		Sub(appRewards).
		Sub(supplierRewards).
		Sub(proposerRewards).
		Sub(sourceOwnerRewards)

	if !coinToDAOAcc.IsZero() {
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         opReason,
			SenderModule:     senderModule,
			RecipientAddress: recipientAddr,
			Coin:             coinToDAOAcc,
		})
	}

	return coinToDAOAcc
}

// sendRewardsToAccount sends (settlementAmtFloat * allocation) tokens from the
// tokenomics module account to the specified address.
func sendRewardsToAccount(
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	opReason tokenomicstypes.SettlementOpReason,
	senderModule string,
	recipientAddr string,
	allocationRat *big.Rat,
	settlementCoin cosmostypes.Coin,
) cosmostypes.Coin {
	logger = logger.With(
		"method", "mintRewardsToAccount",
		"session_id", result.GetSessionId(),
	)

	coinsToAccAmt := calculateAllocationAmount(settlementCoin.Amount, allocationRat)
	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, coinsToAccAmt)

	if coinToAcc.IsZero() {
		return cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	}

	result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
		OpReason:         opReason,
		SenderModule:     senderModule,
		RecipientAddress: recipientAddr,
		Coin:             coinToAcc,
	})

	logger.Info(fmt.Sprintf("operation queued: send (%v) coins from the tokenomics module to the account with address %q", coinToAcc, recipientAddr))

	return coinToAcc
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
	mintAmtCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(newMintAmt))
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

// Float64ToRat converts a float64 to a big.Rat for precise decimal arithmetic.
// TODO_CONSIDERATION: Future versions of CosmosSDK will deprecate float64 values
// with zero copy encoding of scalar values.
// We should consider switching to string representations for tokenomics allocation percentages.
// NB: It is publicly exposed to be used in the tests.
func Float64ToRat(f float64) (*big.Rat, error) {
	// Convert the float64 to a string before big.Rat conversion to avoid floating
	// point precision issues (e.g. bigRat.SetString("0.1") == 1/10 while bigRat.SetFloat64(0.1) == 3602879701896397/36028797018963968)
	allocationPercentageStr := strconv.FormatFloat(f, 'f', -1, 64)
	allocationPercentageRat, ok := new(big.Rat).SetString(allocationPercentageStr)
	if !ok {
		return nil, tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error converting float64 to big.Rat: %f", f)
	}

	return allocationPercentageRat, nil
}
