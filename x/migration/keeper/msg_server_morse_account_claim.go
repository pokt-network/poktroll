package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) CreateMorseAccountClaim(
	ctx context.Context,
	msg *migrationtypes.MsgCreateMorseAccountClaim,
) (*migrationtypes.MsgCreateMorseAccountClaimResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check if the value already exists
	_, isFound := k.GetMorseAccountClaim(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if isFound {
		// TODO_IN_THIS_COMMIT: migration module errors...
		// TODO_UPNEXT(@bryanchriswhite:#1034): grpc status errors...
		return nil, sdkerrors.ErrInvalidRequest.Wrapf(
			"morse account already claimed with address %q",
			msg.MorseSrcAddress,
		)
	}

	morseAccountClaim := migrationtypes.MorseAccountClaim{
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
	}

	if err := msg.ValidateBasic(); err != nil {
		// TODO_UPNEXT(@bryanchriswhite:#1034): migration module errors...
		// TODO_IN_THIS_COMMIT: grpc status errors...
		return nil, sdkerrors.ErrInvalidRequest.Wrap(err.Error())
	}

	// TODO_UPNEXT(@bryanchriswhite#1034): Assign claimedBalance based on the MorseAccountState...
	claimedBalance := sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())

	k.SetMorseAccountClaim(
		sdkCtx,
		morseAccountClaim,
	)

	// TODO_UPNEXT(@bryanchriswhite#1034): Emit EventMorseAccountClaimed...

	return &migrationtypes.MsgCreateMorseAccountClaimResponse{
		MorseSrcAddress: msg.MorseSrcAddress,
		ClaimedBalance:  &claimedBalance,
		ClaimedAtHeight: sdkCtx.BlockHeight(),
	}, nil
}
