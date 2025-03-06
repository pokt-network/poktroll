package keeper

import (
	"context"
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ClaimMorseApplication performs the following steps, given msg is valid and a
// MorseClaimableAccount exists for the given morseSrcAddress:
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

	// Default to the application stake amount recorded in the MorseClaimableAccount.
	if msg.Stake == nil {
		msg.Stake = &morseClaimableAccount.ApplicationStake
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	// The application stake is subsequently escrowed from the shannonDestAddress account balance.
	if err := k.MintClaimedMorseTokens(ctx, shannonAccAddr, morseClaimableAccount.TotalTokens()); err != nil {
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

	// DEV_NOTE: While "down-staking" isn't currently supported for applications,
	// it MAY be in the future. When BOTH:
	// - the claimed Shannon account is already staked as an application
	// - the MsgClaimMorseApplication stake amount ("default" or otherwise)
	//   is less than the current application stake amount
	// then, claimedAppStake is set to zero as it would otherwise result in a negative amount.
	// This value is only used in event(s) and the msg response.
	claimedAppStake, err := app.Stake.SafeSub(initialAppStake)
	if err != nil {
		if !strings.Contains(err.Error(), "negative coin amount") {
			return nil, err
		}
		claimedAppStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	}

	claimedUnstakedTokens := morseClaimableAccount.TotalTokens().Sub(claimedAppStake)

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseApplicationClaimed{
		ShannonDestAddress:      msg.ShannonDestAddress,
		MorseSrcAddress:         msg.MorseSrcAddress,
		ServiceId:               app.GetServiceConfigs()[0].GetServiceId(),
		ClaimedBalance:          claimedUnstakedTokens,
		ClaimedApplicationStake: claimedAppStake,
		ClaimedAtHeight:         sdkCtx.BlockHeight(),
		Application:             app,
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
		Application:             app,
	}, nil
}
