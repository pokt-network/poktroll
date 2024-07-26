package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type TokenLogicModule int

// References:
// - https://docs.pokt.network/pokt-protocol/the-shannon-upgrade/proposed-tokenomics/token-logic-modules
// - https://github.com/pokt-network/shannon-tokenomics-static-tests

const (
	// TODO_BLOCKER: All of these need to be governance params
	MintAllocationDAO         = 0.1
	MintAllocationProposer    = 0.05
	MintAllocationSupplier    = 0.7
	MintAllocationSourceOwner = 0.15

	RelayBurnEqualsMint TokenLogicModule = iota
	// SourceBoost
	// SupplierBoost
	// ComputeUnits
	// SurgePricing
	// ???
)

func init() {
	// Ensure 100% of minted rewards are allocated
	if 1.0 != MintAllocationDAO+MintAllocationProposer+MintAllocationSupplier+MintAllocationSourceOwner {
		assert.Fail("total mint allocation must equal 1.0")
	}
}

func (tlm TokenLogicModule) String() string {
	return [...]string{"RelayBurnEqualsMint"}[tlm]
}

func (tlm TokenLogicModule) EnumIndex() int {
	return int(tlm)
}

// TODO_ITERATE: Need to store which TLM is available on a per-service basis
func (k Keeper) GetTLMsEnabledForService(serviceId string) bool {
	return true
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
func (k Keeper) ProcessTokenLogicModules(
	ctx context.Context,
	claim *prooftypes.Claim,
	computeUnits_ThisClaim uint64,
	computeUnits_LastSessionAllNetworkClaims uint64,
	sessionHeader *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	// TODO_IN_THIS_PR: Consider refactoring
	// - session.NumComputeUnits (usage) * service.ComputeUnitsPerRelay (source owner) * global.ComputeUnitsToTokensMultiplier (DAO)
	settlementCoins cosmostypes.Coin, // Core TLM -> ideal number of tokens to settle
	// computeUnits needed for boost calculations
	// settlementCoins
	// Need # of compute units
) error {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// allSuppliers := k.supplierKeeper.GetAllSuppliers(ctx)

	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, sessionHeader.Service.Id)
	if !found {
		return tokenomicstypes.ErrTokenomicsMissingRelayMiningDifficulty.Wrapf("relay mining difficulty not found for service %s", service.Id)
	}
	logger.Debug(fmt.Sprintf("TODO: Use relayMiningDifficulty to calculate the number of relays: %v", relayMiningDifficulty))

	// Core TLM: Execute the Burn=Mint TLM
	if err := k.TLMRelayBurnEqualsMint(ctx, sessionHeader, application, supplier, settlementCoins); err != nil {
		return err
	}

	// 1. ComputeUnits = 100; 0.01% of the network
	// 2. ComputeUnits = 100; 20% of the network
	// 3. ComputeUnits = 100; 0.00000000000001% of the network
	// We over-mint for every service, for as long as we need to.

	// if computeUnits_ThisClaim / computeUnits_LastSessionAllNetworkClaims < ???
	//

	k.DAOBoost(ctx, computeUnits_LastSessionAllNetworkClaims)

	// Work being done by network; TotalComputeUnitsOnNetwork
	// Cost of work being done by network; POKT_TO_USD and ComputeUnitsPerPOKT
	// Cost of all suppliers on network; CostOfNetwork -> 20K USD per day
	// Equilibrium(TotalComputeUnitsOnNetwork, POKT_TO_USD, ComputeUnitsPerPOKT, CostOfNetwork)

	// 1 boost -> 1 actor

	// Boost TLMs
	//
	// Suppliers Boost
	// Validator Boost
	// DAO Boost
	// Sources Boost

	return nil
}

// TLMRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
func (k Keeper) TLMRelayBurnEqualsMint(
	ctx context.Context,
	sessionHeader *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
) error {
	logger := k.Logger().With("method", "TLMRelayBurnEqualsMint")

	// NB: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	supplierAddr := cosmostypes.AccAddress(supplier.Address)

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
		// Make the settlement amount the maximum stake that the application has remaining.
		settlementCoins = *application.GetStake()

		applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
			ApplicationAddr: application.Address,
			ExpectedBurn:    &expectedBurn,
			EffectiveBurn:   application.GetStake(),
		}
		eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
		if err := eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
			return tokenomicstypes.ErrTokenomicsApplicationOverserviced.Wrapf(
				"application address: %s; expected burn %s; effective burn: %s",
				application.GetAddress(),
				expectedBurn.String(),
				application.GetStake().String(),
			)
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
	sessionHeader *sessiontypes.SessionHeader,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coin,
) error {
	logger := k.Logger().With("method", "distributeMintedRewards").With("session", sessionHeader.SessionId)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	settlementAmtFloat := new(big.Float).SetUint64(settlementCoins.Amount.Uint64())

	// Send a portion of the rewards to the supplier
	supplierAddr := cosmostypes.AccAddress(supplier.Address)
	coinsToSupplierAmt, _ := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintAllocationSupplier)).Int64()
	coinsToSupplier := sdk.NewCoins(cosmostypes.NewCoin(settlementCoins.Denom, math.NewInt(coinsToSupplierAmt)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, supplierAddr, coinsToSupplier,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierRewardFailed.Wrapf(
			"error sending %s tokens to supplier %s: %v",
			settlementCoins,
			supplierAddr,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent %s tokens to supplier %s", settlementCoins, supplierAddr))

	// Send a portion of the rewards to the DAO
	// TODO_IN_THIS_PR: Ensure that this is the DAO address
	daoAddress := k.GetAuthority()
	coinsToDAOAmt, _ := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintAllocationDAO)).Int64()
	coinsToDAO := sdk.NewCoins(cosmostypes.NewCoin(settlementCoins.Denom, math.NewInt(coinsToDAOAmt)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, daoAddress, coinsToDAO,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsDAORewardFailed.Wrapf(
			"error sending %s tokens to DAO %s: %v",
			settlementCoins,
			daoAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent %s tokens to DAO %s", settlementCoins, daoAddress))

	// Send a portion of the rewards to the block proposer
	proposerAddress := sdkCtx.BlockHeader().ProposerAddress
	coinsToProposerAmt, _ := new(big.Float).Mul(settlementAmtFloat, big.NewFloat(MintAllocationProposer)).Int64()
	coinsToProposer := sdk.NewCoins(cosmostypes.NewCoin(settlementCoins.Denom, math.NewInt(coinsToProposerAmt)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, proposerAddress, coinsToProposer,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsProposerRewardFailed.Wrapf(
			"error sending %s tokens to proposer %s: %v",
			settlementCoins,
			proposerAddress,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent %s tokens to proposer %s", settlementCoins, proposerAddress))

	// Send a portion of the rewards to the source owner
	sourceOwner

	return nil
}

// Trying to achieve a boost value for the CUTTM
func (k Keeper) DAOBoost() {
	/*
	   ### DAO Boost
	   aux_boost = dict()
	   aux_boost["name"] = "DAO Boost - CU Based"
	   aux_boost["recipient"] = "DAO"  # Who receives the minting
	   aux_boost["conditions"] = list()  # List of conditions to meet for TLM execution
	   aux_boost["conditions"].append(
	       {
	           "metric": "total_cus",  # Total CUs in the network
	           "low_thrs": 0,
	           "high_thrs": 2500 * 1e9,  # CU/day
	       }
	   )
	   aux_boost["minting_func"] = (
	       boost_cuttm_f_CUs_nonlinear  # A function that accepts as input the services state, the network macro state and the parameters below and return the amount to mint per service
	   )
	   aux_boost["parameters"] = (
	       {  # A structure containing all parameters needed by this module
	           "start": {
	               "x": 250 * 1e9,  # ComputeUnits/day
	               "y": 5e-9,  # USD/ComputeUnits
	           },
	           "end": {
	               "x": 2500 * 1e9,  # ComputeUnits/day
	               "y": 0,  # USD/ComputeUnits
	           },
	           "variables": {
	               "x": "total_cus",  # Control metric for this TLM
	               "y": "CUTTM",  # Target parameter for this TLM
	           },
	           "budget": {  # Can be a fixed number of tokens [POKT] or a percentage of total supply (annualized) [annual_supply_growth]
	               "type": "annual_supply_growth",
	               "value": 0.5,
	           },
	       }
	   )
	*/

}

// 1. Declare non-linear function: y = a * x + b / x
// 2. Calculate a  & b based on hard-coded / DAO-controlled / off-chain values
// 3. For each claim being settled:
// 	-> Find boost (y) based on CUTTM (x) for that claim
//  -> Apply boost N times in total where N is # of claims being settled
