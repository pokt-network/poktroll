package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// CreateMorseAccountClaim creates a new MorseAccountClaim if the claim (msg)
// is valid AND the corresponding morse account has not already been claimed.
func (k msgServer) CreateMorseAccountClaim(
	ctx context.Context,
	msg *migrationtypes.MsgCreateMorseAccountClaim,
) (*migrationtypes.MsgCreateMorseAccountClaimResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Ensure that a claim for the given morseSrcAddress does not already exist.
	_, isFound := k.GetMorseAccountClaim(
		sdkCtx,
		msg.MorseSrcAddress,
	)
	if isFound {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse account already claimed with address %q",
				msg.MorseSrcAddress,
			).Error(),
		)
	}

	morseAccountClaim := migrationtypes.MorseAccountClaim{
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			migrationtypes.ErrMorseAccountClaim.Wrap(err.Error()).Error(),
		)
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
