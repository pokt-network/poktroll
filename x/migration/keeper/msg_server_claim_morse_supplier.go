package keeper

import (
	"context"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// ClaimMorseSupplier performs the following steps, given msg is valid and a
// MorseClaimableAccount exists for the given morse_src_address:
//   - Mint and transfer all tokens (unstaked balance plus supplier stake) of the
//     MorseClaimableAccount to the shannonDestAddress.
//   - Mark the MorseClaimableAccount as claimed (i.e. adding the shannon_dest_address
//     and claimed_at_height).
//   - Stake a supplier for the amount specified in the MorseClaimableAccount,
//     and the services specified in the msg.
func (k msgServer) ClaimMorseSupplier(
	ctx context.Context,
	msg *migrationtypes.MsgClaimMorseSupplier,
) (*migrationtypes.MsgClaimMorseSupplierResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseSupplier")

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonOwnerAddress and shannonOperatorAddress are validated in MsgClaimMorseSupplier#ValidateBasic().
	shannonOwnerAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonOwnerAddress)

	// Default to the shannonOwnerAddr as the shannonOperatorAddr if not provided.
	shannonOperatorAddr := shannonOwnerAddr
	if msg.ShannonOperatorAddress != "" {
		shannonOperatorAddr = cosmostypes.MustAccAddressFromBech32(msg.ShannonOperatorAddress)
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

	// ONLY allow claiming as a supplier account if the MorseClaimableAccount
	// WAS staked as a supplier AND NOT as an application. A claim of staked POKT
	// from Morse to Shannon SHOULD NOT allow applications or suppliers to bypass
	// the onchain unbonding period.
	if !morseClaimableAccount.ApplicationStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is staked as an application, please use `poktrolld migrate claim-application` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	if !morseClaimableAccount.SupplierStake.IsPositive() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is not staked as an supplier or application, please use `poktrolld migrate claim-account` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	// The Supplier stake is subsequently escrowed from the shannonDestAddress account balance.
	// NOTE: The current supplier module's staking fee parameter will subsequently be deducted
	// from the claimed balance.
	if err := k.MintClaimedMorseTokens(ctx, shannonOwnerAddr, morseClaimableAccount.TotalTokens()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonOwnerAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Query for any existing supplier stake prior to staking.
	preClaimSupplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt())
	foundSupplier, isFound := k.supplierKeeper.GetSupplier(ctx, shannonOwnerAddr.String())
	if isFound {
		preClaimSupplierStake = *foundSupplier.Stake
	}

	// Stake (or update) the supplier.
	msgStakeSupplier := suppliertypes.NewMsgStakeSupplier(
		shannonOwnerAddr.String(),
		shannonOwnerAddr.String(),
		shannonOperatorAddr.String(),
		preClaimSupplierStake.Add(morseClaimableAccount.GetSupplierStake()),
		msg.Services,
	)
	supplier, err := k.supplierKeeper.StakeSupplier(ctx, logger, msgStakeSupplier)
	if err != nil {
		// DEV_NOTE: StakeSupplier SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	claimedSupplierStake := morseClaimableAccount.GetSupplierStake()
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	claimedUnstakedBalance := morseClaimableAccount.GetUnstakedBalance()

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseSupplierClaimed{
		MorseSrcAddress:      msg.MorseSrcAddress,
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             supplier,
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
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             supplier,
	}, nil
}
