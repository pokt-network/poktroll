package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	cosmoserrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) ClaimMorseApplication(ctx context.Context, msg *migrationtypes.MsgClaimMorseApplication) (*migrationtypes.MsgClaimMorseApplicationResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseApplication")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	shannonAccAddr, err := cosmostypes.AccAddressFromBech32(msg.ShannonDestAddress)
	// DEV_NOTE: This SHOULD NEVER happen as the shannonDestAddress is validated
	// in MsgClaimMorseApplication#ValidateBasic().
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			cosmoserrors.ErrInvalidAddress.Wrapf(
				"failed to parse shannon destination address (%s): %s",
				msg.ShannonDestAddress, err,
			).Error(),
		)
	}

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				msg.MorseSrcAddress,
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// Default to the stake amount recorded in the MorseClaimableAccount.
	if msg.Stake == nil {
		msg.Stake = &morseClaimableAccount.ApplicationStake
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
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	msgStakeApp := apptypes.NewMsgStakeApplication(
		shannonAccAddr.String(),
		*msg.Stake,
		[]*sharedtypes.ApplicationServiceConfig{msg.ServiceConfig},
	)

	initialAppStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	foundApp, isFound := k.appKeeper.GetApplication(ctx, shannonAccAddr.String())
	if isFound {
		initialAppStake = *foundApp.Stake
	}

	app, err := k.appKeeper.StakeApplication(ctx, logger, msgStakeApp)
	if err != nil {
		// DEV_NOTE: StakeApplication SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	// DEV_NOTE: It is safe to use Coin#Sub() here because #StakeApplication() would error otherwise.
	claimedAppStake := app.Stake.Sub(initialAppStake)
	claimedUnstakedTokens := morseClaimableAccount.TotalTokens().Sub(claimedAppStake)

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseApplicationClaimed{
		ShannonDestAddress:      msg.ShannonDestAddress,
		MorseSrcAddress:         msg.MorseSrcAddress,
		ServiceId:               app.GetServiceConfigs()[0].GetServiceId(),
		ClaimedBalance:          claimedUnstakedTokens,
		ClaimedApplicationStake: claimedAppStake,
		ClaimedAtHeight:         sdkCtx.BlockHeight(),
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseApplicationClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	// Return the response.
	return &migrationtypes.MsgClaimMorseApplicationResponse{
		MorseSrcAddress:         msg.MorseSrcAddress,
		ServiceId:               app.ServiceConfigs[0].GetServiceId(),
		ClaimedBalance:          claimedUnstakedTokens,
		ClaimedApplicationStake: claimedAppStake,
		ClaimedAtHeight:         sdkCtx.BlockHeight(),
	}, nil
}
