package token_logic_module

import (
	"context"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// TODO_BETA(@bryanchriswhite): Make all of these governance params
	MintAllocationDAO         = 0.1
	MintAllocationProposer    = 0.05
	MintAllocationSupplier    = 0.7
	MintAllocationSourceOwner = 0.15
	MintAllocationApplication = 0.0

	// MintDistributionAllowableTolerancePercent is the percent difference that is allowable
	// between the number of minted/ tokens in the tokenomics module and what is distributed
	// to pocket network participants.
	// This internal constant SHOULD ONLY be used in TokenLogicModuleGlobalMint.
	// Due to floating point arithmetic, the total amount of minted coins may be slightly
	// larger than what is distributed to pocket network participants
	// TODO_MAINNET: Figure out if we can avoid this tolerance and use fixed point arithmetic.
	MintDistributionAllowableTolerancePercent = 0.02 // 2%
	// MintDistributionAllowableToleranceAbsolution is similar to MintDistributionAllowableTolerancePercent
	// but provides an absolute number where the % difference might no be
	// meaningful for small absolute numbers.
	// TODO_MAINNET: Figure out if we can avoid this tolerance and use fixed point arithmetic.
	MintDistributionAllowableToleranceAbs = 5.0 // 5 uPOKT
)

var (
	// TODO_BETA(@red-0ne, #732): Make this a governance parameter and give it a non-zero value + tests.
	// GlobalInflationPerClaim is the percentage of the claim amount that is minted
	// by TLMGlobalMint to reward the actors in the network.
	GlobalInflationPerClaim = 0.1

	_ TokenLogicModule = (*tlmGlobalMint)(nil)
)

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

	if GlobalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin, newMintAmtFloat := CalculateGlobalPerClaimMintInflationFromSettlementAmount(tlmCtx.SettlementCoin)
	if newMintCoin.Amount.Int64() == 0 {
		return tokenomicstypes.ErrTokenomicsMintAmountZero
	}

	// Mint new uPOKT to the tokenomics module account
	tlmCtx.Result.AppendMint(tokenomicstypes.MintBurnOp{
		OpReason:          tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION,
		DestinationModule: tokenomicstypes.ModuleName,
		Coin:              newMintCoin,
	})
	logger.Info(fmt.Sprintf("operation queued: mint (%s) to the tokenomics module account", newMintCoin))

	// Send a portion of the rewards to the application
	appCoin, err := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_APPLICATION_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		tlmCtx.Application.GetAddress(),
		&newMintAmtFloat,
		MintAllocationApplication,
	)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to application: %v", err)
	}
	logMsg := fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the application with address %q", appCoin, tlmCtx.Application.GetAddress())
	logRewardOperation(logger, logMsg, &appCoin)

	// Send a portion of the rewards to the supplier shareholders.
	supplierCoinsToShareAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationSupplier)
	supplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierCoinsToShareAmt))
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
		uint64(supplierCoinsToShareAmt),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsModuleMint.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			tlmCtx.Supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("operation queued: send (%v) newley minted coins from the tokenomics module to the supplier with address %q", supplierCoin, tlmCtx.Supplier.OperatorAddress))

	// Send a portion of the rewards to the DAO
	daoRewardAddress := tlmCtx.Params.Tokenomics.GetDaoRewardAddress()
	daoCoin, err := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		daoRewardAddress,
		&newMintAmtFloat,
		MintAllocationDAO,
	)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to DAO: %v", err)
	}
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the DAO with address %q", daoCoin, daoRewardAddress)
	logRewardOperation(logger, logMsg, &daoCoin)

	// Send a portion of the rewards to the source owner
	serviceCoin, err := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		tlmCtx.Service.OwnerAddress,
		&newMintAmtFloat,
		MintAllocationSourceOwner,
	)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to source owner: %v", err)
	}
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the source owner with address %q", serviceCoin, tlmCtx.Service.OwnerAddress)
	logRewardOperation(logger, logMsg, &serviceCoin)

	// Send a portion of the rewards to the block proposer
	proposerAddr := cosmostypes.AccAddress(cosmostypes.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerCoin, err := sendRewardsToAccount(
		logger,
		tlmCtx.Result,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_PROPOSER_REWARD_DISTRIBUTION,
		tokenomicstypes.ModuleName,
		proposerAddr,
		&newMintAmtFloat,
		MintAllocationProposer,
	)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to proposer: %v", err)
	}
	logMsg = fmt.Sprintf("send (%v) newley minted coins from the tokenomics module to the proposer with address %q", proposerCoin, proposerAddr)
	logRewardOperation(logger, logMsg, &proposerCoin)

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
	coinDifference := new(big.Int).Sub(totalMintDistributedCoin.Amount.BigInt(), newMintCoin.Amount.BigInt())
	coinDifference = coinDifference.Abs(coinDifference)
	percentDifference := new(big.Float).Quo(new(big.Float).SetInt(coinDifference), new(big.Float).SetInt(newMintCoin.Amount.BigInt()))

	// Helper booleans for readability
	doesDiscrepancyExist := coinDifference.Cmp(big.NewInt(0)) > 0
	isPercentDifferenceTooLarge := percentDifference.Cmp(big.NewFloat(MintDistributionAllowableTolerancePercent)) > 0
	isAbsDifferenceSignificant := coinDifference.Cmp(big.NewInt(int64(MintDistributionAllowableToleranceAbs))) > 0

	// No discrepancy, return early
	logger.Info(fmt.Sprintf("operation queued: distribute (%v) coins to the application, supplier, DAO, source owner, and proposer", totalMintDistributedCoin))
	if !doesDiscrepancyExist {
		return nil
	}

	// Discrepancy exists and is too large, return an error
	if isPercentDifferenceTooLarge && isAbsDifferenceSignificant {
		return tokenomicstypes.ErrTokenomicsAmountMismatchTooLarge.Wrapf(
			"the total distributed coins (%v) do not equal the amount of newly minted coins (%v) with a percent difference of (%f). Likely floating point arithmetic.\n"+
				"appCoin: %v, supplierCoin: %v, daoCoin: %v, serviceCoin: %v, proposerCoin: %v",
			totalMintDistributedCoin, newMintCoin, percentDifference,
			appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin)
	}

	// Discrepancy exists but is within tolerance, log and return nil
	logger.Warn(fmt.Sprintf("Floating point arithmetic led to a discrepancy of %v (%f) between the total distributed coins (%v) and the amount of new minted coins (%v).\n"+
		"appCoin: %v, supplierCoin: %v, daoCoin: %v, serviceCoin: %v, proposerCoin: %v",
		coinDifference, percentDifference, totalMintDistributedCoin, newMintCoin,
		appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin))
	return nil
}

// sendRewardsToAccount sends (settlementAmtFloat * allocation) tokens from the
// tokenomics module account to the specified address.
func sendRewardsToAccount(
	logger cosmoslog.Logger,
	result *tokenomicstypes.SettlementResult,
	opReason tokenomicstypes.SettlementOpReason,
	senderModule string,
	recipientAddr string,
	settlementAmtFloat *big.Float,
	allocation float64,
) (cosmostypes.Coin, error) {
	logger = logger.With(
		"method", "mintRewardsToAccount",
		"session_id", result.GetSessionId(),
	)

	coinsToAccAmt := calculateAllocationAmount(settlementAmtFloat, allocation)
	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinsToAccAmt))

	if coinToAcc.IsZero() {
		return cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0), nil
	}

	result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
		OpReason:         opReason,
		SenderModule:     senderModule,
		RecipientAddress: recipientAddr,
		Coin:             coinToAcc,
	})

	logger.Info(fmt.Sprintf("operation queued: send (%v) coins from the tokenomics module to the account with address %q", coinToAcc, recipientAddr))

	return coinToAcc, nil
}

// CalculateGlobalPerClaimMintInflationFromSettlementAmount calculates the amount
// of uPOKT to mint based on the global per claim inflation rate as a function of
// the settlement amount for a particular claim(s) or session(s).
func CalculateGlobalPerClaimMintInflationFromSettlementAmount(
	settlementCoin cosmostypes.Coin,
) (cosmostypes.Coin, big.Float) {
	// Determine how much new uPOKT to mint based on global per claim inflation.
	// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
	settlementAmtFloat := new(big.Float).SetUint64(settlementCoin.Amount.Uint64())
	newMintAmtFloat := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(GlobalInflationPerClaim))
	// DEV_NOTE: If new mint is less than 1 and more than 0, ceil it to 1 so that
	// we never expect to process a claim with 0 minted tokens.
	if newMintAmtFloat.Cmp(big.NewFloat(1)) < 0 && newMintAmtFloat.Cmp(big.NewFloat(0)) > 0 {
		newMintAmtFloat = big.NewFloat(1)
	}
	newMintAmtInt, _ := newMintAmtFloat.Int64()
	mintAmtCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(newMintAmtInt))
	return mintAmtCoin, *newMintAmtFloat
}

// calculateAllocationAmount does big float arithmetic to determine the absolute
// amount from amountFloat based on the allocation percentage provided.
// TODO_MAINNET(@bryanchriswhite): Measure and limit the precision loss here.
func calculateAllocationAmount(
	amountFloat *big.Float,
	allocationPercentage float64,
) int64 {
	coinsToAccAmt, _ := big.NewFloat(0).Mul(amountFloat, big.NewFloat(allocationPercentage)).Int64()
	return coinsToAccAmt
}
