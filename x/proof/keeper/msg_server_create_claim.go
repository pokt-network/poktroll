package keeper

import (
	"context"
	"errors"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
)

func (k msgServer) CreateClaim(
	ctx context.Context,
	msg *types.MsgCreateClaim,
) (_ *types.MsgCreateClaimResponse, err error) {
	// Declare claim to reference in telemetry.
	var (
		claim           types.Claim
		isExistingClaim bool
		numRelays       uint64
		numComputeUnits uint64
	)

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingClaim {
			// TODO_IMPROVE: We could track on-chain relays here with claim.GetNumRelays().
			telemetry.ClaimCounter(types.ClaimProofStage_CLAIMED, 1, err)
			telemetry.ClaimComputeUnitsCounter(
				types.ClaimProofStage_CLAIMED,
				numComputeUnits,
				err,
			)
		}
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

	_, isExistingClaim = k.Keeper.GetClaim(ctx, claim.GetSessionHeader().GetSessionId(), claim.GetSupplierAddress())

	k.Keeper.UpsertClaim(ctx, claim)

	logger.Info("created new claim")

	// NB: Don't return these errors, it should not prevent the MsgCreateProofResopnse
	// from being returned. Instead, they will be joined and attached as an "error" label
	// to any metrics tracked in this function.
	// TODO_IMPROVE: While this will surface the error in metrics, it will also cause
	// any counters not to be incremented even though a new proof might have been inserted.
	var tempError error
	numRelays, tempError = claim.GetNumRelays()
	if tempError != nil {
		err = errors.Join(err, tempError)
	}
	numComputeUnits, tempError = claim.GetNumComputeUnits()
	if tempError != nil {
		err = errors.Join(err, tempError)
	}

	// Emit the appropriate event based on whether the claim was created or updated.
	var claimUpsertEvent proto.Message
	switch isExistingClaim {
	case true:
		claimUpsertEvent = proto.Message(
			&types.EventClaimUpdated{
				Claim:           &claim,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	case false:
		claimUpsertEvent = proto.Message(
			&types.EventClaimCreated{
				Claim:           &claim,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	}

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	emitEventErr := sdkCtx.EventManager().EmitTypedEvent(claimUpsertEvent)

	// NB: Don't return this error, it should not prevent the MsgCreateClaimResopnse
	// from being returned. Instead, it will be attached as an "error" label to any
	// metrics tracked in this function.
	// TODO_IMPROVE: While this will surface the error in metrics, it will also cause
	// any counters not to be incremented even though a new claim might have been inserted.
	err = errors.Join(err, emitEventErr)

	// TODO_BETA: return the claim in the response.
	return &types.MsgCreateClaimResponse{}, nil
}
