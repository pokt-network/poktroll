package keeper

import (
	"context"
	"fmt"

	math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// TODO_TECHDEBT: Retrieve this from the SMT package
	// The number of bytes expected to be contained in the root hash being
	// claimed in order to represent both the digest and the sum.
	smstRootSize = 40
)

// atomic.if this function is not atomic.

// SettleSessionAccounting is responsible for all of the post-session accounting
// necessary to burn, mint or transfer tokens depending on the amount of work
// done. The amount of "work done" complete is dictated by `sum` of `root`.
//
// ASSUMPTION: It is assumed the caller of this function validated the claim
// against a proof BEFORE calling this function.
//
// TODO_BLOCKER(@Olshansk): Is there a way to limit who can call this function?
func (k Keeper) SettleSessionAccounting(ctx context.Context, claim *prooftypes.Claim) error {
	// Parse the context
	logger := k.Logger().With("method", "SettleSessionAccounting")

	if claim == nil {
		logger.Error("received a nil claim")
		return types.ErrTokenomicsClaimNil
	}

	sessionHeader := claim.GetSessionHeader()

	// Make sure the session header is not nil
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
	supplierAddr, err := sdk.AccAddressFromBech32(claim.GetSupplierAddress())
	if err != nil {
		return types.ErrTokenomicsSupplierAddressInvalid
	}

	applicationAddress, err := sdk.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil {
		return types.ErrTokenomicsApplicationAddressInvalid
	}

	// Retrieve the application
	application, found := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !found {
		logger.Error(fmt.Sprintf("application for claim with address %s not found", applicationAddress))
		return types.ErrTokenomicsApplicationNotFound
	}

	root := (smt.MerkleRoot)(claim.GetRootHash())

	// TODO_DISCUSS: This check should be the responsibility of the SMST package
	// since it's used to get compute units from the root hash.
	if len(root) != smstRootSize {
		logger.Error(fmt.Sprintf("received an invalid root hash of size: %d", len(root)))
		return types.ErrTokenomicsRootHashInvalid
	}

	// Retrieve the sum of the root as a proxy into the amount of work done
	claimComputeUnits := root.Sum()

	logger.Info(fmt.Sprintf("About to start settling claim for %d compute units", claimComputeUnits))

	// Calculate the amount of tokens to mint & burn
	settlementAmt := k.getCoinFromComputeUnits(ctx, root)
	settlementAmtCoins := sdk.NewCoins(settlementAmt)

	logger.Info(fmt.Sprintf(
		"%d compute units equate to %s for session %s",
		claimComputeUnits,
		settlementAmt,
		sessionHeader.SessionId,
	))

	// NB: We are doing a mint & burn + transfer, instead of a simple transfer
	// of funds from the supplier to the application in order to enable second
	// order economic effects with more optionality. This could include funds
	// going to pnf, delegators, enabling bonuses/rebates, etc...

	// Mint uPOKT to the supplier module account
	if err := k.bankKeeper.MintCoins(ctx, suppliertypes.ModuleName, settlementAmtCoins); err != nil {
		return types.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"minting %s to the supplier module account: %v",
			settlementAmt,
			err,
		)
	}

	logger.Info(fmt.Sprintf("minted %s in the supplier module", settlementAmt))

	// Sent the minted coins to the supplier
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, suppliertypes.ModuleName, supplierAddr, settlementAmtCoins,
	); err != nil {
		return types.ErrTokenomicsSupplierModuleMintFailed.Wrapf(
			"sending %s to supplier with address %s: %v",
			settlementAmt,
			supplierAddr,
			err,
		)
	}

	logger.Info(fmt.Sprintf("sent %s to supplier with address %s", settlementAmt, supplierAddr))

	// Verify that the application has enough uPOKT to pay for the services it consumed
	if application.Stake.IsLT(settlementAmt) {
		logger.Error(fmt.Sprintf(
			"THIS SHOULD NOT HAPPEN. Application with address %s needs to be charged more than it has staked: %v > %v",
			applicationAddress,
			settlementAmtCoins,
			application.Stake,
		))
		// TODO_BLOCKER(@Olshansk, @RawthiL): The application was over-serviced in the last session so it basically
		// goes "into debt". Need to design a way to handle this when we implement
		// probabilistic proofs and add all the parameter logic. Do we touch the application balance?
		// Do we just let it go into debt? Do we penalize the application? Do we unstake it? Etc...
		settlementAmtCoins = sdk.NewCoins(*application.Stake)
	}

	// Undelegate the amount of coins that need to be burnt from the application stake.
	// Since the application commits a certain amount of stake to the network to be able
	// to pay for relay mining, this stake is taken from the funds "in escrow" rather
	// than its balance.
	if err := k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, apptypes.ModuleName, applicationAddress, settlementAmtCoins); err != nil {
		logger.Error(fmt.Sprintf(
			"THIS SHOULD NOT HAPPEN. Application with address %s needs to be charged more than it has staked: %v > %v",
			applicationAddress,
			settlementAmtCoins,
			application.Stake,
		))
	}

	// Send coins from the application to the application module account
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, applicationAddress, apptypes.ModuleName, settlementAmtCoins,
	); err != nil {
		return types.ErrTokenomicsApplicationModuleFeeFailed
	}

	logger.Info(fmt.Sprintf("took %s from application with address %s", settlementAmt, applicationAddress))

	// Burn uPOKT from the application module account
	if err := k.bankKeeper.BurnCoins(ctx, apptypes.ModuleName, settlementAmtCoins); err != nil {
		return types.ErrTokenomicsApplicationModuleBurn
	}

	logger.Info(fmt.Sprintf("burned %s in the application module", settlementAmt))

	// Update the application's on-chain stake
	newAppStake := (*application.Stake).Sub(settlementAmt)
	application.Stake = &newAppStake
	k.applicationKeeper.SetApplication(ctx, application)

	return nil
}

func (k Keeper) getCoinFromComputeUnits(ctx context.Context, root smt.MerkleRoot) sdk.Coin {
	// Retrieve the existing tokenomics params
	params := k.GetParams(ctx)

	upokt := math.NewInt(int64(root.Sum() * params.ComputeUnitsToTokensMultiplier))
	return sdk.NewCoin("upokt", upokt)
}
