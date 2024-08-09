package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
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

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingClaim {
			telemetry.ClaimCounter(types.ClaimProofStage_CLAIMED, 1, err)
			telemetry.ClaimRelaysCounter(types.ClaimProofStage_CLAIMED, numRelays, err)
			telemetry.ClaimComputeUnitsCounter(types.ClaimProofStage_CLAIMED, numComputeUnits, err)
		}
	}()

	logger := k.Logger().With("method", "CreateClaim")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger.Info("creating claim")

	// Basic validation of the CreateClaim message.
	if err = msg.ValidateBasic(); err != nil {
		return nil, err
	}
	logger.Info("validated the createClaim message")

	// Compare msg session header w/ on-chain session header.
	session, err := k.queryAndValidateSessionHeader(ctx, msg.GetSessionHeader(), msg.GetSupplierAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Construct and insert claim
	claim = types.Claim{
		SupplierAddress: msg.GetSupplierAddress(),
		SessionHeader:   session.GetHeader(),
		RootHash:        msg.GetRootHash(),
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.
		With(
			"session_id", session.GetSessionId(),
			"session_end_height", claim.SessionHeader.GetSessionEndBlockHeight(),
			"supplier", msg.GetSupplierAddress(),
		)

	// Validate claim message commit height is within the respective session's
	// claim creation window using the on-chain session header.
	if err = k.validateClaimWindow(ctx, claim.SessionHeader, claim.SupplierAddress); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Get metadata for the event we want to emit
	numRelays, err = claim.GetNumRelays()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}
	numComputeUnits, err = claim.GetNumComputeUnits()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}
	_, isExistingClaim = k.Keeper.GetClaim(ctx, claim.GetSessionHeader().GetSessionId(), claim.GetSupplierAddress())

	// TODO_UPNEXT(#705): Check (and test) that numClaimComputeUnits is equal
	// to num_relays * the_compute_units_per_relay for this_service.
	// Add a comment that for now, we expect it to be the case because every
	// relay for a specific service is wroth the same, but may change in the
	// future.

	// Upsert the claim
	k.Keeper.UpsertClaim(ctx, claim)
	logger.Info("successfully upserted the claim")

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

	return &types.MsgCreateClaimResponse{
		Claim: &claim,
	}, nil
}
