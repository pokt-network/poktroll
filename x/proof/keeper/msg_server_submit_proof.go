package keeper

// TODO_TECHDEBT(@bryanchriswhite): Replace all logs in x/ from `.Info` to
// `.Debug` when the logger is replaced close to or after MainNet launch.
// Ref: https://github.com/pokt-network/poktroll/pull/448#discussion_r1549742985

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

// SubmitProof is the server handler to submit and store a proof on-chain.
// A proof that's stored on-chain is what leads to rewards (i.e. inflation)
// downstream, making this a critical part of the protocol.
//
// Note that the validation of the proof is done in `EnsureValidProof`. However,
// preliminary checks are done in the handler to prevent sybil or DoS attacks on
// full nodes because storing and validating proofs is expensive.
//
// We are playing a balance of security and efficiency here, where enough validation
// is done on proof submission, and exhaustive validation is done during session
// settlement.
//
// The entity sending the SubmitProof messages does not necessarily need
// to correspond to the supplier signing the proof. For example, a single entity
// could (theoretically) batch multiple proofs (signed by the corresponding supplier)
// into one transaction to save on transaction fees.
func (k msgServer) SubmitProof(
	ctx context.Context,
	msg *types.MsgSubmitProof,
) (_ *types.MsgSubmitProofResponse, err error) {
	// Declare claim to reference in telemetry.
	var (
		claim           = new(types.Claim)
		isExistingProof bool
		numRelays       uint64
		numComputeUnits uint64
	)

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingProof {
			telemetry.ClaimCounter(types.ClaimProofStage_PROVEN, 1, err)
			telemetry.ClaimRelaysCounter(types.ClaimProofStage_PROVEN, numRelays, err)
			telemetry.ClaimComputeUnitsCounter(types.ClaimProofStage_PROVEN, numComputeUnits, err)
		}
	}()

	logger := k.Logger().With("method", "SubmitProof")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger.Info("About to start submitting proof")

	// Basic validation of the SubmitProof message.
	if err = msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	logger.Info("validated the submitProof message")

	// Compare msg session header w/ on-chain session header.
	session, err := k.queryAndValidateSessionHeader(ctx, msg.GetSessionHeader(), msg.GetSupplierAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Construct the proof
	proof := types.Proof{
		SupplierAddress:    msg.GetSupplierAddress(),
		SessionHeader:      session.GetHeader(),
		ClosestMerkleProof: msg.GetProof(),
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"session_id", proof.SessionHeader.SessionId,
		"session_end_height", proof.SessionHeader.SessionEndBlockHeight,
		"supplier", proof.SupplierAddress)

	// Validate proof message commit height is within the respective session's
	// proof submission window using the on-chain session header.
	if err = k.validateProofWindow(ctx, proof.SessionHeader, proof.SupplierAddress); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Retrieve the corresponding claim for the proof submitted so it can be
	// used in the proof validation below.
	claim, err = k.queryAndValidateClaimForProof(ctx, proof.SessionHeader, proof.SupplierAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofClaimNotFound.Wrap(err.Error()).Error())
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
	_, isExistingProof = k.GetProof(ctx, proof.SessionHeader.SessionId, proof.SupplierAddress)

	// Upsert the proof
	k.UpsertProof(ctx, proof)
	logger.Info("successfully upserted the proof")

	// Emit the appropriate event based on whether the claim was created or updated.
	var proofUpsertEvent proto.Message
	switch isExistingProof {
	case true:
		proofUpsertEvent = proto.Message(
			&types.EventProofUpdated{
				Claim:           claim,
				Proof:           &proof,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	case false:
		proofUpsertEvent = proto.Message(
			&types.EventProofSubmitted{
				Claim:           claim,
				Proof:           &proof,
				NumRelays:       numRelays,
				NumComputeUnits: numComputeUnits,
			},
		)
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(proofUpsertEvent); err != nil {
		return nil, status.Error(
			codes.Internal,
			sharedtypes.ErrSharedEmitEvent.Wrapf(
				"failed to emit event type %T: %v",
				proofUpsertEvent,
				err,
			).Error(),
		)
	}

	return &types.MsgSubmitProofResponse{
		Proof: &proof,
	}, nil
}
