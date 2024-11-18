package keeper

import (
	"context"
	"fmt"

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
		claim                types.Claim
		isExistingClaim      bool
		numRelays            uint64
		numClaimComputeUnits uint64
	)

	logger := k.Logger().With("method", "CreateClaim")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger.Info("creating claim")

	// Basic validation of the CreateClaim message.
	if err = msg.ValidateBasic(); err != nil {
		return nil, err
	}
	logger.Info("validated the createClaim message")

	// Compare msg session header w/ on-chain session header.
	session, err := k.queryAndValidateSessionHeader(ctx, msg.GetSessionHeader(), msg.GetSupplierOperatorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Defer telemetry calls to a helper function to keep business logic clean.
	defer k.finalizeCreateClaimTelemetry(session, msg, isExistingClaim, numRelays, numClaimComputeUnits, err)

	// Construct and insert claim
	claim = types.Claim{
		SupplierOperatorAddress: msg.GetSupplierOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		RootHash:                msg.GetRootHash(),
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.
		With(
			"session_id", session.GetSessionId(),
			"session_end_height", claim.SessionHeader.GetSessionEndBlockHeight(),
			"supplier_operator_address", msg.GetSupplierOperatorAddress(),
		)

	// Validate claim message commit height is within the respective session's
	// claim creation window using the on-chain session header.
	if err = k.validateClaimWindow(ctx, claim.SessionHeader, claim.SupplierOperatorAddress); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Get the number of volume applicable relays in the claim
	numRelays, err = claim.GetNumRelays()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}

	// Get the number of claimed compute units in the claim
	numClaimComputeUnits, err = claim.GetNumClaimedComputeUnits()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrapf("%v", err).Error())
	}

	// Get the number of compute units per relay for the service
	serviceComputeUnitsPerRelay, err := k.getServiceComputeUnitsPerRelay(ctx, claim.SessionHeader.ServiceId)
	if err != nil {
		return nil, status.Error(codes.NotFound, types.ErrProofServiceNotFound.Wrapf("%v", err).Error())
	}

	// For a specific service, each relay costs the same amount.
	// TODO_POST_MAINNET: Investigate ways of having request specific compute unit
	// costs within the same service.
	numExpectedComputeUnitsToClaim := numRelays * serviceComputeUnitsPerRelay

	// Ensure the number of compute units claimed is equal to the number of relays
	if numClaimComputeUnits != numExpectedComputeUnitsToClaim {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrProofComputeUnitsMismatch.Wrap(
				fmt.Sprintf("claim compute units: %d is not equal to number of relays %d * compute units per relay %d for service %s",
					numClaimComputeUnits,
					numRelays,
					serviceComputeUnitsPerRelay,
					claim.SessionHeader.ServiceId,
				),
			).Error(),
		)
	}

	_, isExistingClaim = k.Keeper.GetClaim(ctx, claim.GetSessionHeader().GetSessionId(), claim.GetSupplierOperatorAddress())

	// Upsert the claim
	k.Keeper.UpsertClaim(ctx, claim)
	logger.Info("successfully upserted the claim")

	// Get the service ID relayMiningDifficulty to calculate the claimed uPOKT.
	serviceId := session.GetHeader().GetServiceId()
	sharedParams := k.sharedKeeper.GetParams(ctx)
	relayMiningDifficulty, _ := k.serviceKeeper.GetRelayMiningDifficulty(ctx, serviceId)
	claimedUPOKT, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)

	// Emit the appropriate event based on whether the claim was created or updated.
	var claimUpsertEvent proto.Message
	switch isExistingClaim {
	case true:
		claimUpsertEvent = proto.Message(
			&types.EventClaimUpdated{
				Claim:                    &claim,
				NumRelays:                numRelays,
				NumClaimedComputeUnits:   numClaimComputeUnits,
				NumEstimatedComputeUnits: numExpectedComputeUnitsToClaim,
				ClaimedUpokt:             &claimedUPOKT,
			},
		)
	case false:
		claimUpsertEvent = proto.Message(
			&types.EventClaimCreated{
				Claim:                    &claim,
				NumRelays:                numRelays,
				NumClaimedComputeUnits:   numClaimComputeUnits,
				NumEstimatedComputeUnits: numExpectedComputeUnitsToClaim,
				ClaimedUpokt:             &claimedUPOKT,
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

// finalizeCreateClaimTelemetry defers telemetry calls to be executed after business logic,
// incrementing counters based on whether a new claim was handled successfully.
// Meant to run deferred.
func (k msgServer) finalizeCreateClaimTelemetry(session *sessiontypes.Session, msg *types.MsgCreateClaim, isExistingClaim bool, numRelays, numClaimComputeUnits uint64, err error) {
	// Only increment these metrics counters if handling a new claim.
	if !isExistingClaim {
		serviceId := session.Header.ServiceId
		applicationAddress := session.Header.ApplicationAddress
		supplierOperatorAddress := msg.GetSupplierOperatorAddress()
		claimProofStage := types.ClaimProofStage_CLAIMED.String()

		telemetry.ClaimCounter(claimProofStage, 1, serviceId, applicationAddress, supplierOperatorAddress, err)
		telemetry.ClaimRelaysCounter(claimProofStage, numRelays, serviceId, applicationAddress, supplierOperatorAddress, err)
		telemetry.ClaimComputeUnitsCounter(claimProofStage, numClaimComputeUnits, serviceId, applicationAddress, supplierOperatorAddress, err)
	}
}
