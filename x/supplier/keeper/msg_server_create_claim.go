package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	claim := types.Claim{
		SupplierAddress:       msg.SupplierAddress,
		SessionId:             msg.SessionHeader.SessionId,
		SessionEndBlockHeight: uint64(msg.SessionHeader.SessionEndBlockHeight),
		RootHash:              msg.RootHash,
	}
	k.Keeper.InsertClaim(ctx, claim)

	/*
		INCOMPLETE: Handling the message

		## Validation

		### Session validation
		1. [ ] claimed session ID matches on-chain session ID
		2. [ ] this supplier is in the session's suppliers list

		### Msg distribution validation (depends on session validation)
		1. [ ] governance-based earliest block offset
		2. [ ] pseudo-randomize earliest block offset

		### Claim validation
		1. [ ] session validation
		2. [ ] msg distribution validation

		## Persistence
		1. [ ] create claim message
			- supplier address
			- session header
			- claim
		2. [ ] last block height commitment; derives:
			- last block committed hash, must match proof path
			- session ID (?)
	*/
	_ = ctx

	return &types.MsgCreateClaimResponse{}, nil
}
