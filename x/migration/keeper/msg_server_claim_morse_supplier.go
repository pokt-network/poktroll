package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	cosmoserrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) ClaimMorseSupplier(ctx context.Context, msg *migrationtypes.MsgClaimMorseSupplier) (*migrationtypes.MsgClaimMorseSupplierResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseSupplier")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	shannonAccAddr, err := cosmostypes.AccAddressFromBech32(msg.ShannonDestAddress)
	// DEV_NOTE: This SHOULD NEVER happen as the shannonDestAddress is validated
	// in MsgClaimMorseSupplier#ValidateBasic().
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
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				msg.MorseSrcAddress,
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// Default to the supplier stake amount recorded in the MorseClaimableAccount.
	if msg.Stake == nil {
		msg.Stake = &morseClaimableAccount.SupplierStake
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	// The Supplier stake is subsequently escrowed from the shannonDestAddress account balance.
	// NOTE: The current supplier module's staking fee parameter will subsequently be deducted
	// from the claimed balance.
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

	msgStakeSupplier := suppliertypes.NewMsgStakeSupplier(
		shannonAccAddr.String(),
		shannonAccAddr.String(),
		shannonAccAddr.String(),
		*msg.Stake,
		[]*sharedtypes.SupplierServiceConfig{msg.ServiceConfig},
	)

	initialSupplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt())
	foundSupplier, isFound := k.supplierKeeper.GetSupplier(ctx, shannonAccAddr.String())
	if isFound {
		initialSupplierStake = *foundSupplier.Stake
	}

	supplier, err := k.supplierKeeper.StakeSupplier(ctx, logger, msgStakeSupplier)
	if err != nil {
		// DEV_NOTE: StakeSupplier SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	// DEV_NOTE: When BOTH:
	// - the claimed Shannon account is already staked as a supplier
	// - the MsgClaimMorseSupplier stake amount ("default" or otherwise)
	//   is less than the current supplier stake amount
	// then, claimedSupplierStake is set to zero as it would otherwise result in a negative amount.
	// This value is only used in event(s) and the msg response.
	claimedSupplierStake, err := supplier.Stake.SafeSub(initialSupplierStake)
	if err != nil {
		if !strings.Contains(err.Error(), "negative coin amount") {
			return nil, err
		}
		claimedSupplierStake = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	}

	claimedUnstakedTokens := morseClaimableAccount.TotalTokens().Sub(claimedSupplierStake)

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseSupplierClaimed{
		ShannonDestAddress:   msg.ShannonDestAddress,
		MorseSrcAddress:      msg.MorseSrcAddress,
		ServiceId:            supplier.GetServices()[0].GetServiceId(),
		ClaimedBalance:       claimedUnstakedTokens,
		ClaimedSupplierStake: claimedSupplierStake,
		ClaimedAtHeight:      sdkCtx.BlockHeight(),
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	// Return the response.
	return &migrationtypes.MsgClaimMorseSupplierResponse{
		MorseSrcAddress:      msg.MorseSrcAddress,
		ServiceId:            supplier.Services[0].GetServiceId(),
		ClaimedBalance:       claimedUnstakedTokens,
		ClaimedSupplierStake: claimedSupplierStake,
		ClaimedAtHeight:      sdkCtx.BlockHeight(),
	}, nil
}
