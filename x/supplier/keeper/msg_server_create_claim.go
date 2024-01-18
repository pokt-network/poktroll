package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *suppliertypes.MsgCreateClaim) (*suppliertypes.MsgCreateClaimResponse, error) {
	// TODO_BLOCKER: Prevent Claim upserts after the ClaimWindow is closed.
	// TODO_BLOCKER: Validate the signature on the Claim message corresponds to the supplier before Upserting.

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger(ctx).With("method", "CreateClaim")
	logger.Debug("creating claim")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	session, err := k.queryAndValidateSessionHeader(
		goCtx,
		msg.GetSessionHeader(),
		msg.GetSupplierAddress(),
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger.
		With(
			"session_id", session.GetSessionId(),
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
