package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var (
	// Governance parameters for the TLMGlobalMint module
	// TODO_BETA(@red-0ne, #732): Make this a governance parameter.
	// GlobalInflationPerClaim is the percentage of the claim amount that is minted
	// by TLMGlobalMint to reward the actors in the network.
	GlobalInflationPerClaim = 0.1
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
	// TODO_MAINNET(@red-0ne): Figure out if we can avoid this tolerance and use fixed point arithmetic.
	MintDistributionAllowableTolerancePercent = 0.02 // 2%
	// MintDistributionAllowableToleranceAbsolution is similar to MintDistributionAllowableTolerancePercent
	// but provides an absolute number where the % difference might no be
	// meaningful for small absolute numbers.
	// TODO_MAINNET(@red-0ne): Figure out if we can avoid this tolerance and use fixed point arithmetic.
	MintDistributionAllowableToleranceAbs = 5.0 // 5 uPOKT
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

	// TLMGlobalMintReimbursementRequest is the token logic module that complements
	// TLMGlobalMint to enable permissionless demand.
	// In order to prevent self-dealing attacks, applications will be overcharged by
	// the amount equal to global inflation, those funds will be sent to the DAO/PNF,
	// and an event will be emitted to track and send reimbursements; managed offchain by PNF.
	// TODO_POST_MAINNET: Introduce proper tokenomics based on the research done by @rawthil and @shane.
	TLMGlobalMintReimbursementRequest
)

var tokenLogicModuleStrings = [...]string{
	"TLMRelayBurnEqualsMint",
	"TLMGlobalMint",
	"TLMGlobalMintReimbursementRequest",
}

func (tlm TokenLogicModule) String() string {
	return tokenLogicModuleStrings[tlm]
}

func (tlm TokenLogicModule) EnumIndex() int {
	return int(tlm)
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
	cosmostypes.Coin, // This is the "actualSettlementCoin" rather than just the "claimCoin" because of how settlement functions; see ensureClaimAmountLimits for details.
	*servicetypes.RelayMiningDifficulty,
) error

// tokenLogicModuleProcessorMap is a map of TLMs to their respective independent processors.
var tokenLogicModuleProcessorMap = map[TokenLogicModule]TokenLogicModuleProcessor{
	TLMRelayBurnEqualsMint:            Keeper.TokenLogicModuleRelayBurnEqualsMint,
	TLMGlobalMint:                     Keeper.TokenLogicModuleGlobalMint,
	TLMGlobalMintReimbursementRequest: Keeper.TokenLogicModuleGlobalMintReimbursementRequest,
}

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner+MintAllocationApplication {
		panic("mint allocation percentages do not add to 1.0")
	}

	_, hasGlobalMintTLM := tokenLogicModuleProcessorMap[TLMGlobalMint]
	_, hasGlobalMintReimbursementRequestTLM := tokenLogicModuleProcessorMap[TLMGlobalMintReimbursementRequest]
	if hasGlobalMintTLM != hasGlobalMintReimbursementRequestTLM {
		panic("TLMGlobalMint and TLMGlobalMintReimbursementRequest must be (de-)activated together")
	}
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
	applicationInitialStake cosmostypes.Coin,
) (err error) {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// Telemetry variable declaration to be emitted at the end of the function
	claimSettlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
	isSuccessful := false

	// This is emitted only when the function returns (successful or not)
	defer telemetry.EventSuccessCounter(
		"process_token_logic_modules",
		func() float32 {
			if claimSettlementCoin.Amount.BigInt() == nil {
				return 0
			}

			// Avoid out of range errors by converting to float64 first
			claimSettlementFloat64, _ := claimSettlementCoin.Amount.BigInt().Float64()
			return float32(claimSettlementFloat64)
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

	// Retrieve the sum (i.e. number of compute units) to determine the amount of work done
	numClaimComputeUnits, err := claim.GetNumClaimedComputeUnits()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("failed to retrieve numClaimComputeUnits: %s", err)
	}
	// TODO_MAINNET(@bryanchriswhite, @red-0ne): Fix the low-volume exploit here.
	// https://www.notion.so/buildwithgrove/RelayMiningDifficulty-and-low-volume-7aab3edf6f324786933af369c2fa5f01?pvs=4
	if numClaimComputeUnits == 0 {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("root hash has zero relays")
	}

	numRelays, err := claim.GetNumRelays()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("failed to retrieve numRelays: %s", err)
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

	// Ensure the number of compute units claimed is equal to the number of relays * CUPR
	expectedClaimComputeUnits := numRelays * service.ComputeUnitsPerRelay
	if numClaimComputeUnits != expectedClaimComputeUnits {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"mismatch: claim compute units (%d) != number of relays (%d) * service compute units per relay (%d)",
			numClaimComputeUnits,
			numRelays,
			service.ComputeUnitsPerRelay,
		)
	}

	// Retrieving the relay mining difficulty for service.
	relayMiningDifficulty, found := k.serviceKeeper.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(ctx, logger, service.Id, servicekeeper.TargetNumRelays)
	}
	sharedParams := k.sharedKeeper.GetParams(ctx)

	// Determine the total number of tokens being claimed (i.e. for the work completed)
	// by the supplier for the amount of work they did to service the application
	// in the session.
	claimSettlementCoin, err = claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	if err != nil {
		return err
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"num_relays", numRelays,
		"num_claim_compute_units", numClaimComputeUnits,
		"claim_settlement_upokt", claimSettlementCoin.Amount,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", sessionHeader.GetServiceId(),
		"supplier_operator", supplier.OperatorAddress,
		"application", application.Address,
	)

	// Ensure the claim amount is within the limits set by Relay Mining.
	// If not, update the settlement amount and emit relevant events.
	// TODO_MAINNET(@red-0ne): Consider pulling this out of Keeper#ProcessTokenLogicModules
	// and ensure claim amount limits are enforced before TLM processing.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &sharedParams, &application, &supplier, claimSettlementCoin, applicationInitialStake)
	if err != nil {
		return err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)
	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) compute units, equal to (%s) claimed", numClaimComputeUnits, actualSettlementCoin))

	// TODO_MAINNET(@red-0ne): Add tests to ensure that a zero settlement coin
	// due to integer division rounding is handled correctly.
	if actualSettlementCoin.Amount.IsZero() {
		logger.Warn(fmt.Sprintf(
			"actual settlement coin is zero, skipping TLM processing, application %q stake %s",
			application.Address, application.Stake,
		))
		return nil
	}

	// Execute all the token logic modules processors
	for tlm, tlmProcessor := range tokenLogicModuleProcessorMap {
		logger.Info(fmt.Sprintf("Starting TLM processing: %q", tlm))
		if err = tlmProcessor(k, ctx, &service, claim.GetSessionHeader(), &application, &supplier, actualSettlementCoin, &relayMiningDifficulty); err != nil {
			return tokenomicstypes.ErrTokenomicsTLMError.Wrapf("TLM %q: %v", tlm, err)
		}
		logger.Info(fmt.Sprintf("Finished TLM processing: %q", tlm))
	}

	// TODO_POST_MAINNET: If we support multiple native tokens, we will need to start checking the denom here.
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	if application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {
		// Mark the application as unbonding if it has less than the minimum stake.
		application.UnstakeSessionEndHeight = uint64(sessionEndHeight)
		unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &application)

		appUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        &application,
			Reason:             apptypes.ApplicationUnbondingReason_BELOW_MIN_STAKE,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		}

		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		if err = sdkCtx.EventManager().EmitTypedEvent(appUnbondingBeginEvent); err != nil {
			err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", appUnbondingBeginEvent, err)
			logger.Error(err.Error())
			return err
		}
	}

	// State mutation: update the application's on-chain record.
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", application.Address))

	// TODO_MAINNET(@bryanchriswhite): If the application stake has dropped to (near?) zero:
	// - Unstake it
	// - Emit an event
	// - Ensure this doesn't happen
	// - Document the decision

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
	settlementCoin cosmostypes.Coin,
	relayMiningDifficulty *servicetypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleRelayBurnEqualsMint")

	// DEV_NOTE: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the application stake to the supplier balance in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint new uPOKT to the supplier module account.
	// These funds will be transferred to the supplier's shareholders below.
	// For reference, see operate/configs/supplier_staking_config.md.
	if err := k.bankKeeper.MintCoins(
		ctx, suppliertypes.ModuleName, sdk.NewCoins(settlementCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleSendFailed.Wrapf(
			"minting %s to the supplier module account: %v",
			settlementCoin,
			err,
		)
	}

	// Update telemetry information
	if settlementCoin.Amount.IsInt64() {
		defer telemetry.MintedTokensFromModule(suppliertypes.ModuleName, float32(settlementCoin.Amount.Int64()))
	}

	logger.Debug(fmt.Sprintf("minted (%v) coins in the supplier module", settlementCoin))

	// Distribute the rewards to the supplier's shareholders based on the rev share percentage.
	if err := k.distributeSupplierRewardsToShareHolders(ctx, supplier, service.Id, settlementCoin.Amount.Uint64()); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			supplier.OperatorAddress,
			err,
		)
	}
	logger.Debug(fmt.Sprintf("sent (%v) from the supplier module to the supplier account with address %q", settlementCoin, supplier.OperatorAddress))

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	if err := k.bankKeeper.BurnCoins(
		ctx, apptypes.ModuleName, sdk.NewCoins(settlementCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationModuleBurn.Wrapf("burning %s from the application module account: %v", settlementCoin, err)
	}

	// Update telemetry information
	if settlementCoin.Amount.IsInt64() {
		defer telemetry.BurnedTokensFromModule(apptypes.ModuleName, float32(settlementCoin.Amount.Int64()))
	}

	logger.Debug(fmt.Sprintf("burned (%v) from the application module account", settlementCoin))

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(settlementCoin)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", application.Address, newAppStake)
	}
	application.Stake = &newAppStake
	logger.Debug(fmt.Sprintf("updated application %q stake to %v", application.Address, newAppStake))

	return nil
}

// TokenLogicModuleGlobalMint processes the business logic for the GlobalMint TLM.
func (k Keeper) TokenLogicModuleGlobalMint(
	ctx context.Context,
	service *sharedtypes.Service,
	_ *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoin cosmostypes.Coin,
	relayMiningDifficulty *servicetypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleGlobalMint")

	if GlobalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin, newMintAmtFloat := CalculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoin)
	if newMintCoin.Amount.Int64() == 0 {
		return tokenomicstypes.ErrTokenomicsMintAmountZero
	}

	// Mint new uPOKT to the tokenomics module account
	if err := k.bankKeeper.MintCoins(ctx, tokenomicstypes.ModuleName, sdk.NewCoins(newMintCoin)); err != nil {
		return tokenomicstypes.ErrTokenomicsModuleMintFailed.Wrapf(
			"minting (%s) to the tokenomics module account: %v", newMintCoin, err)
	}

	// Update telemetry information
	if newMintCoin.Amount.IsInt64() {
		defer telemetry.MintedTokensFromModule(tokenomicstypes.ModuleName, float32(newMintCoin.Amount.Int64()))
	}

	logger.Info(fmt.Sprintf("minted (%s) to the tokenomics module account", newMintCoin))

	// Send a portion of the rewards to the application
	appCoin, err := k.sendRewardsToAccount(ctx, tokenomicstypes.ModuleName, application.GetAddress(), &newMintAmtFloat, MintAllocationApplication)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to application: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the application with address %q", appCoin, application.Address))

	// Send a portion of the rewards to the supplier shareholders.
	supplierCoinsToShareAmt := calculateAllocationAmount(&newMintAmtFloat, MintAllocationSupplier)
	supplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierCoinsToShareAmt))
	// Send funds from the tokenomics module to the supplier module account
	if err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, tokenomicstypes.ModuleName, suppliertypes.ModuleName, sdk.NewCoins(supplierCoin)); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleSendFailed.Wrapf(
			"transferring (%s) from the tokenomics module account to the supplier module account: %v",
			supplierCoin,
			err,
		)
	}
	// Distribute the rewards from within the supplier's module account.
	if err = k.distributeSupplierRewardsToShareHolders(ctx, supplier, service.Id, uint64(supplierCoinsToShareAmt)); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			supplier.OperatorAddress,
			err,
		)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the supplier with address %q", supplierCoin, supplier.OperatorAddress))

	// Send a portion of the rewards to the DAO
	daoCoin, err := k.sendRewardsToAccount(ctx, tokenomicstypes.ModuleName, k.GetAuthority(), &newMintAmtFloat, MintAllocationDAO)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to DAO: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the DAO with address %q", daoCoin, k.GetAuthority()))

	// Send a portion of the rewards to the source owner
	serviceCoin, err := k.sendRewardsToAccount(ctx, tokenomicstypes.ModuleName, service.OwnerAddress, &newMintAmtFloat, MintAllocationSourceOwner)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to source owner: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the source owner with address %q", serviceCoin, service.OwnerAddress))

	// Send a portion of the rewards to the block proposer
	proposerAddr := cosmostypes.AccAddress(sdk.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerCoin, err := k.sendRewardsToAccount(ctx, tokenomicstypes.ModuleName, proposerAddr, &newMintAmtFloat, MintAllocationProposer)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to proposer: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the proposer with address %q", proposerCoin, proposerAddr))

	// Check and log the total amount of coins distributed
	if err := k.ensureMintedCoinsAreDistributed(logger, appCoin, supplierCoin, daoCoin, serviceCoin, proposerCoin, newMintCoin); err != nil {
		return err
	}

	return nil
}

// TokenLogicModuleGlobalMintReimbursementRequest processes the business logic
// for the GlobalMintReimbursementRequest TLM.
func (k Keeper) TokenLogicModuleGlobalMintReimbursementRequest(
	ctx context.Context,
	service *sharedtypes.Service,
	sessionHeader *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	actualSettlementCoin cosmostypes.Coin,
	relayMiningDifficulty *servicetypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleGlobalMintReimbursementRequest")

	// Do not process the reimbursement request if there is no global inflation.
	if GlobalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint Reimbursement Request TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin, _ := CalculateGlobalPerClaimMintInflationFromSettlementAmount(actualSettlementCoin)
	if newMintCoin.Amount.Int64() == 0 {
		return tokenomicstypes.ErrTokenomicsMintAmountZero
	}

	newAppStake, err := application.Stake.SafeSub(newMintCoin)
	// This should THEORETICALLY NEVER fall below zero.
	// `ensureClaimAmountLimits` should have already checked and adjusted the settlement
	// amount so that the application stake covers the global inflation.
	// TODO_POST_MAINNET: Consider removing this since it should never happen just to simplify the code
	if err != nil {
		return err
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %s", application.Address, newAppStake))

	globalInflationMintedCoinsForClaim := sdk.NewCoins(newMintCoin)

	// Send the global per claim mint inflation uPOKT from the tokenomics module
	// account to PNF/DAO.
	daoAccountAddr, err := cosmostypes.AccAddressFromBech32(k.GetAuthority())
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationReimbursementRequestFailed.Wrapf(
			"getting PNF/DAO address: %v",
			err,
		)
	}

	// Send the global per claim mint inflation uPOKT from the application module
	// account to the tokenomics module account as an intermediary step.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx, apptypes.ModuleName, tokenomicstypes.ModuleName, globalInflationMintedCoinsForClaim,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationReimbursementRequestFailed.Wrapf(
			"sending %s from the application module account to the tokenomics module account: %v",
			newMintCoin, err,
		)
	}
	logger.Info(fmt.Sprintf(
		"sent (%s) from the application module account to the tokenomics module account",
		newMintCoin,
	))

	// Send the global per claim mint inflation uPOKT from the tokenomics module
	// for second order economic effects.
	// See: https://discord.com/channels/824324475256438814/997192534168182905/1299372745632649408
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, tokenomicstypes.ModuleName, daoAccountAddr, globalInflationMintedCoinsForClaim,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationReimbursementRequestFailed.Wrapf(
			"sending %s from the tokenomics module account to the PNF/DAO account: %v",
			newMintCoin, err,
		)
	}

	// Prepare and emit the event for the application that'll required reimbursement.
	// Recall that it is being overcharged to compoensate for global inflation while
	// preventing self-dealing attacks.
	reimbursementRequestEvent := &tokenomicstypes.EventApplicationReimbursementRequest{
		ApplicationAddr:      application.Address,
		SupplierOperatorAddr: supplier.OperatorAddress,
		SupplierOwnerAddr:    supplier.OwnerAddress,
		ServiceId:            service.Id,
		SessionId:            sessionHeader.SessionId,
		Amount:               &newMintCoin,
	}

	eventManger := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManger.EmitTypedEvent(reimbursementRequestEvent); err != nil {
		err = tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf(
			"(%+v): %s",
			reimbursementRequestEvent, err,
		)

		logger.Error(err.Error())
		return err
	}

	return nil
}

func (k Keeper) ensureMintedCoinsAreDistributed(
	logger log.Logger,
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
	logger.Info(fmt.Sprintf("distributed (%v) coins to the application, supplier, DAO, source owner, and proposer", totalMintDistributedCoin))
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
func (k Keeper) sendRewardsToAccount(
	ctx context.Context,
	srcModule string,
	destAdr string,
	settlementAmtFloat *big.Float,
	allocation float64,
) (sdk.Coin, error) {
	logger := k.Logger().With("method", "mintRewardsToAccount")

	accountAddr, err := cosmostypes.AccAddressFromBech32(destAdr)
	if err != nil {
		return sdk.Coin{}, err
	}

	coinsToAccAmt := calculateAllocationAmount(settlementAmtFloat, allocation)
	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinsToAccAmt))
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
	sharedParams *sharedtypes.Params,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
	initialApplicationStake cosmostypes.Coin,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// Note that this also incorporates MintPerClaimGlobalInflation since applications
	// are being overcharged by that amount and the funds are sent to the DAO/PNF
	// before being reimbursed to the application in the future.
	appStake := initialApplicationStake

	// The application should have enough stake to cover for the global mint reimbursement.
	// This amount is deducted from the maximum claimable amount.
	globalInflationCoin, _ := CalculateGlobalPerClaimMintInflationFromSettlementAmount(claimSettlementCoin)
	globalInflationAmt := globalInflationCoin.Amount
	minRequiredAppStakeAmt := claimSettlementCoin.Amount.Add(globalInflationAmt)
	totalClaimedCoin := sdk.NewCoin(volatile.DenomuPOKT, minRequiredAppStakeAmt)

	// get the number of pending sessions that share the application stake at claim time
	// This is used to calculate the maximum claimable amount for the supplier within a session.
	numPendingSessions := sharedtypes.GetNumPendingSessions(sharedParams)

	// The maximum any single supplier can claim is a fraction of the app's total stake
	// divided by the number of suppliers per session.
	// Re decentralization - This ensures the app biases towards using all suppliers in a session.
	// Re costs - This is an easy way to split the stake evenly.
	// TODO_FUTURE: See if there's a way to let the application prefer (the best)
	// supplier(s) in a session while maintaining a simple solution to implement this.
	numSuppliersPerSession := int64(k.sessionKeeper.GetParams(ctx).NumSuppliersPerSession)
	maxClaimableAmt := appStake.Amount.
		Quo(math.NewInt(numSuppliersPerSession)).
		Quo(math.NewInt(numPendingSessions))
	maxClaimSettlementAmt := supplierAppStakeToMaxSettlementAmount(maxClaimableAmt)

	// Check if the claimable amount is capped by the max claimable amount.
	// As per the Relay Mining paper, the Supplier claim MUST NOT exceed the application's
	// allocated stake. If it does, the claim is capped by the application's allocated stake
	// and the supplier is effectively "overserviced".
	if minRequiredAppStakeAmt.GT(maxClaimableAmt) {
		logger.Warn(fmt.Sprintf("Claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < Claim amount: %v < %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableAmt, claimSettlementCoin.Amount))

		minRequiredAppStakeAmt = maxClaimableAmt
		maxClaimSettlementAmt = supplierAppStakeToMaxSettlementAmount(minRequiredAppStakeAmt)
	}

	// Nominal case: The claimable amount is within the limits set by Relay Mining.
	if claimSettlementCoin.Amount.LTE(maxClaimSettlementAmt) {
		logger.Info(fmt.Sprintf("Claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount >= Claim amount: %v >= %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimSettlementAmt, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	// Claimable amount is capped by the max claimable amount or the application allocated stake.
	// Determine the max claimable amount for the supplier based on the application's stake in this session.
	maxClaimableCoin := sdk.NewCoin(volatile.DenomuPOKT, maxClaimSettlementAmt)

	// Prepare and emit the event for the application being overserviced
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr:      application.GetAddress(),
		SupplierOperatorAddr: supplier.GetOperatorAddress(),
		ExpectedBurn:         &totalClaimedCoin,
		EffectiveBurn:        &maxClaimableCoin,
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
		return cosmostypes.Coin{},
			tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf("error emitting event %v", applicationOverservicedEvent)
	}

	return maxClaimableCoin, nil
}

// distributeSupplierRewardsToShareHolders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
func (k Keeper) distributeSupplierRewardsToShareHolders(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute uint64,
) error {
	logger := k.Logger().With("method", "distributeSupplierRewardsToShareHolders")

	var serviceRevShare []*sharedtypes.ServiceRevenueShare
	for _, svc := range supplier.Services {
		if svc.ServiceId == serviceId {
			serviceRevShare = svc.RevShare
			break
		}
	}

	// This should theoretically never happen because the following validation
	// is done during staking: MsgStakeSupplier.ValidateBasic() -> ValidateSupplierServiceConfigs() -> ValidateServiceRevShare().
	// The check is here just for redundancy.
	// TODO_MAINNET(@red-0ne): Double check this doesn't happen.
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
		shareAmountCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(int64(shareAmount)))
		shareAmountCoins := cosmostypes.NewCoins(shareAmountCoin)
		shareHolderAccAddress, err := sdk.AccAddressFromBech32(shareHolderAddress)
		if err != nil {
			return err
		}

		// Send the newley minted uPOKT from the supplier module account
		// to the supplier's shareholders.
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, suppliertypes.ModuleName, shareHolderAccAddress, shareAmountCoins,
		); err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("sent %s from the supplier module to the supplier shareholder with address %q", shareAmountCoin, supplier.GetOperatorAddress()))
	}

	logger.Info(fmt.Sprintf("distributed %d uPOKT to supplier %q shareholders", amountToDistribute, supplier.GetOperatorAddress()))

	return nil
}

// CalculateGlobalPerClaimMintInflationFromSettlementAmount calculates the amount
// of uPOKT to mint based on the global per claim inflation rate as a function of
// the settlement amount for a particular claim(s) or session(s).
// DEV_NOTE: This function is publically exposed to be used in the tests.
func CalculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoin sdk.Coin) (sdk.Coin, big.Float) {
	// Determine how much new uPOKT to mint based on global per claim inflation.
	// TODO_MAINNET(@red-0ne): Consider using fixed point arithmetic for deterministic results.
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

// supplierAppStakeToMaxSettlementAmount calculates the max amount of uPOKT the supplier
// can claim based on the stake allocated to the supplier and the global inflation
// allocation percentage.
// This is the inverse of CalculateGlobalPerClaimMintInflationFromSettlementAmount:
// stake = maxSettlementAmt + globalInflationAmt
// stake = maxSettlementAmt + (maxSettlementAmt * MintPerClaimedTokenGlobalInflation)
// stake = maxSettlementAmt * (1 + MintPerClaimedTokenGlobalInflation)
// maxSettlementAmt = stake / (1 + MintPerClaimedTokenGlobalInflation)
func supplierAppStakeToMaxSettlementAmount(stakeAmount math.Int) math.Int {
	stakeAmountFloat := big.NewFloat(0).SetInt(stakeAmount.BigInt())
	maxSettlementAmountFloat := big.NewFloat(0).Quo(stakeAmountFloat, big.NewFloat(1+GlobalInflationPerClaim))

	settlementAmount, _ := maxSettlementAmountFloat.Int(nil)
	return math.NewIntFromBigInt(settlementAmount)
}

// calculateAllocationAmount does big float arithmetic to determine the absolute
// amount from amountFloat based on the allocation percentage provided.
// TODO_MAINNET(@red-0ne): Measure and limit the precision loss here.
func calculateAllocationAmount(
	amountFloat *big.Float,
	allocationPercentage float64,
) int64 {
	coinsToAccAmt, _ := big.NewFloat(0).Mul(amountFloat, big.NewFloat(allocationPercentage)).Int64()
	return coinsToAccAmt
}

// GetShareAmountMap calculates the amount of uPOKT to distribute to each revenue
// shareholder based on the rev share percentage of the service.
// It returns a map of the shareholder address to the amount of uPOKT to distribute.
// The first shareholder gets any remainder due to floating point arithmetic.
// NB: It is publically exposed to be used in the tests.
func GetShareAmountMap(
	serviceRevShare []*sharedtypes.ServiceRevenueShare,
	amountToDistribute uint64,
) (shareAmountMap map[string]uint64) {
	totalDistributed := uint64(0)
	shareAmountMap = make(map[string]uint64, len(serviceRevShare))
	for _, revShare := range serviceRevShare {
		// TODO_MAINNET(@red-0ne): Consider using fixed point arithmetic for deterministic results.
		sharePercentageFloat := big.NewFloat(float64(revShare.RevSharePercentage) / 100)
		amountToDistributeFloat := big.NewFloat(float64(amountToDistribute))
		shareAmount, _ := big.NewFloat(0).Mul(amountToDistributeFloat, sharePercentageFloat).Uint64()
		shareAmountMap[revShare.Address] = shareAmount
		totalDistributed += shareAmount
	}

	// Add any remainder due to floating point arithmetic to the first shareholder.
	remainder := amountToDistribute - totalDistributed
	shareAmountMap[serviceRevShare[0].Address] += remainder

	return shareAmountMap
}
