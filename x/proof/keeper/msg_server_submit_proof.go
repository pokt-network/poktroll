package keeper

// TODO_TECHDEBT(@bryanchriswhite): Replace all logs in x/ from `.Info` to
// `.Debug` when the logger is replaced close to or after MainNet launch.
// Ref: https://github.com/pokt-network/poktroll/pull/448#discussion_r1549742985

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
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
		claim                = new(types.Claim)
		isExistingProof      bool
		numRelays            uint64
		numClaimComputeUnits uint64
	)

	logger := k.Logger().With("method", "SubmitProof")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger.Info("About to start submitting proof")

	// Basic validation of the SubmitProof message.
	if err = msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	logger.Info("validated the submitProof message")

	// Compare msg session header w/ on-chain session header.
	session, err := k.queryAndValidateSessionHeader(ctx, msg.GetSessionHeader(), msg.GetSupplierOperatorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		// Only increment these metrics counters if handling a new claim.
		if !isExistingProof {
			serviceId := session.Header.ServiceId
			applicationAddress := session.Header.ApplicationAddress
			supplierOperatorAddress := msg.GetSupplierOperatorAddress()

			telemetry.ClaimCounter(types.ClaimProofStage_PROVEN, 1, serviceId, applicationAddress, supplierOperatorAddress, err)
			telemetry.ClaimRelaysCounter(types.ClaimProofStage_PROVEN, numRelays, serviceId, applicationAddress, supplierOperatorAddress, err)
			telemetry.ClaimComputeUnitsCounter(types.ClaimProofStage_PROVEN, numClaimComputeUnits, serviceId, applicationAddress, supplierOperatorAddress, err)
		}
	}()

	if err = k.deductProofSubmissionFee(ctx, msg.GetSupplierOperatorAddress()); err != nil {
		logger.Error(fmt.Sprintf("failed to deduct proof submission fee: %v", err))
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Construct the proof
	proof := types.Proof{
		SupplierOperatorAddress: msg.GetSupplierOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		ClosestMerkleProof:      msg.GetProof(),
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"session_id", proof.SessionHeader.SessionId,
		"session_end_height", proof.SessionHeader.SessionEndBlockHeight,
		"supplier_operator_address", proof.SupplierOperatorAddress)

	// Validate proof message commit height is within the respective session's
	// proof submission window using the on-chain session header.
	if err = k.validateProofWindow(ctx, proof.SessionHeader, proof.SupplierOperatorAddress); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Retrieve the corresponding claim for the proof submitted so it can be
	// used in the proof validation below.
	claim, err = k.queryAndValidateClaimForProof(ctx, proof.SessionHeader, proof.SupplierOperatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofClaimNotFound.Wrap(err.Error()).Error())
	}

	// Check if a proof is required for the claim.
	proofRequirement, err := k.ProofRequirementForClaim(ctx, claim)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if proofRequirement == types.ProofRequirementReason_NOT_REQUIRED {
		logger.Warn("trying to submit a proof for a claim that does not require one")
		return nil, status.Error(codes.FailedPrecondition, types.ErrProofNotRequired.Error())
	}

	// Get metadata for the event we want to emit
	numRelays, err = claim.GetNumRelays()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}
	// DEV_NOTE: It is assumed that numClaimComputeUnits = numRelays * serviceComputeUnitsPerRelay
	// has been checked during the claim creation process.
	numClaimComputeUnits, err = claim.GetNumClaimedComputeUnits()
	if err != nil {
		return nil, status.Error(codes.Internal, types.ErrProofInvalidClaimRootHash.Wrap(err.Error()).Error())
	}

	// Check if a prior proof already exists.
	_, isExistingProof = k.GetProof(ctx, proof.SessionHeader.SessionId, proof.SupplierOperatorAddress)

	// Upsert the proof
	k.UpsertProof(ctx, proof)
	logger.Info("successfully upserted the proof")

	// Emit the appropriate event based on whether the claim was created or updated.
	var proofUpsertEvent proto.Message
	switch isExistingProof {
	case true:
		proofUpsertEvent = proto.Message(
			&types.EventProofUpdated{
				Claim:                  claim,
				Proof:                  &proof,
				NumRelays:              numRelays,
				NumClaimedComputeUnits: numClaimComputeUnits,
				// TODO_FOLLOWUP: Add NumEstimatedComputeUnits and ClaimedAmountUpokt
			},
		)
	case false:
		proofUpsertEvent = proto.Message(
			&types.EventProofSubmitted{
				Claim:                  claim,
				Proof:                  &proof,
				NumRelays:              numRelays,
				NumClaimedComputeUnits: numClaimComputeUnits,
				// TODO_FOLLOWUP: Add NumEstimatedComputeUnits and ClaimedAmountUpokt
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

// deductProofSubmissionFee deducts the proof submission fee from the supplier operator's account balance.
func (k Keeper) deductProofSubmissionFee(ctx context.Context, supplierOperatorAddress string) error {
	proofSubmissionFee := k.GetParams(ctx).ProofSubmissionFee
	supplierOperatorAccAddress, err := cosmostypes.AccAddressFromBech32(supplierOperatorAddress)
	if err != nil {
		return err
	}

	accCoins := k.bankKeeper.SpendableCoins(ctx, supplierOperatorAccAddress)
	if accCoins.Len() == 0 {
		return types.ErrProofNotEnoughFunds.Wrapf(
			"account has no spendable coins",
		)
	}

	// Check the balance of upokt is enough to cover the ProofSubmissionFee.
	accBalance := accCoins.AmountOf("upokt")
	if accBalance.LTE(proofSubmissionFee.Amount) {
		return types.ErrProofNotEnoughFunds.Wrapf(
			"account has %s, but the proof submission fee is %s",
			accBalance, proofSubmissionFee,
		)
	}

	// Deduct the proof submission fee from the supplier operator's balance.
	proofSubmissionFeeCoins := cosmostypes.NewCoins(*proofSubmissionFee)
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, supplierOperatorAccAddress, types.ModuleName, proofSubmissionFeeCoins); err != nil {
		return types.ErrProofFailedToDeductFee.Wrapf(
			"account has %s, failed to deduct %s",
			accBalance, proofSubmissionFee,
		)
	}

	return nil
}

// ProofRequirementForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_BLOCKER(@olshansk, #419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) ProofRequirementForClaim(ctx context.Context, claim *types.Claim) (_ types.ProofRequirementReason, err error) {
	logger := k.logger.With("method", "proofRequirementForClaim")

	var requirementReason = types.ProofRequirementReason_NOT_REQUIRED

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		telemetry.ProofRequirementCounter(
			requirementReason,
			claim.SessionHeader.ServiceId,
			claim.SessionHeader.ApplicationAddress,
			claim.SupplierOperatorAddress,
			err,
		)
	}()

	proofParams := k.GetParams(ctx)
	sharedParams := k.sharedKeeper.GetParams(ctx)

	serviceId := claim.GetSessionHeader().GetServiceId()
	relayMiningDifficulty, found := k.serviceKeeper.GetRelayMiningDifficulty(ctx, serviceId)
	if !found {
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(ctx, logger, serviceId, servicekeeper.TargetNumRelays)
	}

	// Retrieve the number of tokens claimed to compare against the threshold.
	// Different services have varying compute_unit -> token multipliers, so the
	// threshold value is done in a common unit denomination.
	claimeduPOKT, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	if err != nil {
		return requirementReason, err
	}

	// Require a proof if the claim's compute units meets or exceeds the threshold.
	// TODO_MAINNET(@olshansk, @red-0ne): Should the threshold be dependant on the stake as well
	// so we slash proportional to the compute units?
	// TODO_IMPROVE(@red-0ne): It might make sense to include whether there was a proof
	// submission error downstream from here. This would require a more comprehensive metrics API.
	if claimeduPOKT.Amount.GTE(proofParams.GetProofRequirementThreshold().Amount) {
		requirementReason = types.ProofRequirementReason_THRESHOLD

		logger.Info(fmt.Sprintf(
			"claim requires proof due to claimed tokens (%s) exceeding threshold (%s)",
			claimeduPOKT,
			proofParams.GetProofRequirementThreshold(),
		))
		return requirementReason, nil
	}

	// Hash of block when proof submission is allowed.
	proofRequirementSeedBlockHash, err := k.getProofRequirementSeedBlockHash(ctx, claim)
	if err != nil {
		return requirementReason, err
	}

	// The probability that a proof is required.
	proofRequirementSampleValue, err := claim.GetProofRequirementSampleValue(proofRequirementSeedBlockHash)
	if err != nil {
		return requirementReason, err
	}

	// Require a proof probabilistically based on the proof_request_probability param.
	// NB: A random value between 0 and 1 will be less than or equal to proof_request_probability
	// with probability equal to the proof_request_probability.
	if proofRequirementSampleValue <= proofParams.GetProofRequestProbability() {
		requirementReason = types.ProofRequirementReason_PROBABILISTIC

		logger.Info(fmt.Sprintf(
			"claim requires proof due to random sample (%.2f) being less than or equal to probability (%.2f)",
			proofRequirementSampleValue,
			proofParams.GetProofRequestProbability(),
		))
		return requirementReason, nil
	}

	logger.Info(fmt.Sprintf(
		"claim does not require proof due to claimed amount (%s) being less than the threshold (%s) and random sample (%.2f) being greater than probability (%.2f)",
		claimeduPOKT,
		proofParams.GetProofRequirementThreshold(),
		proofRequirementSampleValue,
		proofParams.GetProofRequestProbability(),
	))
	return requirementReason, nil
}

// getProofRequirementSeedBlockHash returns the block hash of the seed block for
// the proof requirement probabilistic check.
func (k Keeper) getProofRequirementSeedBlockHash(
	ctx context.Context,
	claim *types.Claim,
) (blockHash []byte, err error) {
	sharedParams, err := k.sharedQuerier.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	sessionEndHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	supplierOperatorAddress := claim.GetSupplierOperatorAddress()

	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)
	proofWindowOpenBlockHash := k.sessionKeeper.GetBlockHash(ctx, proofWindowOpenHeight)

	// TODO_TECHDEBT(@red-0ne): Update the method header of this function to accept (sharedParams, Claim, BlockHash).
	// After doing so, please review all calling sites and simplify them accordingly.
	earliestSupplierProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddress,
	)

	// The proof requirement seed block is the last block of the session, and it is
	// the block that is before the earliest block at which a proof can be committed.
	return k.sessionKeeper.GetBlockHash(ctx, earliestSupplierProofCommitHeight-1), nil
}
