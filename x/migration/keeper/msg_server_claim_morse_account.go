package keeper

import (
	"context"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) ClaimMorseAccount(ctx context.Context, msg *migrationtypes.MsgClaimMorseAccount) (*migrationtypes.MsgClaimMorseAccountResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonDestAddress is validated in MsgClaimMorseAccount#ValidateBasic().
	shannonAccAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonDestAddress)

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				msg.MorseSrcAddress,
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// Validate the Morse signature.
	publicKey := ed25519.PubKey(morseClaimableAccount.GetPublicKey())
	if err := msg.ValidateMorseSignature(publicKey); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonAccAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// ONLY allow claiming as a non-actor account if the MorseClaimableAccount
	// WAS NOT staked as an application or supplier.
	// A claim of staked POKT from Morse to Shannon
	// SHOULD NOT allow applications or suppliers to bypass the onchain unbonding
	if !morseClaimableAccount.ApplicationStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"Morse account %q is staked as an application, please use `poktrolld migrate claim-application` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	if !morseClaimableAccount.SupplierStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"Morse account %q is staked as an supplier, please use `poktrolld migrate claim-supplier` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Mint the totalTokens to the shannonDestAddress account balance.
	unstakedBalance := morseClaimableAccount.UnstakedBalance
	if err := k.MintClaimedMorseTokens(ctx, shannonAccAddr, unstakedBalance); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Emit an event which signals that the morse account has been claimed.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	event := migrationtypes.EventMorseAccountClaimed{
		SessionEndHeight:   sessionEndHeight,
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
		ClaimedBalance:     unstakedBalance,
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

	return &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress:  msg.MorseSrcAddress,
		ClaimedBalance:   unstakedBalance,
		SessionEndHeight: sessionEndHeight,
	}, nil
}
