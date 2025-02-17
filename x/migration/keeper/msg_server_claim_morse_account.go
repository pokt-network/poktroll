package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorseAccount(ctx context.Context, msg *migrationtypes.MsgClaimMorseAccount) (*migrationtypes.MsgClaimMorseAccountResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseClaimableAccount.Wrap(err.Error()).Error(),
		)
	}

	shannonAccAddr, err := cosmostypes.AccAddressFromBech32(msg.ShannonDestAddress)
	// DEV_NOTE: This SHOULD NEVER happen as the shannonDestAddress is validated
	// in MsgClaimMorseAccount#ValidateBasic().
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			errors.ErrInvalidAddress.Wrapf(
				"failed to parse shannon destination address (%s): %s",
				msg.ShannonDestAddress, err,
			).Error(),
		)
	}

	// Ensure that a claim for the given morseSrcAddress does not already exist.
	morseAccountClaim, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseClaimableAccount.Wrapf(
				"no morse claimable account exists with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	k.SetMorseClaimableAccount(
		sdkCtx,
		morseAccountClaim,
	)

	// Add any actor stakes to the account balance because we're not creating
	// a shannon actor (i.e. not a re-stake claim).
	totalTokens := morseAccountClaim.UnstakedBalance.
		Add(morseAccountClaim.ApplicationStake).
		Add(morseAccountClaim.SupplierStake)

	// Mint the sum of the account balance (totalTokens) and any actor stakes to the migration module account.
	if err = k.bankKeeper.MintCoins(ctx, migrationtypes.ModuleName, cosmostypes.NewCoins(totalTokens)); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseClaimableAccount.Wrapf(
				"failed to mint coins: %v",
				err,
			).Error(),
		)
	}

	// Transfer the totalTokens to the shannonDestAddress account.
	if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx,
		migrationtypes.ModuleName,
		shannonAccAddr,
		cosmostypes.NewCoins(totalTokens),
	); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseClaimableAccount.Wrapf(
				"failed to send coins: %v",
				err,
			).Error(),
		)
	}

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseAccountClaimed{
		ClaimedAtHeight:    sdkCtx.BlockHeight(),
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
		ClaimedBalance:     totalTokens,
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseClaimableAccount.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	return &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress: msg.MorseSrcAddress,
		ClaimedBalance:  totalTokens,
		ClaimedAtHeight: sdkCtx.BlockHeight(),
	}, nil
}
