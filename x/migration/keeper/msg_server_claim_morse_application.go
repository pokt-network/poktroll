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

	// Retrieve the shared parameters and calculate various session height params.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())
	previousSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStartHeight-1)

	// Retrieve the staked and unstaked application balance.
	claimedAppStake := morseClaimableAccount.GetApplicationStake()
	claimedUnstakedBalance := morseClaimableAccount.GetUnstakedBalance()

	// Construct unbonded application for cases where it is already or will become unbonded immediately
	// E.g. below min stake, or if unbonding period has already elapsed.
	unbondedApp := &apptypes.Application{
		Address:                 shannonAccAddr.String(),
		Stake:                   &claimedAppStake,
		UnstakeSessionEndHeight: uint64(previousSessionEndHeight),
		// ServiceConfigs:       (intentionally omitted, no service was staked),
	}

	// Construct the base response.
	// It will be modified, as necessary, prior to returning.
	claimMorseAppResponse := &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         morseClaimableAccount.GetMorseSrcAddress(),
		ClaimedBalance:          claimedUnstakedBalance,
		ClaimedApplicationStake: claimedAppStake,
		SessionEndHeight:        sessionEndHeight,
		Application:             unbondedApp,
	}

	// Construct the base application claim event.
	// It will be modified, as necessary, prior to emission.
	morseAppClaimedEvent := &migrationtypes.EventMorseApplicationClaimed{
		MorseSrcAddress:         msg.GetMorseSignerAddress(),
		ClaimedBalance:          claimedUnstakedBalance,
		ClaimedApplicationStake: claimedAppStake,
		SessionEndHeight:        sessionEndHeight,
		Application:             unbondedApp,
	}

	// Construct the application unbonding end event.
	// It will be conditionally emitted if the application unbonding period
	// began on Morse, and ended while waiting to be claimed.
	morseAppUnbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
		Application:        unbondedApp,
		Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: previousSessionEndHeight,
	}

	// Collect events for emission. Events are appended prior to emission to allow
	// for conditional modification prior to emission.
	//
	// Always emitted:
	// - EventMorseApplicationClaimed
	//
	// Conditionally emitted:
	// - EventApplicationUnbondingBegin
	// - EventApplicationUnbondingEnd
	events := make([]cosmostypes.Msg, 0)

	// If unbonding is complete:
	// - No further minting is needed (already minted to shannonDestAddr)
	// - Use a lookup table to estimate block times per network
	// - Calculate unstake session end height using shared parameters and block time
	// - Emit events for Morse application claimed and unbonding started
	if morseClaimableAccount.HasUnbonded() {
		events = append(events, morseAppClaimedEvent)
		events = append(events, morseAppUnbondingEndEvent)
		if err = emitEvents(ctx, events); err != nil {
			return nil, err
		}

		return claimMorseAppResponse, nil
	}

	// Query for any existing application stake prior to staking.
	preClaimAppStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
	foundApp, isFound := k.appKeeper.GetApplication(ctx, shannonAccAddr.String())
	if isFound {
		preClaimAppStake = *foundApp.Stake
	}
	postClaimAppStake := preClaimAppStake.Add(morseClaimableAccount.GetApplicationStake())

	// If the claimed stake is below the minimum, the application is unstaked.
	// Stake and tokens are already minted to shannonDestAddr.
	minStake := k.appKeeper.GetParams(ctx).MinStake
	if postClaimAppStake.Amount.LT(minStake.Amount) {
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

	// Emit the application claim event first.
	// An unbonding begin event MAY follow.
	events = append(events, morseAppClaimedEvent)

	// If the claimed application is still unbonding:
	// - Set the unstake session end height on the application
	// - Emit an unbonding begin event
	if morseClaimableAccount.IsUnbonding() {
		estimatedUnstakeSessionEndHeight, isUnbonded := morseClaimableAccount.GetEstimatedUnbondingEndHeight(ctx)

		// DEV_NOTE: SHOULD NEVER happen, the check above is the same, but in terms of time instead of block height...
		if isUnbonded {
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseApplicationClaim.Wrapf(
					"(SHOULD NEVER HAPPEN) estimated unbonding height is negative (%d)",
					estimatedUnstakeSessionEndHeight,
				).Error(),
			)
		}

		// Set the application's unstake session end height.
		app.UnstakeSessionEndHeight = uint64(estimatedUnstakeSessionEndHeight)
		k.appKeeper.SetApplication(ctx, *app)

		// Emit an event which signals that the claimed Morse application's unbonding
		// period began on Morse and will end on Shannon ad unbonding_end_height
		// (i.e. estimatedUnstakeSessionEndHeight).
		morseAppUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        app,
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: estimatedUnstakeSessionEndHeight,
		}

		// Emit the application unbonding begin event
		// AFTER the application claim event.
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
