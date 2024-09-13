package keeper

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomictypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var (
	// Governance parameters for the TLMGlobalMint module
	// TODO_UPNEXT(@olshansk, #732): Make this a governance parameter and give it a non-zero value + tests.
	MintPerClaimedTokenGlobalInflation = 0.1

	// TODO_UPNEXT: Make these a govenrance parameter
	supplierStakeFloorMultiplier         = sdkmath.NewInt(0) // Set to 0 to disable
	supplierStakeWeightCeiling           = sdkmath.NewInt(100)
	supplierStakeFloorMultiplierExponent = int64(2)

	// TODO_UPNEXT: Update the Service type to contain the needed info to calculate
	// supplierStakeWeightedMultiplier
	supplierStakeWeightMultiplier = int64(1)
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
	MintDistributionAllowableToleranceAbs = 5 // 5 uPOKT

	PIP22ExponentDenominator = int64(100)
)

type TokenLogicModule int

const (
	// TLMRelayBurnEqualsMint is the token logic module that burns the application's
	// stake balance based on the amount of work done by the supplier.
	// The same amount of tokens is minted and added to the supplier account balance.
	// When the network achieves maturity in the far future, this is theoretically
	// the only TLM that will be necessary.
	TLMRelayBurnEqualsMint TokenLogicModule = iota

	// TLMGlobalMint is the token logic module that mints new tokens based on the
	// global governance parameters in order to reward the participants providing
	// services while keeping inflation in check.
	TLMGlobalMint

	// TLMStakeWeightedMint is the token logic module that mints new tokens based on the
	// stake of the supplier in order to reward high staking suppliers.
	TLMStakeWeightedMint
)

var tokenLogicModuleStrings = [...]string{
	"TLMRelayBurnEqualsMint",
	"TLMGlobalMint",
	"TLMStakeWeightedMint",
}

func (tlm TokenLogicModule) String() string {
	return tokenLogicModuleStrings[tlm]
}

type RewardedActor int

const (
	RewardedApplication RewardedActor = iota
	RewardedSupplier
	RewardedDAO
	RewardedProposer
	RewardedSourceOwner
)

var rewardedActorStrings = [...]string{
	"application",
	"supplier",
	"dao",
	"validator",
	"service owner",
}

func (ra RewardedActor) String() string {
	return rewardedActorStrings[ra]
}

// RewardInstruction is a struct that holds the cumulative reward amount and the
// address of the entity that will be receiving the rewards.
type RewardInstruction struct {
	Amount        sdkmath.Int
	rewardedActor RewardedActor
	moduleName    string
}

// RewardsAccumulator is a map of actor addresses to RewardInstructions that
// accumulates the rewards across the processed TLMs which will be minted
// and distributed at the end of the TLM processing.
// TODO_TECHDEBT: Extend this to support burning stake as well.
type RewardsAccumulator map[string]*RewardInstruction

// updateReward updates the rewards distribution map with the new reward amount.
// The reward amount could be positive or negative allowing for the rewards to be
// adjusted based on the TLMs processed.
func (rd RewardsAccumulator) updateReward(
	moduleName string,
	rewardedActor RewardedActor,
	rewardedActorAddr string,
	amount sdkmath.Int,
) {
	if _, ok := rd[rewardedActorAddr]; !ok {
		rd[rewardedActorAddr] = &RewardInstruction{
			Amount:        sdkmath.NewInt(0),
			rewardedActor: rewardedActor,
			moduleName:    moduleName,
		}
	}

	rd[rewardedActorAddr].Amount = rd[rewardedActorAddr].Amount.Add(amount)
}

// TokenLogicModuleProcessor is the method signature that all token logic modules
// are expected to implement.
// IMPORTANT_SIDE_EFFECTS: Please note that TLMs may update the application and supplier objects,
// which is why they are passed in as pointers. NOTE: TLMs SHOULD NOT persist any state changes.
// Persistence of updated application and supplier to the keeper is currently done by the TLM
// processor in `ProcessTokenLogicModules()`. This design and separation of concerns may change
// in the future.
// DEV_NOTE: As of writing this, this is only in anticipation of potentially unstaking
// actors if their stake falls below a certain threshold.
type TokenLogicModuleProcessor func(
	Keeper,
	context.Context,
	*sharedtypes.Service,
	*sessiontypes.SessionHeader,
	*apptypes.Application,
	*sharedtypes.Supplier,
	cosmostypes.Coin,
	*tokenomictypes.RelayMiningDifficulty,
	RewardsAccumulator,
) error

// tokenLogicModuleProcessorMap is a map of TLMs to their respective independent processors.
var tokenLogicModuleProcessorMap = map[TokenLogicModule]TokenLogicModuleProcessor{
	TLMRelayBurnEqualsMint: Keeper.TokenLogicModuleRelayBurnEqualsMint,
	TLMGlobalMint:          Keeper.TokenLogicModuleGlobalMint,
	TLMStakeWeightedMint:   Keeper.TokenLogicModuleStakeWeightedMint,
}

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner+MintAllocationApplication {
		panic("mint allocation percentages do not add to 1.0")
	}

	// TODO_UPNEXT(@Olshansk): Ensure that if `TLMGlobalMint` is present in the map,
	// then TLMGlobalMintReimbursementRequest will need to be there too.
}

// ProcessTokenLogicModules is the entrypoint for all TLM processing.
//
// It is responsible for running all the independent TLMs necessary to limit, burn, mint or transfer tokens as a result of the
// amount of work (i.e. relays, compute units) done in proportion to the global governance parameters.
//
// Prior to running the TLMs, it handles the business logic of converting the claimed
// amount to the actual settlement amount and handling the case for overserviced applications.
//
// IMPORTANT: It is assumed that the proof for the claim has been validated BEFORE calling this function.
func (k Keeper) ProcessTokenLogicModules(
	ctx context.Context,
	claim *prooftypes.Claim,
) (err error) {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// Telemetry variable declaration to be emitted at the end of the function
	claimSettlementCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, sdkmath.NewInt(0))
	isSuccessful := false

	// This is emitted only when the function returns (successful or not)
	defer telemetry.EventSuccessCounter(
		"process_token_logic_modules",
		func() float32 {
			if claimSettlementCoin.Amount.BigInt() == nil {
				return 0
			}
			return float32(claimSettlementCoin.Amount.Int64())
		},
		func() bool { return isSuccessful },
	)

	// Sanity check the claim is not nil. Validation of the claim is expected by the caller.
	if claim == nil {
		logger.Error("received a nil claim")
		return tokenomicstypes.ErrTokenomicsClaimNil
	}

	// Retrieve & validate the session header
	sessionHeader := claim.GetSessionHeader()
	if sessionHeader == nil {
		logger.Error("received a nil session header")
		return tokenomicstypes.ErrTokenomicsSessionHeaderNil
	}
	if err = sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return tokenomicstypes.ErrTokenomicsSessionHeaderInvalid
	}

	// Retrieve and validate the root of the claim to determine the amount of work done
	root := (smt.MerkleSumRoot)(claim.GetRootHash())
	if !root.HasDigestSize(protocol.TrieHasherSize) {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"root hash has invalid digest size (%d), expected (%d)",
			root.DigestSize(), protocol.TrieHasherSize,
		)
	}

	// Retrieve the count (i.e. number of relays) to determine the amount of work done
	numRelays, err := root.Count()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("%v", err)
	}
	// TODO_MAINNET(@bryanchriswhite, @red-0ne): Fix the low-volume exploit here.
	// https://www.notion.so/buildwithgrove/RelayMiningDifficulty-and-low-volume-7aab3edf6f324786933af369c2fa5f01?pvs=4
	if numRelays == 0 {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("root hash has zero relays")
	}

	/*
		TODO_POST_MAINNET: Because of how things have evolved, we are now using
		root.Count (numRelays) instead of root.Sum (numComputeUnits) to determine
		the amount of work done. This is because the compute_units_per_relay is
		a service specific (not request specific) parameter that will be maintained
		by the service owner to capture the average amount of resources (i.e.
		compute, storage, bandwidth, electricity, etc...) per request. Modifying
		this on a per request basis has been deemed too complex and not a mainnet
		blocker.
	*/

	// Retrieve the application address that is being charged; getting services and paying tokens.
	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid.Wrapf("address (%q)", sessionHeader.GetApplicationAddress())
	}

	// Retrieve the on-chain staked application record
	application, isAppFound := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !isAppFound {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the supplier operator address that will be getting rewarded; providing services and earning tokens
	supplierOperatorAddr, err := cosmostypes.AccAddressFromBech32(claim.GetSupplierOperatorAddress())
	if err != nil || supplierOperatorAddr == nil {
		return tokenomicstypes.ErrTokenomicsSupplierOperatorAddressInvalid.Wrapf("address (%q)", claim.GetSupplierOperatorAddress())
	}

	// Retrieve the on-chain staked supplier record
	supplier, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddr.String())
	if !isSupplierFound {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierOperatorAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	service, isServiceFound := k.serviceKeeper.GetService(ctx, sessionHeader.ServiceId)
	if !isServiceFound {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", sessionHeader.ServiceId)
	}

	// Determine the total number of tokens being claimed (i.e. for the work completed)
	// by the supplier for the amount of work they did to service the application
	// in the session.
	claimSettlementCoin, err = k.numRelaysToCoin(ctx, numRelays, &service)
	if err != nil {
		return err
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"num_relays", numRelays,
		"claim_settlement_upokt", claimSettlementCoin.Amount,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", sessionHeader.GetServiceId(),
		"supplier_operator", supplier.OperatorAddress,
		"application", application.Address,
	)

	// Retrieving the relay mining difficulty for the service at hand
	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		relayMiningDifficulty = newDefaultRelayMiningDifficulty(ctx, logger, service.Id, numRelays)
	}

	// Ensure the claim amount is within the limits set by Relay Mining.
	// If not, update the settlement amount and emit relevant events.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &application, &supplier, claimSettlementCoin)
	if err != nil {
		return err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)

	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) relays, equal to (%s) claimed", numRelays, actualSettlementCoin))

	// Execute all the token logic modules processors and collect the rewards distribution
	// to be minted and distributed once all TLMs have been processed.
	// This approach allows for further adjustments to the rewards distribution by TLMs
	// that might lower or increase the rewards based on the business logic.
	actorsRewardsState := make(RewardsAccumulator)
	for tlm, tlmProcessor := range tokenLogicModuleProcessorMap {
		logger.Info(fmt.Sprintf("Starting TLM processing: %q", tlm))
		if err := tlmProcessor(k, ctx, &service, claim.GetSessionHeader(), &application, &supplier, actualSettlementCoin, &relayMiningDifficulty, actorsRewardsState); err != nil {
			return tokenomictypes.ErrTokenomicsTLMError.Wrapf("TLM %q: %v", tlm, err)
		}
		logger.Info(fmt.Sprintf("Finished TLM processing: %q", tlm))
	}

	// Mint and distribute the resulting rewards based on rewardsDistribution
	k.mintAndDistributeRewards(ctx, logger, actorsRewardsState, &supplier, &service)

	// State mutation: update the application's on-chain record
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", application.Address))

	// TODO_MAINNET: If the application stake has dropped to (near?) zero, should
	// we unstake it? Should we use it's balance? Should there be a payee of last resort?
	// Make sure to document whatever decision we come to.

	// State mutation: Update the suppliers's on-chain record
	k.supplierKeeper.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("updated on-chain supplier record with address %q", supplier.OperatorAddress))

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return nil
}

// TokenLogicModuleRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
func (k Keeper) TokenLogicModuleRelayBurnEqualsMint(
	ctx context.Context,
	service *sharedtypes.Service,
	_ *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	actualSettlementCoin cosmostypes.Coin, // Note that actualSettlementCoin may differ from claimSettlementCoin; see ensureClaimAmountLimits for details.
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
	rewardsAccumulator RewardsAccumulator,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleRelayBurnEqualsMint")

	// DEV_NOTE: We are doing a burn & mint + transfer instead of a simple transfer
	// of funds from the application stake to the supplier balance in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(actualSettlementCoin)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", application.Address, newAppStake)
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %v", application.Address, newAppStake))

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	if err := k.bankKeeper.BurnCoins(
		ctx, apptypes.ModuleName, sdk.NewCoins(actualSettlementCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationModuleBurn.Wrapf("burning %s from the application module account: %v", actualSettlementCoin, err)
	}
	logger.Info(fmt.Sprintf("burned (%v) from the application module account", actualSettlementCoin))

	// Collect the rewards for the supplier to be minted and distributed after all
	// TLMs have been processed.
	rewardsAccumulator.updateReward(
		suppliertypes.ModuleName,
		RewardedSupplier,
		supplier.OperatorAddress,
		actualSettlementCoin.Amount,
	)

	return nil
}

// TokenLogicModuleGlobalMint processes the business logic for the GlobalMint TLM.
func (k Keeper) TokenLogicModuleGlobalMint(
	ctx context.Context,
	service *sharedtypes.Service,
	_ *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	actualSettlementCoin cosmostypes.Coin, // Note that actualSettlementCoin may differ from claimSettlementCoin; see ensureClaimAmountLimits for details.
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
	rewardsAccumulator RewardsAccumulator,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleGlobalMint")

	if MintPerClaimedTokenGlobalInflation == 0 {
		// TODO_UPNEXT(@olshansk): Make sure to skip GMRR TLM in this case as well.
		logger.Warn("global inflation is set to zero. Skipping Global Mint TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintAmtFloat := calculateGlobalPerClaimMintInflationFromSettlementAmount(actualSettlementCoin)
	if newMintAmtFloat.Cmp(big.NewFloat(0)) == 0 {
		return tokenomicstypes.ErrTokenomicsMintAmountZero
	}

	// Calculate the GlobalMint allocations and update the rewards for each actor.

	applicationMintAllocationAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationApplication)
	rewardsAccumulator.updateReward(
		// TODO_CONSIDERATION: Should we send the rewards from the corresponding module account?
		tokenomictypes.ModuleName,
		RewardedApplication,
		application.GetAddress(),
		applicationMintAllocationAmt,
	)

	supplierMintAllocationAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationSupplier)
	rewardsAccumulator.updateReward(
		tokenomictypes.ModuleName,
		RewardedSupplier,
		supplier.GetOperatorAddress(),
		supplierMintAllocationAmt,
	)

	daoMintAllocationAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationDAO)
	rewardsAccumulator.updateReward(
		tokenomictypes.ModuleName,
		RewardedDAO,
		k.GetAuthority(),
		daoMintAllocationAmt,
	)

	sourceOwnerMintAllocationAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationSourceOwner)
	rewardsAccumulator.updateReward(
		tokenomictypes.ModuleName,
		RewardedSourceOwner,
		service.GetOwnerAddress(),
		sourceOwnerMintAllocationAmt,
	)

	proposerAddr := cosmostypes.AccAddress(sdk.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerMintAllocationAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationProposer)
	rewardsAccumulator.updateReward(
		tokenomictypes.ModuleName,
		RewardedProposer,
		proposerAddr,
		proposerMintAllocationAmt,
	)

	allocations := map[RewardedActor]sdkmath.Int{
		RewardedApplication: applicationMintAllocationAmt,
		RewardedSupplier:    supplierMintAllocationAmt,
		RewardedDAO:         daoMintAllocationAmt,
		RewardedSourceOwner: sourceOwnerMintAllocationAmt,
		RewardedProposer:    proposerMintAllocationAmt,
	}

	newMintAmtInt, _ := newMintAmtFloat.Int(nil)
	// Check and log the total amount of coins distributed
	if err := k.ensureMintedCoinsAreDistributed(logger, sdkmath.NewIntFromBigInt(newMintAmtInt), allocations); err != nil {
		return err
	}

	return nil
}

// TokenLogicModuleStakeWeightedMint processes the business logic for the StakeWeightedMint TLM.
// It adjusts the rewards of BurnEqualsMint rewards by adding or subtracting tokens based on the
// stake of the supplier.
// This TLM does not affect the application's stake but only the supplier's stake.
func (k Keeper) TokenLogicModuleStakeWeightedMint(
	ctx context.Context,
	service *sharedtypes.Service,
	_ *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	actualSettlementCoin cosmostypes.Coin, // Note that actualSettlementCoin may differ from claimSettlementCoin; see ensureClaimAmountLimits for details.
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
	rewardsAccumulator RewardsAccumulator,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleRelayBurnEqualsMint")

	if supplierStakeFloorMultiplier.Int64() == 0 {
		logger.Warn("supplier stake floor multiplier is set to zero. Skipping Stake Weighted Mint TLM.")
		return nil
	}

	if supplierStakeWeightMultiplier == 0 {
		return types.ErrTokenomicsTLMError.Wrapf("supplier stake weight multiplier is set to zero")
	}

	stake := supplier.Stake.Amount
	cappedFlooredStake := sdkmath.MinInt(
		stake.Sub(stake.Mod(supplierStakeFloorMultiplier)),
		supplierStakeWeightCeiling.Sub(supplierStakeWeightCeiling.Mod(supplierStakeFloorMultiplier)),
	)

	// Int division could never lose 1 or more in precision. This should be safe
	// as long as it yields the same result.
	bin := cappedFlooredStake.Quo(supplierStakeFloorMultiplier)

	weight := fracPow(bin, supplierStakeFloorMultiplierExponent, PIP22ExponentDenominator).
		Quo(sdkmath.NewInt(supplierStakeWeightMultiplier))

	stakeWeightedReward := actualSettlementCoin.Amount.Mul(weight)

	// Calculate the difference between the stake-weighted reward and the actual
	// settlement amount if the stake-weighted reward is less than the actual
	// settlement amount, the supplier gets its reward reduced by the difference.
	baseRewardStakeWeightedRewardDiff := stakeWeightedReward.Sub(actualSettlementCoin.Amount)

	// Update the supplier's stake with the stake-weighted reward
	rewardsAccumulator.updateReward(
		suppliertypes.ModuleName,
		RewardedSupplier,
		supplier.OperatorAddress,
		baseRewardStakeWeightedRewardDiff,
	)

	return nil
}

func (k Keeper) mintAndDistributeRewards(
	ctx context.Context,
	logger log.Logger,
	rewardsDistribution RewardsAccumulator,
	supplier *sharedtypes.Supplier,
	service *sharedtypes.Service,
) error {
	for rewardedActorAddr, rewardInstruction := range rewardsDistribution {
		rewardAmount := rewardInstruction.Amount
		moduleName := rewardInstruction.moduleName
		rewardedActor := rewardInstruction.rewardedActor

		// If the reward amount is zero, log and skip the distribution.
		if rewardAmount.IsZero() {
			logger.Warn(fmt.Sprintf(
				"skipping 0 coins distribution from %q module to %s with address %q",
				rewardedActor, moduleName, rewardedActorAddr,
			))
			continue
		}

		// Ensure the reward amount is not negative
		if rewardAmount.IsNegative() {
			return tokenomicstypes.ErrTokenomicsRewardDistributionFailed.Wrapf(
				"negative reward amount (%d) for %s with address %q",
				rewardAmount, rewardedActor, rewardedActorAddr,
			)
		}

		// Always mint the coins to the tokenomics module account.
		rewardCoin := sdk.NewCoin(volatile.DenomuPOKT, rewardAmount)
		if err := k.bankKeeper.MintCoins(ctx, tokenomicstypes.ModuleName, sdk.NewCoins(rewardCoin)); err != nil {
			return tokenomicstypes.ErrTokenomicsRewardDistributionFailed.Wrapf("minting %s to the %q module account: %v", rewardCoin, moduleName, err)
		}
		logger.Info(fmt.Sprintf("minted (%s) coins in the %q module account", rewardCoin, "tokenomics"))

		// If moduleName is not the tokenomics module, send the funds to the target
		// module account before distributing the rewards to the target actor.
		if moduleName != tokenomictypes.ModuleName {
			// Send funds from the tokenomics module to the target module account.
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, tokenomicstypes.ModuleName, moduleName, sdk.NewCoins(rewardCoin)); err != nil {
				return tokenomicstypes.ErrTokenomicsRewardDistributionFailed.Wrapf(
					"transferring (%s) from the %q module account to the %q module account: %v",
					rewardCoin,
					"tokenomics",
					moduleName,
					err,
				)
			}
		}

		switch rewardedActor {
		// Supplier rewards are distributed to the supplier's shareholders and should
		// have their reward distribution handled separately.
		case RewardedSupplier:
			if err := k.distributeSupplierRewardsToShareHolders(ctx, supplier, service.Id, rewardAmount); err != nil {
				return tokenomicstypes.ErrTokenomicsRewardDistributionFailed.Wrapf(
					"distributing rewards from the %q module to supplier with operator address %q shareholders: %v",
					moduleName,
					supplier.OperatorAddress,
					err,
				)
			}
			logger.Info(fmt.Sprintf(
				"sent (%s) from the %q module to the supplier account with address %q",
				moduleName, rewardCoin, supplier.OperatorAddress,
			))
		default:
			actorCoin, err := k.sendRewardsToAccount(ctx, moduleName, rewardedActorAddr, rewardAmount)
			if err != nil {
				return tokenomictypes.ErrTokenomicsRewardDistributionFailed.Wrapf(
					"sending rewards from module %q to %s with address %q: %v",
					moduleName, rewardedActor, rewardedActorAddr, err,
				)
			}
			logger.Debug(fmt.Sprintf(
				"sent (%s) newly minted coins from the %q module to the %s with address %q",
				actorCoin, moduleName, rewardedActor, rewardedActorAddr,
			))
		}
	}

	return nil
}

// ensureMintedCoinsAreDistributed checks if the total amount of minted coins is equal to the
// amount of coins distributed to pocket network participants while accounting for floating point
// arithmetic discrepancies.
func (k Keeper) ensureMintedCoinsAreDistributed(
	logger log.Logger,
	newMintAmt sdkmath.Int,
	allocations map[RewardedActor]sdkmath.Int,
) error {
	totalMintDistributedAmt := sdkmath.NewInt(0)
	// Collect the rewarded actor types and their respective amounts for logging.
	rewardedActorNamesToAmounts := make(map[string]sdkmath.Int)
	for rewardedActor, amount := range allocations {
		totalMintDistributedAmt = totalMintDistributedAmt.Add(amount)
		rewardedActorNamesToAmounts[rewardedActor.String()] = amount
	}

	// Compute the difference between the total distributed coins and the amount of newly minted coins
	amtDifference := totalMintDistributedAmt.Sub(newMintAmt).Abs()
	percentDifference := new(big.Float).Quo(new(big.Float).SetInt(amtDifference.BigInt()), new(big.Float).SetInt(newMintAmt.BigInt()))

	// Helper booleans for readability
	doesDiscrepancyExist := !amtDifference.IsZero()
	isPercentDifferenceTooLarge := percentDifference.Cmp(big.NewFloat(MintDistributionAllowableTolerancePercent)) > 0
	isAbsDifferenceSignificant := amtDifference.GT(sdkmath.NewInt(MintDistributionAllowableToleranceAbs))

	// No discrepancy, return early
	logger.Info(fmt.Sprintf("distributed total %d coins %v", totalMintDistributedAmt, rewardedActorNamesToAmounts))
	if !doesDiscrepancyExist {
		return nil
	}

	// Discrepancy exists and is too large, return an error
	if isPercentDifferenceTooLarge || isAbsDifferenceSignificant {
		return tokenomictypes.ErrTokenomicsAmountMismatchTooLarge.Wrapf(
			"the total distributed coins (%d) do not equal the amount of newly minted coins (%d) with a percent difference of (%f). Likely floating point arithmetic.\n%v",
			totalMintDistributedAmt, newMintAmt, percentDifference, rewardedActorNamesToAmounts,
		)
	}

	// Discrepancy exists but is within tolerance, log and return nil
	logger.Warn(fmt.Sprintf(
		"Floating point arithmetic led to a discrepancy of %d (%f) between the total distributed coins (%d) and the amount of new minted coins (%d).\n%v",
		amtDifference, percentDifference, totalMintDistributedAmt, newMintAmt, rewardedActorNamesToAmounts,
	))
	return nil
}

// sendRewardsToAccount sends rewardAmt tokens from the tokenomics module account
// to the specified address.
func (k Keeper) sendRewardsToAccount(
	ctx context.Context,
	srcModule string,
	destAdr string,
	rewardAmt sdkmath.Int,
) (sdk.Coin, error) {
	logger := k.Logger().With("method", "mintRewardsToAccount")

	accountAddr, err := cosmostypes.AccAddressFromBech32(destAdr)
	if err != nil {
		return sdk.Coin{}, err
	}

	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, rewardAmt)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, srcModule, accountAddr, sdk.NewCoins(coinToAcc),
	); err != nil {
		return sdk.Coin{}, err
	}
	logger.Info(fmt.Sprintf("sent (%v) coins from the tokenomics module to the account with address %q", coinToAcc, destAdr))

	return coinToAcc, nil
}

// ensureClaimAmountLimits checks if the application was overserviced and handles
// the case if it was.
// Per Algorithm #1 in the Relay Mining paper, the maximum amount that a single supplier
// can claim in a session is AppStake/NumSuppliersPerSession.
// If this is not the case, then the supplier essentially did "free work" and the
// actual claim amount is less than what was claimed.
// Ref: https://arxiv.org/pdf/2305.10672
func (k Keeper) ensureClaimAmountLimits(
	ctx context.Context,
	logger log.Logger,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// TODO_BETA_OR_MAINNET(@red-0ne): The application stake gets reduced with every claim
	// settlement. Relay miners use the appStake at the beginning of a session to determine
	// the maximum amount they can claim. We need to somehow access and propagate this
	// value (via context?) so it is the same for all TLM processors for each claim.
	// Note that this will also need to incorporate MintPerClaimGlobalInflation because
	// applications are being overcharged by that amount in the meantime. Whatever the
	// solution and implementation ends up being, make sure to KISS.
	appStake := application.GetStake()

	// Determine the max claimable amount for the supplier based on the application's stake in this session.
	maxClaimableCoin := sdk.NewCoin(volatile.DenomuPOKT, appStake.Amount.Quo(sdkmath.NewInt(sessionkeeper.NumSupplierPerSession)))

	if maxClaimableCoin.Amount.GTE(claimSettlementCoin.Amount) {
		logger.Info(fmt.Sprintf("Claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount >= Claim amount: %v >= %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableCoin, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	logger.Warn(fmt.Sprintf("Claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < Claim amount: %v < %v",
		supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableCoin, claimSettlementCoin.Amount))

	// Reduce the settlement amount if the application was over-serviced
	actualSettlementCoins = maxClaimableCoin

	// Prepare and emit the event for the application being overserviced
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr:      application.Address,
		SupplierOperatorAddr: supplier.GetOperatorAddress(),
		ExpectedBurn:         &claimSettlementCoin,
		EffectiveBurn:        &maxClaimableCoin,
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
		return cosmostypes.Coin{},
			tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf("error emitting event %v", applicationOverservicedEvent)
	}

	return actualSettlementCoins, nil
}

// numRelaysToCoin calculates the amount of uPOKT to mint based on the number of compute units.
func (k Keeper) numRelaysToCoin(
	ctx context.Context,
	numRelays uint64, // numRelays is a session specific parameter
	service *sharedtypes.Service,
) (cosmostypes.Coin, error) {
	// CUTTM is a GLOBAL network wide parameter
	computeUnitsToTokensMultiplier := k.GetParams(ctx).ComputeUnitsToTokensMultiplier
	// CUPR is a LOCAL service specific parameter
	computeUnitsPerRelay := service.ComputeUnitsPerRelay
	upoktAmount := sdkmath.NewInt(int64(numRelays * computeUnitsPerRelay * computeUnitsToTokensMultiplier))
	if upoktAmount.IsNegative() {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return cosmostypes.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}

// distributeSupplierRewardsToShareHolders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
func (k Keeper) distributeSupplierRewardsToShareHolders(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute sdkmath.Int,
) error {
	logger := k.Logger().With("method", "distributeSupplierRewardsToShareHolders")

	var serviceRevShare []*sharedtypes.ServiceRevenueShare
	for _, svc := range supplier.Services {
		if svc.ServiceId == serviceId {
			serviceRevShare = svc.RevShare
			break
		}
	}

	if serviceRevShare == nil {
		return tokenomicstypes.ErrTokenomicsSupplierRevShareFailed.Wrapf(
			"service %q not found for supplier %v",
			serviceId,
			supplier,
		)
	}

	shareAmountMap := GetShareAmountMap(serviceRevShare, amountToDistribute)
	for shareHolderAddress, shareAmount := range shareAmountMap {
		// TODO_TECHDEBT(@red-0ne): Refactor to reuse the sendRewardsToAccount helper here.
		shareAmountCoin, err := k.sendRewardsToAccount(ctx, suppliertypes.ModuleName, shareHolderAddress, shareAmount)
		if err != nil {
			return err
		}

		logger.Info(fmt.Sprintf(
			"sent %s from the supplier module to the supplier shareholder with address %q",
			shareAmountCoin, supplier.GetOperatorAddress(),
		))
	}

	logger.Info(fmt.Sprintf(
		"distributed %d uPOKT to supplier %q shareholders",
		amountToDistribute, supplier.GetOperatorAddress(),
	))

	return nil
}

// calculateGlobalPerClaimMintInflationFromSettlementAmount calculates the amount
// of uPOKT to mint based on the global per claim inflation rate as a function of
// the settlement amount for a particular claim(s) or session(s).
func calculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoin sdk.Coin) big.Float {
	// Determine how much new uPOKT to mint based on global per claim inflation.
	// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
	settlementAmtFloat := new(big.Float).SetUint64(settlementCoin.Amount.Uint64())
	newMintAmtFloat := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintPerClaimedTokenGlobalInflation))
	return *newMintAmtFloat
}

// calculateAllocationAmount does big float arithmetic to determine the absolute
// amount from amountFloat based on the allocation percentage provided.
// TODO_MAINNET(@bryanchriswhite): Measure and limit the precision loss here.
func calculateAllocationAmount(
	amountFloat *big.Float,
	allocationPercentage float64,
) sdkmath.Int {
	coinsToAccAmtFloat := big.NewFloat(0).Mul(amountFloat, big.NewFloat(allocationPercentage))
	coinsToAccAmtInt, _ := coinsToAccAmtFloat.Int(nil)

	return sdkmath.NewIntFromBigInt(coinsToAccAmtInt)
}

// GetShareAmountMap calculates the amount of uPOKT to distribute to each revenue
// shareholder based on the rev share percentage of the service.
// It returns a map of the shareholder address to the amount of uPOKT to distribute.
// The first shareholder gets any remainder due to floating point arithmetic.
// NB: It is publically exposed to be used in the tests.
func GetShareAmountMap(
	serviceRevShare []*sharedtypes.ServiceRevenueShare,
	amountToDistribute sdkmath.Int,
) (shareAmountMap map[string]sdkmath.Int) {
	totalDistributed := sdkmath.NewInt(0)
	shareAmountMap = make(map[string]sdkmath.Int, len(serviceRevShare))
	for _, revShare := range serviceRevShare {
		// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
		sharePercentageFloat := big.NewFloat(float64(revShare.RevSharePercentage) / 100)
		amountToDistributeFloat := new(big.Float).SetInt(amountToDistribute.BigInt())
		shareAmount, _ := big.NewFloat(0).Mul(amountToDistributeFloat, sharePercentageFloat).Int(nil)
		shareAmountMap[revShare.Address] = sdkmath.NewIntFromBigInt(shareAmount)
		totalDistributed = totalDistributed.Add(sdkmath.NewIntFromBigInt(shareAmount))
	}

	// Add any remainder due to floating point arithmetic to the first shareholder.
	remainder := amountToDistribute.Sub(totalDistributed)
	shareAmountMap[serviceRevShare[0].Address] = shareAmountMap[serviceRevShare[0].Address].Add(remainder)

	return shareAmountMap
}

// fracPow calculates the fractional power of a base integer to an exponent with a denominator.
// It uses floating point arithmetic to calculate the power and then converts the result back to an integer.
// TODO_MAINNET: This is a far form ideal implementation and needs refactoring
// to use fixed point arithmetic for deterministic results and really be careful
// about precision loss.
func fracPow(base sdkmath.Int, exponent int64, denominator int64) sdkmath.Int {
	// Convert base to a big.Float for exponentiation
	baseFloat, _ := new(big.Float).SetInt(base.BigInt()).Float64()

	// Compute the fractional exponent
	expFloat := float64(exponent) / float64(denominator)

	// Compute the power using floating-point arithmetic
	resultFloat := new(big.Float).SetFloat64(math.Pow(baseFloat, expFloat))

	// Convert the result back to an integer
	resultInt, _ := resultFloat.Int(nil)

	return sdkmath.NewIntFromBigInt(resultInt)
}
