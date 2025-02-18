package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	cosmoserrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) ClaimMorseApplication(ctx context.Context, msg *migrationtypes.MsgClaimMorseApplication) (*migrationtypes.MsgClaimMorseApplicationResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	shannonAccAddr, err := cosmostypes.AccAddressFromBech32(msg.ShannonDestAddress)
	// DEV_NOTE: This SHOULD NEVER happen as the shannonDestAddress is validated
	// in MsgClaimMorseAccount#ValidateBasic().
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

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonAccAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Add any non-application actor stakes to the account balance because we're not creating
	// a shannon actor (i.e. not a re-stake claim).
	unstakedBalanceTokens := morseClaimableAccount.UnstakedBalance.
		Add(morseClaimableAccount.SupplierStake)

	// Mint the unstakedBalanceTokens to the shannonDestAddress account balance.
	if err = k.MintClaimedMorseTokens(ctx, shannonAccAddr, unstakedBalanceTokens); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Stake an on-chain application. If the application already exists,
	// increment the stake and replace the service config.
	app, isFound := k.appKeeper.GetApplication(ctx, shannonAccAddr.String())
	if isFound {
		newStake := app.Stake.Add(msg.Stake)
		app.Stake = &newStake
		app.ServiceConfigs = []*sharedtypes.ApplicationServiceConfig{
			msg.ServiceConfig,
		}
	} else {
		app = apptypes.Application{
			Address: shannonAccAddr.String(),
			Stake:   &msg.Stake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				msg.ServiceConfig,
			},
		}
	}

	// Save or update application.
	k.appKeeper.SetApplication(ctx, app)

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseAccountClaimed{
		ClaimedAtHeight:    sdkCtx.BlockHeight(),
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
		ClaimedBalance:     unstakedBalanceTokens,
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
		ClaimedBalance:          unstakedBalanceTokens,
		ClaimedApplicationStake: *app.Stake,
		ClaimedAtHeight:         sdkCtx.BlockHeight(),
		ServiceId:               app.ServiceConfigs[0].GetServiceId(),
	}, nil
}
