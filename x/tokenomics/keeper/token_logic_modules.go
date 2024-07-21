package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type TokenLogicModule int

// References:
// - https://docs.pokt.network/pokt-protocol/the-shannon-upgrade/proposed-tokenomics/token-logic-modules
// - https://github.com/pokt-network/shannon-tokenomics-static-tests

const (
	RelayBurnEqualsMint TokenLogicModule = iota
	// SourceBoost
	// SupplierBoost
	// ComputeUnits
	// SurgePricing
	// ???
)

func (tlm TokenLogicModule) String() string {
	return [...]string{"RelayBurnEqualsMint"}[tlm]
}

func (tlm TokenLogicModule) EnumIndex() int {
	return int(tlm)
}

func (k Keeper) GetTLMsEnabledForService(serviceId string) bool {
	return true
}

func (k Keeper) GetTLMsEnabledForAllServices() bool {
	return true
}

// Enable on a per service basis
// Need to store which TLM is availalbe for each service

// TLMRelayBurnEqualsMint processes the business logic for the RelayBurnEqualsMint TLM.
func (k Keeper) TLMRelayBurnEqualsMint(
	ctx context.Context,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	settlementCoins cosmostypes.Coins,
) error {
	logger := k.Logger().With("method", "TLMRelayBurnEqualsMint")

	supplierAddr := cosmostypes.AccAddress(supplier.Address)

	// NB: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint new uPOKT to the supplier module account.
	// These funds will be transferred to the supplier below.
	if err := k.bankKeeper.MintCoins(
		ctx, suppliertypes.ModuleName, settlementCoins,
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
		ctx, suppliertypes.ModuleName, supplierAddr, settlementCoins,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"sending %s to supplier with address %s: %v",
			settlementCoins,
			supplierAddr,
			err,
		)
	}
	logger.Info(fmt.Sprintf("sent %s from the supplier module to the supplier account with address %q", settlementCoin, supplierAddr))

	// Verify that the application has enough uPOKT to pay for the services it consumed
	if application.GetStake().IsLT(settlementCoins[]) {
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
		expectedBurn := settlementCoin
		// Make the settlement amount the maximum stake that the application has remaining.
		settlementCoin = *application.GetStake()
		settlementCoins = cosmostypes.NewCoins(settlementCoin)

		applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
			ApplicationAddr: applicationAddress.String(),
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
		ctx, apptypes.ModuleName, settlementCoins,
	); err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationModuleBurn.Wrapf("burning %s from the application module account: %v", settlementCoin, err)
	}
	logger.Info(fmt.Sprintf("burned %s from the application module account", settlementCoin))

	// Update the application's on-chain stake
	newAppStake, err := application.Stake.SafeSub(settlementCoin)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf("application %q stake cannot be reduced to a negative amount %v", applicationAddress, newAppStake)
	}
	application.Stake = &newAppStake

}
