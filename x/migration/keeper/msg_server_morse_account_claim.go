package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) CreateMorseAccountClaim(goCtx context.Context, msg *types.MsgCreateMorseAccountClaim) (*types.MsgCreateMorseAccountClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetMorseAccountClaim(
		ctx,
		msg.MorseSrcAddress,
	)
	if isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "index already set")
	}

	var morseAccountClaim = types.MorseAccountClaim{
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
		MorseSignature:     msg.MorseSignature,
	}

	k.SetMorseAccountClaim(
		ctx,
		morseAccountClaim,
	)

}
