package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ClaimMorseApplication performs the following steps, given msg is valid and a
// MorseClaimableAccount exists for the given morse_src_address:
//   - Mint and transfer all tokens (unstaked balance plus application stake) of the
//     MorseClaimableAccount to the shannonDestAddress.
//   - Mark the MorseClaimableAccount as claimed (i.e. adding the shannon_dest_address
//     and claimed_at_height).
//   - Stake an application for the amount specified in the MorseClaimableAccount,
//     and the service specified in the msg.
func (k msgServer) ClaimMorseApplication(ctx context.Context, msg *migrationtypes.MsgClaimMorseApplication) (*migrationtypes.MsgClaimMorseApplicationResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseApplication")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonDestAddress is validated in MsgClaimMorseApplication#ValidateBasic().
	shannonAccAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonDestAddress)

	// Retrieve the MorseClaimableAccount for the given morseSrcAddress.
	morseClaimableAccount, err := k.CheckMorseClaimableApplicationAccount(ctx, msg.GetMorseSignerAddress())
	if err != nil {
		return nil, err
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	// The application stake is subsequently escrowed from the shannonDestAddress account balance.
	if err = k.MintClaimedMorseTokens(ctx, shannonAccAddr, morseClaimableAccount.TotalTokens()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonAccAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(sdkCtx, *morseClaimableAccount)

	claimedAppStake := morseClaimableAccount.GetApplicationStake()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	claimedUnstakedBalance := morseClaimableAccount.GetUnstakedBalance()

	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())
	previousSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStartHeight-1)

	// Query for any existing application stake prior to staking.
	preClaimAppStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
	foundApp, isFound := k.appKeeper.GetApplication(ctx, shannonAccAddr.String())
	if isFound {
		preClaimAppStake = *foundApp.Stake
	}
	postClaimAppStake := preClaimAppStake.Add(morseClaimableAccount.GetApplicationStake())

	// Construct unbonded application for cases where it is already or will become unbonded
	// immediately (i.e. below min stake, or if unbonding period has already elpased).
	unbondedApp := &apptypes.Application{
		Address:                 shannonAccAddr.String(),
		Stake:                   &postClaimAppStake,
		UnstakeSessionEndHeight: uint64(previousSessionEndHeight),
		// ServiceConfigs:       (intentionally omitted, no service was staked),
	}

	// Construct the base response. It will be modified, as necessary, prior to returning.
	claimMorseAppResponse := &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         morseClaimableAccount.GetMorseSrcAddress(),
		ClaimedBalance:          claimedUnstakedBalance,
		ClaimedApplicationStake: claimedAppStake,
		SessionEndHeight:        sessionEndHeight,
		Application:             unbondedApp,
	}

	morseAppClaimedEvent := &migrationtypes.EventMorseApplicationClaimed{
		MorseSrcAddress:         msg.GetMorseSignerAddress(),
		ClaimedBalance:          claimedUnstakedBalance,
		ClaimedApplicationStake: claimedAppStake,
		SessionEndHeight:        sessionEndHeight,
		Application:             unbondedApp,
	}

	// Conditionally emit an event which signals that the claimed Morse application's unbonding
	// period began on Morse, and ended while waiting to be claimed.
	morseAppUnbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
		Application:        unbondedApp,
		Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: sessionEndHeight,
	}

	// Collect events for emission. Events are appended prior to emission to allow
	// for conditional modification prior to emission.
	//
	// Always emitted:
	// - EventMorseApplicationClaimed
	// Conditionally emitted:
	// - EventApplicationUnbondingBegin
	// - EventApplicationUnbondingEnd
	events := make([]cosmostypes.Msg, 0)

	// If the claimed application stake is less than the minimum stake, the application is immediately unstaked.
	// - All stake and unstaked tokens have already been minted to shannonDestAddr account
	minStake := k.appKeeper.GetParams(ctx).MinStake
	if postClaimAppStake.Amount.LT(minStake.Amount) {
		// Emit the supplier claim event first, then the unbonding end event.
		events = append(events, morseAppClaimedEvent)
		events = append(events, morseAppUnbondingEndEvent)
		if err = emitEvents(ctx, events); err != nil {
			return nil, err
		}

		return claimMorseAppResponse, nil
	}

	// This condition checks whether the MorseClaimableAccount has completed unbonding.
	// If unbonding is complete, the following steps occur:
	// - Since minting to shannonDestAddr has already occurred, no additional action is necessary.
	// - A lookup table is used to estimate block times for each network.
	// - Shared parameters and block time are utilized to calculate the unstake session end height.
	// - Emit an event which signals that the Morse application was claimed.
	// - Emit an event which signals that the Morse application began unbonding.
	if morseClaimableAccount.HasUnbonded() {
		// Emit the supplier claim event first, then the unbonding end event.
		events = append(events, morseAppClaimedEvent)
		events = append(events, morseAppUnbondingEndEvent)
		if err = emitEvents(ctx, events); err != nil {
			return nil, err
		}

		return claimMorseAppResponse, nil
	}

	// Stake (or update) the application.
	msgStakeApp := apptypes.NewMsgStakeApplication(
		shannonAccAddr.String(),
		preClaimAppStake.Add(morseClaimableAccount.GetApplicationStake()),
		[]*sharedtypes.ApplicationServiceConfig{msg.ServiceConfig},
	)

	app, err := k.appKeeper.StakeApplication(ctx, logger, msgStakeApp)
	if err != nil {
		// DEV_NOTE: If the error is non-nil, StakeApplication SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	// Update the application claim response.
	claimMorseAppResponse.ClaimedBalance = morseClaimableAccount.GetUnstakedBalance()
	claimMorseAppResponse.ClaimedApplicationStake = morseClaimableAccount.GetApplicationStake()
	claimMorseAppResponse.Application = app

	// Update the application claim event.
	morseAppClaimedEvent.ClaimedBalance = morseClaimableAccount.GetUnstakedBalance()
	morseAppClaimedEvent.ClaimedApplicationStake = morseClaimableAccount.GetApplicationStake()
	morseAppClaimedEvent.Application = app

	// Emit the application claim event first, an unbonding begin event MAY follow.
	events = append(events, morseAppClaimedEvent)

	// If the claimed application is still unbonding:
	// - Set the unstake session end height on the application
	// - Emit an unbonding begin event
	if morseClaimableAccount.IsUnbonding() {
		estimatedUnstakeSessionEndHeight := morseClaimableAccount.GetEstimatedUnbondingEndHeight(ctx)

		// DEV_NOTE: SHOULD NEVER happen, the check above is the same, but in terms of time instead of block height...
		if estimatedUnstakeSessionEndHeight < 0 {
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseApplicationClaim.Wrapf(
					"estimated unbonding height is negative (%d)",
					estimatedUnstakeSessionEndHeight,
				).Error(),
			)
		}

		// Set the application's unstake session end height.
		app.UnstakeSessionEndHeight = uint64(estimatedUnstakeSessionEndHeight)
		k.appKeeper.SetApplication(ctx, *app)

		// Emit an event which signals that the claimed Morse supplier's unbonding
		// period began on Morse and will end on Shannon ad unbonding_end_height
		// (i.e. estimatedUnstakeSessionEndHeight).
		morseAppUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        app,
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: estimatedUnstakeSessionEndHeight,
		}

		// Emit the supplier unbonding begin event
		// AFTER the supplier claim event.
		events = append(events, morseAppUnbondingBeginEvent)
	}

	if err = emitEvents(ctx, events); err != nil {
		return nil, err
	}

	// Return the response.
	return claimMorseAppResponse, nil
}

// CheckMorseClaimableApplicationAccount attempts to retrieve a MorseClaimableAccount for the given morseSrcAddress.
// It ensures the MorseClaimableAccount meets the following criteria:
// - It exists on-chain
// - It not already been claimed
// - It has a non-zero application stake
// - It has zero supplier stake
// If the MorseClaimableAccount does not exist, it returns an error.
// If the MorseClaimableAccount has already been claimed, any waived gas fees are charged and an error is returned.
func (k msgServer) CheckMorseClaimableApplicationAccount(
	ctx context.Context,
	morseSrcAddress string,
) (*migrationtypes.MorseClaimableAccount, error) {
	// Ensure that a MorseClaimableAccount exists and has not been claimed for the given morseSrcAddress.
	morseClaimableAccount, err := k.CheckMorseClaimableAccount(ctx, morseSrcAddress, migrationtypes.ErrMorseApplicationClaim)
	if err != nil {
		return nil, err
	}

	// ONLY allow claiming as a supplier account if the MorseClaimableAccount
	// WAS staked as a supplier AND NOT as an application. A claim of staked POKT
	// from Morse to Shannon SHOULD NOT allow applications or suppliers to bypass
	// the onchain unbonding period.
	if !morseClaimableAccount.SupplierStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"Morse account %q is staked as a supplier, please use `pocketd tx migration claim-supplier` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	if !morseClaimableAccount.ApplicationStake.IsPositive() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"Morse account %q is not staked as a application or supplier, please use `pocketd tx migration claim-account` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	return morseClaimableAccount, nil
}
