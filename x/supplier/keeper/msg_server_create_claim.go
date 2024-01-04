package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *suppliertypes.MsgCreateClaim) (*suppliertypes.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "CreateClaim")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	sessionRes, err := k.ValidateSessionHeader(
		goCtx,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	)
	if err != nil {
		return nil, err
	}

	logger.
		With(
			"session_id", sessionRes.GetSession().GetSessionId(),
			"session_end_height", msg.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		).
		Debug("validated claim")

	/*
		TODO_INCOMPLETE:

		### Msg distribution validation (depends on sessionRes validation)
		1. [ ] governance-based earliest block offset
		2. [ ] pseudo-randomize earliest block offset

		### Claim validation
		1. [x] sessionRes validation
		2. [ ] msg distribution validation
	*/

	// Construct and upsert claim after all validation.
	claim := suppliertypes.Claim{
		SupplierAddress: msg.GetSupplierAddress(),
		SessionHeader:   msg.GetSessionHeader(),
		RootHash:        msg.RootHash,
	}

	// TODO_TECHDEBT: check if this claim already exists and return an appropriate error
	// in any case where the supplier should no longer be able to update the given proof.
	k.Keeper.UpsertClaim(ctx, claim)

	logger.
		With(
			"session_id", claim.GetSessionHeader().GetSessionId(),
			"session_end_height", claim.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", claim.GetSupplierAddress(),
		).
		Debug("created claim")

	// TODO: return the claim in the response.
	return &suppliertypes.MsgCreateClaimResponse{}, nil
}
