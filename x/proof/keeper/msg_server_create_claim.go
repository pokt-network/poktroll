package keeper

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k msgServer) CreateClaim(
	ctx context.Context,
	msg *types.MsgCreateClaim,
) (_ *types.MsgCreateClaimResponse, err error) {
	// TODO_BLOCKER(@bryanchriswhite): Prevent Claim upserts after the ClaimWindow is closed.
	// TODO_BLOCKER(@bryanchriswhite): Validate the signature on the Claim message corresponds to the supplier before Upserting.

	// Declare claim to reference in telemetry.
	var claim types.Claim

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// TODO_IMPROVE: We could track on-chain relays here with claim.GetNumRelays().
		numComputeUnits, deferredErr := claim.GetNumComputeUnits()
		err = errors.Join(err, deferredErr)

		telemetry.ClaimCounter(telemetry.ClaimProofStageClaimed, 1, err)
		telemetry.ClaimComputeUnitsCounter(
			telemetry.ClaimProofStageClaimed,
			numComputeUnits,
			err,
		)
	}()

	logger := k.Logger().With("method", "CreateClaim")
	logger.Info("creating claim")

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Compare msg session header w/ on-chain session header.
	session, err := k.queryAndValidateSessionHeader(ctx, msg)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Use the session header from the on-chain hydrated session.
	sessionHeader := session.GetHeader()

	// Set the session header to the on-chain hydrated session header.
	msg.SessionHeader = sessionHeader

	// Validate claim message commit height is within the respective session's
	// claim creation window using the on-chain session header.
	if err := k.validateClaimWindow(ctx, msg); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	logger = logger.
		With(
			"session_id", session.GetSessionId(),
			"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		)

	/*
		TODO_BLOCKER(@bryanchriswhite):

		### Msg distribution validation (depends on sessionRes validation)
		1. [ ] governance-based earliest block offset
		2. [ ] pseudo-randomize earliest block offset

		### Claim validation
		1. [x] sessionRes validation
		2. [ ] msg distribution validation
	*/

	logger.Info("validated claim")

	// Assign and upsert claim after all validation.
	claim = types.Claim{
		SupplierAddress: msg.GetSupplierAddress(),
		SessionHeader:   sessionHeader,
		RootHash:        msg.GetRootHash(),
	}

	// TODO_BLOCKER(@Olshansk): check if this claim already exists and return an
	// appropriate error in any case where the supplier should no longer be able
	// to update the given proof.
	k.Keeper.UpsertClaim(ctx, claim)
	defer telemetry.ComputeUnitsCounter(telemetry.ClaimProofStageClaimed, &claim)

	logger.Info("created new claim")

	// TODO_BETA: return the claim in the response.
	return &types.MsgCreateClaimResponse{}, nil
}
