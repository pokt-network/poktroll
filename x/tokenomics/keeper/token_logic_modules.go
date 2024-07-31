package keeper

import (
	"context"
	"encoding/hex"
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
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// References:
// - https://docs.pokt.network/pokt-protocol/the-shannon-upgrade/proposed-tokenomics/token-logic-modules
// - https://github.com/pokt-network/shannon-tokenomics-static-tests

const (
	// TODO_UPNEXT(@olshansk): Make all of the governance params
	MintAllocationDAO         = 0.1
	MintAllocationProposer    = 0.05
	MintAllocationSupplier    = 0.7
	MintAllocationSourceOwner = 0.15
	MintAllocationApplication = 0.0
)

type TokenLogicModule int

const (
	RelayBurnEqualsMint TokenLogicModule = iota
	// TODO_UPNEXT(@olshansk): Add more TLMs
	// DAOBoost
	// SourceBoost
	// SupplierBoost
	// SurgePricing
	// ...
)

func (tlm TokenLogicModule) String() string {
	return [...]string{"RelayBurnEqualsMint"}[tlm]
}

func (tlm TokenLogicModule) EnumIndex() int {
	return int(tlm)
}

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner+MintAllocationApplication {
		panic("mint allocation percentages do not add to 1.0")
	}
}

// Inputs:
// - Global view of the network
//	- total_supply (all available POKT)
//  - total_relays (all relays)
//  - Per service: ServiceComputeUnits = relayMiningDifficulty.EMA * service.ComputeUnits
//      -
//  - All services: sum(ServiceComputeUnits) over all services
// 		- Not needed on MainNet launch if no normalization

// Tradeoffs of Global Normalization vs Responsiveness of rewards on new services; on Launch
// -> If there's no normalization -> takes more time to incentivize new services

// "daily relays":512183851.0
// "supplier nodes":198390.0

// Service is. Relays are real. Compute units are controlled by service owner.

// TLMRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
//
// ProcessTokenLogicModules is called for each claim.

// type TokenLogicModule interface {
// 	ProcessTokens(ctx context.Context, claim *prooftypes.Claim) error
// }

// 1. ComputeUnits = 100; 0.01% of the network
// 2. ComputeUnits = 100; 20% of the network
// 3. ComputeUnits = 100; 0.00000000000001% of the network
// We over-mint for every service, for as long as we need to.

// if computeUnits_ThisClaim / computeUnits_LastSessionAllNetworkClaims < ???
//

// Work being done by network; TotalComputeUnitsOnNetwork
// Cost of work being done by network; POKT_TO_USD and ComputeUnitsPerPOKT
// Cost of all suppliers on network; CostOfNetwork -> 20K USD per day
// Equilibrium(TotalComputeUnitsOnNetwork, POKT_TO_USD, ComputeUnitsPerPOKT, CostOfNetwork)

// 1. Declare non-linear function: y = a * x + b / x
// 2. Calculate a  & b based on hard-coded / DAO-controlled / off-chain values
// 3. For each claim being settled:
// 	-> Find boost (y) based on CUTTM (x) for that claim
//  -> Apply boost N times in total where N is # of claims being settled

// ProcessTokenLogicModules is responsible for calling all of the token logic
// modules (i.e. post session claim accounting) necessary to burn, min or transfer
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

	// Retrieve the sum of the root hash to determine the amount of work (compute units) done
	claimComputeUnits, err := root.Sum()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("%v", err)
	}

	// Retrieve the on-chain staked application record
	application, foundApplication := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !foundApplication {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the on-chain staked supplier record
	supplier, foundSupplier := k.supplierKeeper.GetSupplier(ctx, supplierAddr.String())
	if !foundSupplier {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	service, ok := k.serviceKeeper.GetService(ctx, sessionHeader.Service.Id)
	if !ok {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", sessionHeader.Service.Id)
	}

	// Determine the total number of tokens that'll be used for settling the session.
	// When the network achieves equilibrium, this will be the mint & burn.
	settlementCoin, err = k.computeUnitsToCoin(ctx, claimComputeUnits, &service)
	if err != nil {
		return err
	}

	// Retrieving the relay mining difficulty for the service at hand
	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		return tokenomicstypes.ErrTokenomicsMissingRelayMiningDifficulty.Wrapf("relay mining difficulty not found for service %s", service.Id)
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"compute_units", claimComputeUnits,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", sessionHeader.GetSessionId(),
		"supplier", supplierAddr,
		"application", applicationAddress,
	)
	logger.Info(fmt.Sprintf("About to start claiming (%d) compute units equaling to (%s) coins for session (%s)", claimComputeUnits, settlementCoin, sessionHeader.SessionId))

	// Core TLM: Execute the Burn=Mint TLM
	if err := k.TLMRelayBurnEqualsMint(ctx, &service, &relayMiningDifficulty, &application, &supplier, settlementCoin); err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Finished executing TLM %q", RelayBurnEqualsMint))

	// TODO: Add other boosts

	// Update the application's on-chain record
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", applicationAddress))

	// Update the suppliers's on-chain record
	k.supplierKeeper.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("updated on-chain supplier record with address %q", supplierAddr))

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return nil
}

// TLMRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
func (k Keeper) TLMRelayBurnEqualsMint(
	ctx context.Context,
	service *sharedtypes.Service,
	relayMiningDifficulty *types.RelayMiningDifficulty,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
) error {
	logger := k.Logger().With("method", "TLMRelayBurnEqualsMint")

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
	if err := k.bankKeeper.MintCoins(
		ctx, suppliertypes.ModuleName, sdk.NewCoins(settlementCoins),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"minting %s to the supplier module account: %v",
			settlementCoins,
			err,
		)
	}
	logger.Info(fmt.Sprintf("minted %s in the supplier module", settlementCoins))

	// Send the newley minted uPOKT from the supplier module account
	// to the supplier's account.
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, supplierAddr, sdk.NewCoins(settlementCoins),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"sending %s to supplier with address %s: %v",
			settlementCoins,
			supplierAddr,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent %v from the supplier module to the supplier account with address %q", settlementCoins, supplierAddr))

	// Verify that the application has enough uPOKT to pay for the services it consumed
	if application.GetStake().IsLT(settlementCoins) {
		settlementCoins, err = k.handleOverservicedApplication(ctx, application, settlementCoins)
		if err != nil {
			return err
		}
	}

	// Burn uPOKT from the application module account which was held in escrow
	// on behalf of the application account.
	if err := k.bankKeeper.BurnCoins(
		ctx, apptypes.ModuleName, sdk.NewCoins(settlementCoins),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationModuleBurn.Wrapf("burning %s from the application module account: %v", settlementCoins, err)
	}
	logger.Info(fmt.Sprintf("burned %s from the application module account", settlementCoins))

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(settlementCoins)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", application.Address, newAppStake)
	}
	application.Stake = &newAppStake

	return nil
}

func (k Keeper) distributeMintedRewards(
	ctx context.Context,
	service *sharedtypes.Service,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
) error {
	settlementAmtFloat := new(big.Float).SetUint64(settlementCoins.Amount.Uint64())

	// Send a portion of the rewards to the application
	if err := k.sendRewardsToAccount(ctx, application.Address, settlementAmtFloat, MintAllocationApplication); err != nil {
		return err
	}

	// Send a portion of the rewards to the supplier
	if err := k.sendRewardsToAccount(ctx, supplier.Address, settlementAmtFloat, MintAllocationSupplier); err != nil {
		return err
	}

	// Send a portion of the rewards to the DAO
	if err := k.sendRewardsToAccount(ctx, k.GetAuthority(), settlementAmtFloat, MintAllocationDAO); err != nil {
		return err
	}

	// Send a portion of the rewards to the source owner
	if err := k.sendRewardsToAccount(ctx, service.OwnerAddress, settlementAmtFloat, MintAllocationSourceOwner); err != nil {
		return err
	}

	// Send a portion of the rewards to the block proposer
	proposerAddr, err := k.getProposerAddr(ctx)
	if err != nil {
		return err
	}
	if err := k.sendRewardsToAccount(ctx, proposerAddr, settlementAmtFloat, MintAllocationProposer); err != nil {
		return err
	}

	return nil
}

func (k Keeper) sendRewardsToAccount(ctx context.Context, addr string, settlementAmtFloat *big.Float, allocation float64) error {
	accountAddr, err := cosmostypes.AccAddressFromBech32(addr)
	if err != nil {
		return err
	}
	coinsToAccAmt, _ := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(allocation)).Int64()
	coinsToAcc := sdk.NewCoins(cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinsToAccAmt)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, accountAddr, coinsToAcc,
	); err != nil {
		return err
	}
	return nil
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

// computeUnitsToCoin calculates the amount of uPOKT to mint based on the number of compute units.
func (k Keeper) computeUnitsToCoin(ctx context.Context, numComputeUnits uint64, service *sharedtypes.Service) (cosmostypes.Coin, error) {
	computeUnitsPerRelay := service.ComputeUnitsPerRelay
	computeUnitsToTokensMultiplier := k.GetParams(ctx).ComputeUnitsToTokensMultiplier
	upoktAmount := math.NewInt(int64(numComputeUnits * computeUnitsPerRelay * computeUnitsToTokensMultiplier))
	if upoktAmount.IsNegative() {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return cosmostypes.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}

func (k Keeper) getProposerAddr(ctx context.Context) (string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	proposerAddressBz, err := hex.DecodeString(string(sdkCtx.BlockHeader().ProposerAddress))
	if err != nil {
		return "", err
	}
	return sdk.AccAddress(proposerAddressBz).String(), nil

}
