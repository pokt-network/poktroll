package keeper

// References:
// - https://docs.pokt.network/pokt-protocol/the-shannon-upgrade/proposed-tokenomics/token-logic-modules
// - https://github.com/pokt-network/shannon-tokenomics-static-tests

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomictypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// TODO_UPNEXT(@olshansk): Make all of the governance params
	MintAllocationDAO         = 0.1
	MintAllocationProposer    = 0.05
	MintAllocationSupplier    = 0.7
	MintAllocationSourceOwner = 0.15
	MintAllocationApplication = 0.0
	// TODO_UPNEXT(@olshansk): Remove this. An ephemeral placeholder before
	// real values are introduced. When this is changed to a governance param,
	// make sure to also add the necessary unit tests.
	MintGlobalAllocation = 0.0000000
)

type TokenLogicModule int

const (
	TLMRelayBurnEqualsMint TokenLogicModule = iota
	TLMGlobalMint
	// TODO_UPNEXT(@olshansk): Add more TLMs
)

var tokenLogicModuleStrings = [...]string{
	"TLMRelayBurnEqualsMint",
	"TLMGlobalMint",
}

func (tlm TokenLogicModule) String() string {
	return tokenLogicModuleStrings[tlm]
}

func (tlm TokenLogicModule) EnumIndex() int {
	return int(tlm)
}

// TokenLogicModuleProcessor is the method signature that all token logic modules
// are expected to implement.
// IMPORTANT SIDE EFFECTS: Please note that TLMS may update the application and supplier
// objects, which is why they are passed in as pointers. However, this IS NOT persisted.
// The persistence to the keeper is currently done by ProcessTokenLogicModules only.
// This may be an interim state of the implementation and may change in the future.
type TokenLogicModuleProcessor func(
	Keeper,
	context.Context,
	*sharedtypes.Service,
	*apptypes.Application,
	*sharedtypes.Supplier,
	cosmostypes.Coin,
	*tokenomictypes.RelayMiningDifficulty,
) error

// tokenLogicModuleProcessorMap is a map of token logic modules to their respective processors.
var tokenLogicModuleProcessorMap = map[TokenLogicModule]TokenLogicModuleProcessor{
	TLMRelayBurnEqualsMint: Keeper.TokenLogicModuleRelayBurnEqualsMint,
	TLMGlobalMint:          Keeper.TokenLogicModuleGlobalMint,
}

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner+MintAllocationApplication {
		panic("mint allocation percentages do not add to 1.0")
	}
}

// ProcessTokenLogicModules is responsible for calling all of the token logic
// modules (i.e. post session claim accounting) necessary to burn, mint or transfer
// tokens as a result of the amount of work (i.e. compute units) done.
func (k Keeper) ProcessTokenLogicModules(
	ctx context.Context,
	claim *prooftypes.Claim, // IMPORTANT: It is assumed the proof for the claim has been validated BEFORE calling this function
) (err error) {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// Declaring variables that will be emitted by telemetry
	settlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
	isSuccessful := false

	// This is emitted only when the function returns (successful or not)
	defer telemetry.EventSuccessCounter(
		"process_token_logic_modules",
		func() float32 {
			if settlementCoin.Amount.BigInt() == nil {
				return 0
			}
			return float32(settlementCoin.Amount.Int64())
		},
		func() bool { return isSuccessful },
	)

	// Ensure the claim is not nil
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

	// Retrieve the supplier address that will be getting rewarded; providing services
	supplierAddr, err := cosmostypes.AccAddressFromBech32(claim.GetSupplierAddress())
	if err != nil || supplierAddr == nil {
		return tokenomicstypes.ErrTokenomicsSupplierAddressInvalid
	}

	// Retrieve the application address that is being charged; getting services
	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid
	}

	// Retrieve the root of the claim to determine the amount of work done
	root := (smt.MerkleSumRoot)(claim.GetRootHash())

	// Ensure the root hash is valid
	if !root.HasDigestSize(protocol.TrieHasherSize) {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"root hash has invalid digest size (%d), expected (%d)",
			root.DigestSize(), protocol.TrieHasherSize,
		)
	}

	// Retrieve the on-chain staked application record
	application, isAppFound := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !isAppFound {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the on-chain staked supplier record
	supplier, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierAddr.String())
	if !isSupplierFound {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	service, isServiceFound := k.serviceKeeper.GetService(ctx, sessionHeader.Service.Id)
	if !isServiceFound {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", sessionHeader.Service.Id)
	}

	// Retrieve the count (i.e. number of relays) to determine the amount of work done
	numRelays, err := root.Count()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("%v", err)
	}
	// TODO_POST_MAINNET: Because of how things have evolved, we are now using
	// root.Count (numRelays) instead of root.Sum (numComputeUnits) to determine
	// the amount of work done. This is because the compute_units_per_relay is
	/// a service specific (not request specific) parameter that will be maintained
	// by the service owner to capture the average amount of resources (i.e.
	// compute, storage, bandwidth, electricity, etc...) per request. Modifying
	// this on a per request basis has been deemed too complex and not a mainnet
	// blocker.

	// Determine the total number of tokens that'll be used for settling the session.
	// When the network achieves equilibrium, this will be the mint & burn.
	settlementCoin, err = k.numRelaysToCoin(ctx, numRelays, &service)
	if err != nil {
		return err
	}

	// Retrieving the relay mining difficulty for the service at hand
	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		if err != nil {
			return err
		}
		logger.Warn(fmt.Sprintf("relay mining difficulty for service %q not found. Using default difficulty", service.Id))
		relayMiningDifficulty = tokenomicstypes.RelayMiningDifficulty{
			ServiceId:    service.Id,
			BlockHeight:  sdk.UnwrapSDKContext(ctx).BlockHeight(),
			NumRelaysEma: numRelays,
			TargetHash:   prooftypes.DefaultRelayDifficultyTargetHash,
		}
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"num_relays", numRelays,
		"num_settlement_upokt", settlementCoin.Amount,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", sessionHeader.GetService().Id,
		"supplier", supplier.Address,
		"application", application.Address,
	)
	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) relays equaling to (%s) coins", numRelays, settlementCoin))

	// Execute all the token logic modules processors
	for tlm, tlmProcessor := range tokenLogicModuleProcessorMap {
		logger.Info(fmt.Sprintf("Starting to execute TLM %q", tlm))
		if err := tlmProcessor(k, ctx, &service, &application, &supplier, settlementCoin, &relayMiningDifficulty); err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("Finished executing TLM %q", tlm))
	}

	// Update the application's on-chain record
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", application.Address))

	// Update the suppliers's on-chain record
	k.supplierKeeper.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("updated on-chain supplier record with address %q", supplier.Address))

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return nil
}

// TokenLogicModuleRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
func (k Keeper) TokenLogicModuleRelayBurnEqualsMint(
	ctx context.Context,
	service *sharedtypes.Service,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoin cosmostypes.Coin,
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleRelayBurnEqualsMint")

	supplierAddr, err := cosmostypes.AccAddressFromBech32(supplier.Address)
	if err != nil {
		return err
	}

	// NB: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

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

	amount := settlementCoin.Amount.Uint64()
	if err := k.distributeSupplierRewardsToShareHolders(ctx, supplierAddr.String(), service.Id, amount); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with address %s shareholders: %v",
			supplierAddr,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent (%v) from the supplier module to the supplier account with address %q", settlementCoin, supplier.Address))

	// TODO_MAINNET: Decide on the behaviour here when an app is over serviced.
	// If an app has 10 POKT staked, but the supplier earned 20 POKT. We still
	// end up minting 20 POKT but only burn 10 POKT from the app. There are
	// questions and nuance here that needs to be addressed.

	// Verify that the application has enough uPOKT to pay for the services it consumed
	if application.GetStake().IsLT(settlementCoin) {
		settlementCoin, err = k.handleOverservicedApplication(ctx, application, settlementCoin)
		if err != nil {
			return err
		}
	}

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
// TODO_UPNEXT(@olshansk): Delete this in favor of a real TLM that mints tokens
// and distributes them to the appropriate accounts via boosts.
func (k Keeper) TokenLogicModuleGlobalMint(
	ctx context.Context,
	service *sharedtypes.Service,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
	relayMiningDifficulty *tokenomictypes.RelayMiningDifficulty,
) error {
	logger := k.Logger().With("method", "TokenLogicModuleGlobalMint")

	// Determine how much new uPOKT to mint based on global inflation
	// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
	settlementAmtFloat := new(big.Float).SetUint64(settlementCoins.Amount.Uint64())
	newMintAmtFloat := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintGlobalAllocation))
	newMintAmtInt, _ := newMintAmtFloat.Int64()
	newMintCoins := sdk.NewCoins(cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(newMintAmtInt)))

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
	if err := k.distributeSupplierRewardsToShareHolders(ctx, supplier.Address, service.Id, uint64(newMintAmtInt)); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"distributing rewards to supplier with address %s shareholders: %v",
			supplier.Address,
			err,
		)
	}
	supplierCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(newMintAmtInt))
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the supplier with address %q", supplierCoin, supplier.Address))

	// Send a portion of the rewards to the DAO
	daoCoin, err := k.sendRewardsToAccount(ctx, k.GetAuthority(), newMintAmtFloat, MintAllocationDAO)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to DAO: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the DAO with address %q", daoCoin, k.GetAuthority()))

	// Send a portion of the rewards to the source owner
	serviceCoins, err := k.sendRewardsToAccount(ctx, service.OwnerAddress, newMintAmtFloat, MintAllocationSourceOwner)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to source owner: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the source owner with address %q", serviceCoins, service.OwnerAddress))

	// Send a portion of the rewards to the block proposer
	proposerAddr := cosmostypes.AccAddress(sdk.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String()
	proposerCoins, err := k.sendRewardsToAccount(ctx, proposerAddr, newMintAmtFloat, MintAllocationProposer)
	if err != nil {
		return tokenomictypes.ErrTokenomicsSendingMintRewards.Wrapf("sending rewards to proposer: %v", err)
	}
	logger.Debug(fmt.Sprintf("sent (%v) newley minted coins from the tokenomics module to the proposer with address %q", proposerCoins, proposerAddr))

	// TODO_MAINNET: Verify that the total distributed coins equals the settlement coins which could happen due to float rounding
	totalDistributedCoins := appCoin.Add(supplierCoin).Add(*daoCoin).Add(*serviceCoins).Add(*proposerCoins)
	if totalDistributedCoins.Amount.BigInt().Cmp(settlementCoins.Amount.BigInt()) != 0 {
		logger.Error(fmt.Sprintf("TODO_MAINNET: The total distributed coins (%v) does not equal the settlement coins (%v)", totalDistributedCoins, settlementCoins.Amount.BigInt()))
	}
	logger.Info(fmt.Sprintf("distributed (%v) coins to the application, supplier, DAO, source owner, and proposer", totalDistributedCoins))

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

	coinsToAccAmt, _ := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(allocation)).Int64()
	coinToAcc := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinsToAccAmt))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, accountAddr, sdk.NewCoins(coinToAcc),
	); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("sent (%v) coins from the tokenomics module to the account with address %q", coinToAcc, addr))

	return &coinToAcc, nil
}

func (k Keeper) handleOverservicedApplication(
	ctx context.Context,
	application *apptypes.Application,
	settlementCoins cosmostypes.Coin,
) (
	newSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger := k.Logger().With("method", "handleOverservicedApplication")
	// over-serviced application
	logger.Warn(fmt.Sprintf(
		"THIS SHOULD NEVER HAPPEN. Application with address %s needs to be charged more than it has staked: %v > %v",
		application.Address,
		settlementCoins,
		application.Stake,
	))

	// TODO_MAINNET(@Olshansk, @RawthiL): The application was over-serviced in the last session so it basically
	// goes "into debt". Need to design a way to handle this when we implement
	// probabilistic proofs and add all the parameter logic. Do we touch the application balance?
	// Do we just let it go into debt? Do we penalize the application? Do we unstake it? Etc...
	// See this document from @red-0ne and @bryanchriswhite for more context: notion.so/buildwithgrove/Off-chain-Application-Stake-Tracking-6a8bebb107db4f7f9dc62cbe7ba555f7
	expectedBurn := settlementCoins

	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr: application.Address,
		ExpectedBurn:    &expectedBurn,
		EffectiveBurn:   application.GetStake(),
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsApplicationOverserviced.Wrapf(
			"application address: %s; expected burn %s; effective burn: %s",
			application.GetAddress(),
			expectedBurn.String(),
			application.GetStake().String(),
		)
	}
	return *application.Stake, nil
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
// shareholders based on the rev share percentage of the service.
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

	var serviceRevShare []*sharedtypes.ServiceRevShare
	for _, svc := range supplier.Services {
		if svc.Service.Id == serviceId {
			serviceRevShare = svc.RevShare
			break
		}
	}

	if serviceRevShare == nil {
		return tokenomicstypes.ErrTokenomicsSupplierRevShareFailed.Wrapf(
			"service %q not found in supplier %v",
			serviceId,
			supplier,
		)
	}

	totalDistributed := int64(0)
	settlementAmountFloat := new(big.Float).SetUint64(amountToDistribute)
	shareAmountMap := make(map[string]int64, len(serviceRevShare))

	for _, revshare := range serviceRevShare {
		// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
		shareFloat := big.NewFloat(float64(revshare.RevSharePercentage) / 100)
		shareAmount, _ := big.NewFloat(0).Mul(settlementAmountFloat, shareFloat).Int64()
		totalDistributed += shareAmount
		shareAmountMap[revshare.Address] = shareAmount
	}

	// Add any remainder due to floating point arithmetic to the first shareholder.
	remainder := amountToDistribute - uint64(totalDistributed)
	shareAmountMap[serviceRevShare[0].Address] += int64(remainder)

	for shareHolderAddress, shareAmount := range shareAmountMap {
		shareAmountCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(shareAmount))
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
