package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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

	// Defer telemetry calls so that they reference the 	final values the relevant variables.
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

	if err = msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Compare msg session header w/ on-chain session header.
	var session *sessiontypes.Session
	session, err = k.queryAndValidateSessionHeader(ctx, msg)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Use the session header from the on-chain hydrated session.
	sessionHeader := session.GetHeader()

	// Set the session header to the on-chain hydrated session header.
	msg.SessionHeader = sessionHeader

	// Validate claim message commit height is within the respective session's
	// claim creation window using the on-chain session header.
	if err = k.validateClaimWindow(ctx, msg); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	logger = logger.
		With(
			"session_id", session.GetSessionId(),
			"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		)

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

	numRelays, err = claim.GetNumRelays()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}
	numComputeUnits, err = claim.GetNumComputeUnits()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
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
	if err = sdkCtx.EventManager().EmitTypedEvent(claimUpsertEvent); err != nil {
		return nil, status.Error(
			codes.Internal,
			sharedtypes.ErrSharedEmitEvent.Wrapf(
				"failed to emit event type %T: %v",
				claimUpsertEvent,
				err,
			).Error(),
		)
	}

	// TODO_BETA: return the claim in the response.
	return &types.MsgCreateClaimResponse{
		Claim: &claim,
	}, nil
}
