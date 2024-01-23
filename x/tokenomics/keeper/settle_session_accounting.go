package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettleSessionAccounting implements TokenomicsKeeper#SettleSessionAccounting
// It is ASSUMED that the caller of this function validated the claim
// against a proof BEFORE calling this function.
//
// TODO_BLOCKER(@Olshansk): Is there a way to limit who can call this function?
// TODO_BLOCKER: This is just a first naive implementation of the business logic.
func (k TokenomicsKeeper) SettleSessionAccounting(
	goCtx context.Context,
	claim *suppliertypes.Claim,
) error {
	// Parse the context
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "SettleSessionAccounting")

	if claim == nil {
		logger.Error("received a nil claim")
		return types.ErrTokenomicsClaimNil
	}

	// Make sure the session header is not nil
	sessionHeader := claim.SessionHeader
	if sessionHeader == nil {
		logger.Error("received a nil session header")
		return types.ErrTokenomicsSessionHeaderNil
	}

	// Validate the session header
	if err := sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return types.ErrTokenomicsSessionHeaderInvalid
	}

	// Decompose the claim into its constituent parts for readability
	supplierAddress, err := sdk.AccAddressFromBech32(claim.SupplierAddress)
	if err != nil {
		return types.ErrTokenomicsSupplierAddressInvalid
	}
	applicationAddress, err := sdk.AccAddressFromBech32(claim.SessionHeader.ApplicationAddress)
	if err != nil {
		return types.ErrTokenomicsApplicationAddressInvalid
	}
	root := (smt.MerkleRoot)(claim.RootHash)

	// Retrieve the application
	application, found := k.appKeeper.GetApplication(ctx, applicationAddress.String())
	if !found {
		logger.Error(fmt.Sprintf("application for claim with address %s not found", applicationAddress))
		return types.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the sum of the root as a proxy into the amount of work done
	claimComputeUnits := root.Sum()

	logger.Info(fmt.Sprintf("About to start settling claim for %d compute units", claimComputeUnits))

	// Retrieve the existing tokenomics params
	params := k.GetParams(ctx)

	// Calculate the amount of tokens to mint & burn
	upokt := sdk.NewInt(int64(claimComputeUnits * params.ComputeUnitsToTokensMultiplier))
	upoktCoin := sdk.NewCoin("upokt", upokt)
	upoktCoins := sdk.NewCoins(upoktCoin)

	logger.Info(fmt.Sprintf("%d compute units equate to %d uPOKT for session %s", claimComputeUnits, upokt, sessionHeader.SessionId))

	// NB: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint uPOKT to the supplier module account
	if err := k.bankKeeper.MintCoins(ctx, suppliertypes.ModuleName, upoktCoins); err != nil {
		return types.ErrTokenomicsApplicationModuleFeeFailed
	}

	logger.Info(fmt.Sprintf("minted %d uPOKT in the supplier module", upokt))

	// Sent the minted coins to the supplier
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, supplierAddress, upoktCoins,
	); err != nil {
		return types.ErrTokenomicsApplicationModuleFeeFailed
	}

	logger.Info(fmt.Sprintf("sent %d uPOKT to supplier with address %s", upokt, supplierAddress))

	// Verify that the application has enough uPOKT to pay for the services it consumed
	if application.Stake.IsLT(upoktCoin) {
		logger.Error(fmt.Sprintf("THIS SHOULD NOT HAPPEN. Application with address %s needs to be charged more than it has staked: %v > %v", applicationAddress, upoktCoins, application.Stake))
		// TODO_BLOCKER(@Olshansk, @RawthiL): The application was over-serviced in the last session so it basically
		// goes "into debt". Need to design a way to handle this when we implement
		// probabilistic proofs and add all the parameter logic. Do we touch the application balance?
		// Do we just let it go into debt? Do we penalize the application? Do we unstake it? Etc...
		upoktCoins = sdk.NewCoins(*application.Stake)
	}

	// Undelegate the amount of coins that need to be burnt
	if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, apptypes.ModuleName, applicationAddress, upoktCoins); err != nil {
		logger.Error(fmt.Sprintf("THIS SHOULD NOT HAPPEN. Application with address %s needs to be charged more than it has staked: %v > %v", applicationAddress, upoktCoins, application.Stake))

	}

	// Send coins from the application to the application module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, applicationAddress, apptypes.ModuleName, upoktCoins,
	); err != nil {
		return types.ErrTokenomicsApplicationModuleFeeFailed
	}

	logger.Info(fmt.Sprintf("took %d uPOKT from application with address %s", upokt, applicationAddress))

	// Burn uPOKT from the application module account
	if err := k.bankKeeper.BurnCoins(ctx, apptypes.ModuleName, upoktCoins); err != nil {
		return types.ErrTokenomicsApplicationModuleBurn
	}

	logger.Info(fmt.Sprintf("burned %d uPOKT in the application module", upokt))

	// Update the application's on-chain stake
	newAppStake := (*application.Stake).Sub(upoktCoin)
	application.Stake = &newAppStake
	k.appKeeper.SetApplication(ctx, application)

	return nil
}
