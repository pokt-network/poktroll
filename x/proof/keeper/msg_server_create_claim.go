package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k msgServer) CreateClaim(ctx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	// TODO_BLOCKER: Prevent Claim upserts after the ClaimWindow is closed.
	// TODO_BLOCKER: Validate the signature on the Claim message corresponds to the supplier before Upserting.

	isSuccessful := false
	defer telemetry.AppMsgCounter(ctx, "create_claim", func() bool { return isSuccessful })

	logger := k.Logger().With("method", "CreateClaim")
	logger.Debug("creating claim")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	session, err := k.queryAndValidateSessionHeader(
		ctx,
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
	claim := types.Claim{
		SupplierAddress: msg.GetSupplierAddress(),
		SessionHeader:   msg.GetSessionHeader(),
		RootHash:        msg.GetRootHash(),
	}

	// TODO_BLOCKER: check if this claim already exists and return an appropriate error
	// in any case where the supplier should no longer be able to update the given proof.
	k.Keeper.UpsertClaim(ctx, claim)

	logger.
		With(
			"session_id", claim.GetSessionHeader().GetSessionId(),
			"session_end_height", claim.GetSessionHeader().GetSessionEndBlockHeight(),
			"supplier", claim.GetSupplierAddress(),
		).
		Debug("created claim")

	isSuccessful = true
	// TODO: return the claim in the response.
	return &types.MsgCreateClaimResponse{}, nil
}
