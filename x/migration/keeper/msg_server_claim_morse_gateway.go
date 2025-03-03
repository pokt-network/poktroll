package keeper

import (
	"context"
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorseGateway(ctx context.Context, msg *migrationtypes.MsgClaimMorseGateway) (*migrationtypes.MsgClaimMorseGatewayResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseGateway")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonDestAddress is validated in MsgClaimMorseGateway#ValidateBasic().
	shannonAccAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonDestAddress)

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				msg.MorseSrcAddress,
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	// The gateway stake is subsequently escrowed from the shannonDestAddress account balance.
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

	msgStakeGateway := gatewaytypes.NewMsgStakeGateway(
		shannonAccAddr.String(),
		msg.Stake,
	)

	initialGatewayStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	foundGateway, isFound := k.gatewayKeeper.GetGateway(ctx, shannonAccAddr.String())
	if isFound {
		initialGatewayStake = *foundGateway.Stake
	}

	gateway, err := k.gatewayKeeper.StakeGateway(ctx, logger, msgStakeGateway)
	if err != nil {
		// DEV_NOTE: StakeGateway SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	// DEV_NOTE: While "down-staking" isn't currently supported for gateways,
	// it MAY be in the future. When BOTH:
	// - the claimed Shannon account is already staked as a gateway
	// - the MsgClaimMorseGateway stake amount ("default" or otherwise)
	//   is less than the current gateway stake amount
	// then, claimedGatewayStake is set to zero as it would otherwise result in a negative amount.
	// This value is only used in event(s) and the msg response.
	claimedGatewayStake, err := gateway.Stake.SafeSub(initialGatewayStake)
	if err != nil {
		if !strings.Contains(err.Error(), "negative coin amount") {
			return nil, err
		}
		claimedGatewayStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	}

	claimedUnstakedTokens := morseClaimableAccount.TotalTokens().Sub(claimedGatewayStake)

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseGatewayClaimed{
		ShannonDestAddress:  msg.ShannonDestAddress,
		MorseSrcAddress:     msg.MorseSrcAddress,
		ClaimedBalance:      claimedUnstakedTokens,
		ClaimedGatewayStake: claimedGatewayStake,
		ClaimedAtHeight:     sdkCtx.BlockHeight(),
		Gateway:             gateway,
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseGatewayClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	// Return the response.
	return &migrationtypes.MsgClaimMorseGatewayResponse{
		MorseSrcAddress:     msg.MorseSrcAddress,
		ClaimedBalance:      claimedUnstakedTokens,
		ClaimedGatewayStake: claimedGatewayStake,
		ClaimedAtHeight:     sdkCtx.BlockHeight(),
		Gateway:             gateway,
	}, nil
}
