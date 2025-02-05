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
	return &types.MsgCreateMorseAccountClaimResponse{}, nil
}

func (k msgServer) UpdateMorseAccountClaim(goCtx context.Context, msg *types.MsgUpdateMorseAccountClaim) (*types.MsgUpdateMorseAccountClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value exists
	valFound, isFound := k.GetMorseAccountClaim(
		ctx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg shannonDestAddress is the same as the current owner
	if msg.ShannonDestAddress != valFound.ShannonDestAddress {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	var morseAccountClaim = types.MorseAccountClaim{
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.MorseSrcAddress,
		MorseSignature:     msg.MorseSignature,
	}

	k.SetMorseAccountClaim(ctx, morseAccountClaim)

	return &types.MsgUpdateMorseAccountClaimResponse{}, nil
}

func (k msgServer) DeleteMorseAccountClaim(goCtx context.Context, msg *types.MsgDeleteMorseAccountClaim) (*types.MsgDeleteMorseAccountClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value exists
	valFound, isFound := k.GetMorseAccountClaim(
		ctx,
		msg.MorseSrcAddress,
	)
	if !isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg shannonDestAddress is the same as the current owner
	if msg.ShannonDestAddress != valFound.ShannonDestAddress {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	k.RemoveMorseAccountClaim(
		ctx,
		msg.MorseSrcAddress,
	)

	return &types.MsgDeleteMorseAccountClaimResponse{}, nil
}
