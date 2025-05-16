package keeper

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) RecoverMorseAccount(ctx context.Context, msg *types.MsgRecoverMorseAccount) (*types.MsgRecoverMorseAccountResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate morese account recovery message.
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the authority is valid.
	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.GetAuthority(),
			).Error(),
		)
	}

	// Check if the morse account is listed in the recoverable accounts list.
	if !slices.Contains(RecoveryAllowlist, msg.GetMorseSrcAddress()) {
		return nil, status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"morse account %q is not recoverable",
				msg.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Look up the morse account in the store.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		msg.GetMorseSrcAddress(),
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"no morse recoverable account exists with address %q",
				msg.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"morse address %q has already been recovered at height %d onto shannon address %q",
				msg.GetMorseSrcAddress(),
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// Recover any balances or stakes from the morse account.
	recoveredBalance := morseClaimableAccount.UnstakedBalance
	recoveredBalance = recoveredBalance.Add(morseClaimableAccount.ApplicationStake)
	recoveredBalance = recoveredBalance.Add(morseClaimableAccount.SupplierStake)

	currentHeight := sdkCtx.BlockHeight()

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = msg.GetShannonDestAddress()
	morseClaimableAccount.ClaimedAtHeight = currentHeight

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Mint the recovered balance to the shannonDestAddress account balance.
	shannonAccAddr := sdk.MustAccAddressFromBech32(msg.GetShannonDestAddress())
	if err := k.MintClaimedMorseTokens(ctx, shannonAccAddr, recoveredBalance); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Emit an event which signals that the morse account has been recovered.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	event := migrationtypes.EventMorseAccountRecovered{
		SessionEndHeight:   sessionEndHeight,
		RecoveredBalance:   recoveredBalance,
		ShannonDestAddress: msg.GetShannonDestAddress(),
		MorseSrcAddress:    msg.GetMorseSrcAddress(),
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	return &types.MsgRecoverMorseAccountResponse{
		SessionEndHeight:   sessionEndHeight,
		RecoveredBalance:   recoveredBalance,
		ShannonDestAddress: msg.GetShannonDestAddress(),
		MorseSrcAddress:    msg.GetMorseSrcAddress(),
	}, nil
}
