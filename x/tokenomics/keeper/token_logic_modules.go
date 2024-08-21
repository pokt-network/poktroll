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
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomictypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// Governance parameters for the TLMGlobalMint module
	// TODO_UPNEXT(@olshansk, #732): Make this a governance parameter and give it a non-zero value + tests.
	MintPerClaimGlobalInflation = 0.0000000

	// TODO_BETA(@bryanchriswhite): Make all of the governance params
	MintAllocationDAO         = 0.1
	MintAllocationProposer    = 0.05
	MintAllocationSupplier    = 0.7
	MintAllocationSourceOwner = 0.15
	MintAllocationApplication = 0.0
)

type TokenLogicModule int

const (
	// TLMRelayBurnEqualsMint is the token logic module that burns the application's
	// stake based on the amount of work done by the supplier.
	// The same amount of tokens is minted and sent to the supplier.
	// When the network achieves equilibrium, this is theoretically the only TLM that will be necessary.
	TLMRelayBurnEqualsMint TokenLogicModule = iota

	// TLMGlobalMint is the token logic module that mints new tokens based on the
	// on global governance parameters in order to reward the participants providing
	// services while keeping inflation in check.
	TLMGlobalMint

	// TLMGlobalMintReimbursementRequest is the token logic module that complements
	// TLMGlobalMint to enable permissionless demand. In order to prevent self-dealing
	// attacks, applications will be overcharged by the amount equal to global inflation,
	// those funds will be sent to the DAO/PNF, and an event will be emitted to track
	// and send reimbursements; managed offchain by PNF.
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
// IMPORTANT_SIDE_EFFECTS: Please note that TLMs may update the application and supplier
// objects, which is why they are passed in as pointers. NOTE THAT THIS IS NOT PERSISTED.
// The persistence to the keeper is currently done by the TLM processor: ProcessTokenLogicModules.
// This design and separation of concerns may change in the future.
type TokenLogicModuleProcessor func(
	Keeper,
	context.Context,
	*sharedtypes.Service,
	*sessiontypes.SessionHeader,
	*apptypes.Application,
	*sharedtypes.Supplier,
	cosmostypes.Coin,
	*tokenomictypes.RelayMiningDifficulty,
) error

// tokenLogicModuleProcessorMap is a map of TLMs to their respective independent processors.
var tokenLogicModuleProcessorMap = map[TokenLogicModule]TokenLogicModuleProcessor{
	TLMRelayBurnEqualsMint: Keeper.TokenLogicModuleRelayBurnEqualsMint,
	TLMGlobalMint:          Keeper.TokenLogicModuleGlobalMint,
	// TODO_UPNEXT(@olshansk, #732): Uncomment this, finish implementation, and add tests.
	// TLMGlobalMintReimbursementRequest: Keeper.TokenLogicModuleGlobalMintReimbursementRequest,
}

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner+MintAllocationApplication {
		panic("mint allocation percentages do not add to 1.0")
	}
}

// ProcessTokenLogicModules is the main TLM processor. It is responsible for running
// all of the independent TLMs necessary to limit, burn, mint or transfer tokens
// as a result of the amount of work (i.e. relays, compute units) done in proportion
// to the global governance parameters.
// IMPORTANT: It is assumed the proof for the claim has been validated BEFORE calling this function.
func (k Keeper) ProcessTokenLogicModules(
	ctx context.Context,
	claim *prooftypes.Claim,
) (err error) {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// Telemetry variable declaration to be emitted a the end of the function
	claimSettlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
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

	// Retrieve the application address that is being charged; getting services and paying tokens
	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid
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
		return tokenomicstypes.ErrTokenomicsSupplierOperatorAddressInvalid
	}

	// Retrieve the on-chain staked supplier record
	supplier, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddr.String())
	if !isSupplierFound {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierOperatorAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	// Retrieve the service that the supplier is providing
	service, isServiceFound := k.serviceKeeper.GetService(ctx, sessionHeader.Service.Id)
	if !isServiceFound {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", sessionHeader.Service.Id)
	}

	// Determine the total number of tokens being claimed (i.e. requested)
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
		"service_id", sessionHeader.GetService().Id,
		"supplier_operator", supplier.OperatorAddress,
		"application", application.Address,
	)

	// Retrieving the relay mining difficulty for the service at hand
	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		relayMiningDifficulty = newDefaultRelayMiningDifficulty(ctx, logger, service.Id, numRelays)
	}

	// Ensure the claim amount is within the limits set by Relay Mining.
	// Update the settlement amount if not and emit any necessary events in doing so.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &application, &supplier, claimSettlementCoin)
	if err != nil {
		return err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)

	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) relays equaling to (%s) upokt claimed", numRelays, actualSettlementCoin))
	// Execute all the token logic modules processors
	for tlm, tlmProcessor := range tokenLogicModuleProcessorMap {
		logger.Info(fmt.Sprintf("Starting TLM processing: %q", tlm))
		if err := tlmProcessor(k, ctx, &service, claim.SessionHeader, &application, &supplier, actualSettlementCoin, &relayMiningDifficulty); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("Finished TLM processing: %q", tlm))
	}

	// State mutation: update the application's on-chain record
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", application.Address))

	// TODO_MAINNET: If the application stake has dropped to (near?) zero, should
	// we unstake it? Should we use it's balance? Should their be a payee of last resort?
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
	settlementCoin cosmostypes.Coin,
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleRelayBurnEqualsMint")

	// DEV_NOTE: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	ownerAddr, err := cosmostypes.AccAddressFromBech32(supplier.OwnerAddress)
	if err != nil {
		return err
	}

	// Mint new uPOKT to the supplier module account.
	// These funds will be transferred to the supplier below.
	if err = k.bankKeeper.MintCoins(
		ctx, suppliertypes.ModuleName, sdk.NewCoins(settlementCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleSendFailed.Wrapf(
			"minting %s to the supplier module account: %v",
			settlementCoin,
			err,
		)
	}
	logger.Info(fmt.Sprintf("minted (%v) coins in the supplier module", settlementCoin))

	// Distribute the rewards to the supplier's shareholders based on the rev share percentage.
	if err = k.distributeSupplierRewardsToShareHolders(ctx, ownerAddr.String(), service.Id, settlementCoin.Amount.Uint64()); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			supplier.OperatorAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent (%v) from the supplier module to the supplier account with address %q", settlementCoin, supplier.OperatorAddress))

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	if err = k.bankKeeper.BurnCoins(
		ctx, apptypes.ModuleName, sdk.NewCoins(settlementCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationModuleBurn.Wrapf("burning %s from the application module account: %v", settlementCoin, err)
	}
	logger.Info(fmt.Sprintf("burned (%v) from the application module account", settlementCoin))

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(settlementCoin)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", application.Address, newAppStake)
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %v", application.Address, newAppStake))

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
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleGlobalMint")

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoins, newMintAmtFloat := calculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoin)

	// Mint new uPOKT to the tokenomics module account
	if err := k.bankKeeper.MintCoins(ctx, tokenomictypes.ModuleName, newMintCoins); err != nil {
		return tokenomicstypes.ErrTokenomicsModuleMintFailed.Wrapf(
			"minting %s to the tokenomics module account: %v", newMintCoins, err)
	}
	logger.Info(fmt.Sprintf("minted (%v) coins in the tokenomics module", newMintCoins))

	// Send a portion of the rewards to the application
	appCoin, err := k.sendRewardsToAccount(ctx, application.Address, newMintAmtFloat, MintAllocationApplication)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to application: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the application with address %q", appCoin, application.Address))

	// Send a portion of the rewards to the supplier shareholders.
	supplierCoinsToShareAmt := calculateAllocationAmount(newMintAmtFloat, MintAllocationSupplier)
	supplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierCoinsToShareAmt))
	if err = k.distributeSupplierRewardsToShareHolders(ctx, supplier.OperatorAddress, service.Id, uint64(supplierCoinsToShareAmt)); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with operator address %s shareholders: %v",
			supplier.OperatorAddress,
			err,
		)
	}

	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the supplier with address %q", supplierCoin, supplier.OperatorAddress))

	// Send a portion of the rewards to the DAO
	daoCoin, err := k.sendRewardsToAccount(ctx, k.GetAuthority(), newMintAmtFloat, MintAllocationDAO)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to DAO: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the DAO with address %q", daoCoin, k.GetAuthority()))

	// Send a portion of the rewards to the source owner
	serviceCoin, err := k.sendRewardsToAccount(ctx, service.OwnerAddress, newMintAmtFloat, MintAllocationSourceOwner)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to source owner: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the source owner with address %q", serviceCoin, service.OwnerAddress))

	// Send a portion of the rewards to the block proposer
	proposerAddr := cosmostypes.AccAddress(sdk.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerCoin, err := k.sendRewardsToAccount(ctx, proposerAddr, newMintAmtFloat, MintAllocationProposer)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to proposer: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the proposer with address %q", proposerCoin, proposerAddr))

	// Check and log the total amount of coins distributed
	totalDistributedCoins := appCoin.Add(supplierCoin).Add(*daoCoin).Add(*serviceCoin).Add(*proposerCoin)
	if totalDistributedCoins.Amount.BigInt().Cmp(newMintCoins[0].Amount.BigInt()) != 0 {
		return tokenomictypes.ErrTokenomicsAmountMismatch.Wrapf("the total distributed coins (%v) do not equal the settlement coins (%v). Likely floating point arithmetic.", totalDistributedCoins, settlementCoin.Amount.BigInt())
	}
	logger.Info(fmt.Sprintf("distributed (%v) coins to the application, supplier, DAO, source owner, and proposer", totalDistributedCoins))

	return nil
}

// TokenLogicModuleGlobalMintReimbursementRequest processes the business logic for the GlobalMintReimbursementRequest TLM.
func (k Keeper) TokenLogicModuleGlobalMintReimbursementRequest(
	ctx context.Context,
	service *sharedtypes.Service,
	sessionHeader *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
) error {

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoins, _ := calculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoins)

	/*
		TODO_UPNEXT(@olshansk, #732): Finish implementing this:
		1. Overcharge the application (reduce stake and burn app module tokens)
		2. Send the overcharge to the DAO/PNF address
		3. Add necessary tests.
	*/

	// Prepare and emit the event for the application being overcharged
	reimbursementRequestEvent := tokenomictypes.EventApplicationReimbursementRequest{
		ApplicationAddr: application.Address,
		ServiceId:       service.Id,
		SessionId:       sessionHeader.SessionId,
		Amount:          &newMintCoins[0],
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManager.EmitTypedEvent(&reimbursementRequestEvent); err != nil {
		return tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf("error emitting event %v", reimbursementRequestEvent)
	}

	return nil
}

// sendRewardsToAccount sends (settlementAmtFloat * allocation) tokens from the
// tokenomics module account to the specified address.
func (k Keeper) sendRewardsToAccount(
	ctx context.Context,
	addr string,
	settlementAmtFloat *big.Float,
	allocation float64,
) (*sdk.Coin, error) {
	logger := k.Logger().With("method", "mintRewardsToAccount")

	accountAddr, err := cosmostypes.AccAddressFromBech32(addr)
	if err != nil {
		return nil, err
	}

	coinsToAccAmt := calculateAllocationAmount(settlementAmtFloat, allocation)
	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinsToAccAmt))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, accountAddr, sdk.NewCoins(coinToAcc),
	); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("sent (%v) coins from the tokenomics module to the account with address %q", coinToAcc, addr))

	return &coinToAcc, nil
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
	methodLogger log.Logger,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger := methodLogger.With("helper", "ensureClaimAmountLimits")

	// TODO_BETA_OR_MAINNET(@red-0ne): The application stake gets reduced with every claim
	// settlement. Relay miners use the appStake at the beginning of a session to determine
	// the maximum amount they can claim. We need to somehow access and propagate this
	// value (via context?) so it is the same for all TLM processors for each claim.
	// Note that this will also need to incorporate MintPerClaimGlobalInflation because
	// applications are being overcharged by that amount in the meantime. Whatever the
	// solution and implementation ends up being, make sure to KISS.
	appStake := application.GetStake()

	// Determine the max claimable amount for the supplier based on the application's stake
	// in this session.
	maxClaimableCoin := sdk.NewCoin(volatile.DenomuPOKT, appStake.Amount.Quo(math.NewInt(sessionkeeper.NumSupplierPerSession)))

	if maxClaimableCoin.Amount.GTE(claimSettlementCoin.Amount) {
		logger.Info(fmt.Sprintf("Claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount < Claim amount: %v < %v",
			supplier.OperatorAddress, application.Address, maxClaimableCoin, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	logger.Warn(fmt.Sprintf("Claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < Claim amount: %v < %v",
		supplier.OperatorAddress, application.Address, maxClaimableCoin, claimSettlementCoin.Amount))

	// Reduce the settlement amount if the application was over-serviced
	actualSettlementCoins = maxClaimableCoin

	// Prepare and emit the event for the application being overserviced
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr: application.Address,
		SupplierAddr:    supplier.OperatorAddress,
		ExpectedBurn:    &claimSettlementCoin,
		EffectiveBurn:   &maxClaimableCoin,
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
	upoktAmount := math.NewInt(int64(numRelays * computeUnitsPerRelay * computeUnitsToTokensMultiplier))
	if upoktAmount.IsNegative() {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return cosmostypes.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}

// distributeSupplierRewardsToShareHolders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
func (k Keeper) distributeSupplierRewardsToShareHolders(
	ctx context.Context,
	supplierAddr string,
	serviceId string,
	amountToDistribute uint64,
) error {
	logger := k.Logger().With("method", "distributeSupplierRewardsToShareHolders")

	supplier, supplierFound := k.supplierKeeper.GetSupplier(ctx, supplierAddr)
	if !supplierFound {
		return tokenomicstypes.ErrTokenomicsSupplierRevShareFailed.Wrapf(
			"supplier with address %q not found",
			supplierAddr,
		)
	}

	var serviceRevShare []*sharedtypes.ServiceRevenueShare
	for _, svc := range supplier.Services {
		if svc.Service.Id == serviceId {
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

		logger.Info(fmt.Sprintf("sent %s from the supplier module to the supplier shareholder with address %q", shareAmountCoin, supplierAddr))
	}

	logger.Info(fmt.Sprintf("distributed %d uPOKT to supplier %q shareholders", amountToDistribute, supplierAddr))

	return nil
}

// calculateGlobalPerClaimMintInflationFromSettlementAmount calculates the amount
// of uPOKT to mint based on the global per claim inflation rate as a function of
// the settlement amount for a particular claim(s) or session(s).
func calculateGlobalPerClaimMintInflationFromSettlementAmount(settlementCoin sdk.Coin) (sdk.Coins, *big.Float) {
	// Determine how much new uPOKT to mint based on global per claim inflation.
	// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
	settlementAmtFloat := new(big.Float).SetUint64(settlementCoin.Amount.Uint64())
	newMintAmtFloat := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintPerClaimGlobalInflation))
	newMintAmtInt, _ := newMintAmtFloat.Int64()
	mintAmtCoins := sdk.NewCoins(cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(newMintAmtInt)))
	return mintAmtCoins, newMintAmtFloat
}

// calculateAllocationAmount does big float arithmetic to determine the absolute
// amount from amountFloat based on the allocation percentage provided.
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
		// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
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
