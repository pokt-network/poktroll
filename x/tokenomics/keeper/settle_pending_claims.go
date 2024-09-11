package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
// Events are emitted for each claim that is settled or removed.
// On-chain Claims & Proofs are deleted after they're settled or expired to free up space.
//
// TODO_TECHDEBT: Refactor this function to return a struct instead of multiple return values.
func (k Keeper) SettlePendingClaims(ctx sdk.Context) (
	settledResult types.PendingClaimsResult,
	expiredResult types.PendingClaimsResult,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		return settledResult, expiredResult, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	// Initialize results structs.
	settledResult = types.NewClaimSettlementResult()
	expiredResult = types.NewClaimSettlementResult()

	// A map from a supplier operator address to the number of expired claims that
	// supplier has in this session.
	// Expired claims due to reasons such as invalid or missing proofs when required.
	supplierToExpiredClaimCount := make(map[string]uint64)
	logger.Debug("settling expiring claims")
	for _, claim := range expiringClaims {
		var (
			numClaimComputeUnits uint64
			numClaimRelays       uint64
			proofRequirement     prooftypes.ProofRequirementReason
		)

		// NB: Note that not every (Req, Res) pair in the session is inserted in
		// the tree for scalability reasons. This is the count of non-empty leaves
		// that matched the necessary difficulty and is therefore an estimation
		// of the total number of relays serviced and work done.
		numClaimComputeUnits, err = claim.GetNumComputeUnits()
		if err != nil {
			return settledResult, expiredResult, err
		}

		numClaimRelays, err = claim.GetNumRelays()
		if err != nil {
			return settledResult, expiredResult, err
		}

		sessionId := claim.SessionHeader.SessionId

		proof, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierOperatorAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		proofRequirement, err = k.proofRequirementForClaim(ctx, &claim)
		if err != nil {
			return settledResult, expiredResult, err
		}

		logger = k.logger.With(
			"session_id", sessionId,
			"supplier_operator_address", claim.SupplierOperatorAddress,
			"num_claim_compute_units", numClaimComputeUnits,
			"num_relays_in_session_tree", numClaimRelays,
			"proof_requirement", proofRequirement,
		)

		proofIsRequired := (proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED)
		if proofIsRequired {
			expirationReason := types.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED // EXPIRATION_REASON_UNSPECIFIED is the default

			if isProofFound {
				if err = k.proofKeeper.EnsureValidProof(ctx, &proof); err != nil {
					logger.Warn(fmt.Sprintf("Proof was found but is invalid due to %v", err))
					expirationReason = types.ClaimExpirationReason_PROOF_INVALID
				}
			} else {
				expirationReason = types.ClaimExpirationReason_PROOF_MISSING
			}

			// If the proof is missing or invalid -> expire it
			if expirationReason != types.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED {
				// Proof was required but is invalid or not found.
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:            &claim,
					NumComputeUnits:  numClaimComputeUnits,
					NumRelays:        numClaimRelays,
					ExpirationReason: expirationReason,
					// TODO_CONSIDERATION: Add the error to the event if the proof was invalid.
				}
				if err = ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return settledResult, expiredResult, err
				}

				logger.Info(fmt.Sprintf(
					"claim expired due to %s",
					types.ClaimExpirationReason_name[int32(expirationReason)]),
				)

				// Collect all the slashed supplier operator addresses to later check
				// if they have to be unstaked because of stake below the minimum.
				// The unstaking check is not done here because the slashed supplier may
				// have other valid claims and the protocol might want to touch the supplier
				// owner or operator balances if the stake is negative.
				supplierToExpiredClaimCount[claim.SupplierOperatorAddress]++

				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)
				if isProofFound {
					k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierOperatorAddress)
				}

				expiredResult.NumClaims++
				expiredResult.NumRelays += numClaimRelays
				expiredResult.NumComputeUnits += numClaimComputeUnits
				continue
			}
		}

		// If this code path is reached, then either:
		// 1. The claim does not require a proof.
		// 2. The claim requires a proof and a valid proof was found.

		// Manage the mint & burn accounting for the claim.
		if err = k.ProcessTokenLogicModules(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error processing token logic modules for claim %q: %v", claim.SessionHeader.SessionId, err))
			return settledResult, expiredResult, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:            &claim,
			NumRelays:        numClaimRelays,
			NumComputeUnits:  numClaimComputeUnits,
			ProofRequirement: proofRequirement,
		}

		if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return settledResult, expiredResult, err
		}

		if err = ctx.EventManager().EmitTypedEvent(&prooftypes.EventProofUpdated{
			Claim:           &claim,
			Proof:           nil,
			NumRelays:       0,
			NumComputeUnits: 0,
		}); err != nil {
			return settledResult, expiredResult, err
		}

		logger.Info("claim settled")

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)
		// Whether or not the proof is required, the supplier may have submitted one
		// so we need to delete it either way. If we don't have the if structure,
		// a safe error will be printed, but it can be confusing to the operator
		// or developer.
		if isProofFound {
			k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierOperatorAddress)
		}

		settledResult.NumClaims++
		settledResult.NumRelays += numClaimRelays
		settledResult.NumComputeUnits += numClaimComputeUnits
		settledResult.RelaysPerServiceMap[claim.SessionHeader.ServiceId] += numClaimRelays

		logger.Info(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	// Slash all the suppliers that have been marked for slashing slashingCount times.
	for supplierOperatorAddress, slashingCount := range supplierToExpiredClaimCount {
		if err := k.slashSupplierStake(ctx, supplierOperatorAddress, slashingCount); err != nil {
			logger.Error(fmt.Sprintf("error slashing supplier %s: %s", supplierOperatorAddress, err))
			return settledResult, expiredResult, err
		}
	}

	logger.Info(fmt.Sprintf(
		"settled %d and expired %d claims at block height %d",
		settledResult.NumClaims,
		expiredResult.NumClaims,
		blockHeight,
	))

	return settledResult, expiredResult, nil
}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim, err error) {
	blockHeight := ctx.BlockHeight()

	// NB: This error can be safely ignored as on-chain SharedQueryClient implementation cannot return an error.
	sharedParams, _ := k.sharedQuerier.GetParams(ctx)

	// expiringSessionEndHeight is the session end height of the session whose proof
	// window has most recently closed.
	expiringSessionEndHeight := blockHeight - int64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)+1)

	var nextKey []byte
	for {
		claimsRes, err := k.proofKeeper.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{
			Pagination: &query.PageRequest{
				Key: nextKey,
			},
			Filter: &prooftypes.QueryAllClaimsRequest_SessionEndHeight{
				SessionEndHeight: uint64(expiringSessionEndHeight),
			},
		})
		if err != nil {
			return nil, err
		}

		expiringClaims = append(expiringClaims, claimsRes.GetClaims()...)

		// Continue if there are more claims to fetch.
		nextKey = claimsRes.Pagination.GetNextKey()
		if nextKey != nil {
			continue
		}

		break
	}

	// Return the actually expiring claims
	return expiringClaims, nil
}

// proofRequirementForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_BLOCKER(@bryanchriswhite, #419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) proofRequirementForClaim(ctx sdk.Context, claim *prooftypes.Claim) (_ prooftypes.ProofRequirementReason, err error) {
	logger := k.logger.With("method", "proofRequirementForClaim")

	var requirementReason = prooftypes.ProofRequirementReason_NOT_REQUIRED

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		telemetry.ProofRequirementCounter(requirementReason, err)
	}()

	// NB: Assumption that claim is non-nil and has a valid root sum because it
	// is retrieved from the store and validated, on-chain, at time of creation.
	var numClaimComputeUnits uint64
	numClaimComputeUnits, err = claim.GetNumComputeUnits()
	if err != nil {
		return requirementReason, err
	}

	proofParams := k.proofKeeper.GetParams(ctx)
	tokenomicsParams := k.GetParams(ctx)

	// Proof requirement threshold should be checked against the claimed uPOKT since
	// the corresponding governance parameter represents coins and not compute units.
	claimeduPOKT, err := tokenomicsParams.NumComputeUnitsToCoin(numClaimComputeUnits)
	if err != nil {
		return requirementReason, err
	}

	// Require a proof if the claim's compute units meets or exceeds the threshold.
	//
	// TODO_BLOCKER(@bryanchriswhite, #419): This is just VERY BASIC placeholder logic to have something
	// in place while we implement proper probabilistic proofs. If you're reading it,
	// do not overthink it and look at the documents linked in #419.
	//
	// TODO_IMPROVE(@bryanchriswhite, @red-0ne): It might make sense to include
	// whether there was a proof submission error downstream from here. This would
	// require a more comprehensive metrics API.
	if claimeduPOKT.Amount.GTE(proofParams.GetProofRequirementThreshold().Amount) {
		requirementReason = prooftypes.ProofRequirementReason_THRESHOLD

		logger.Info(fmt.Sprintf(
			"claim requires proof due to claimed amount (%s) exceeding threshold (%s)",
			claimeduPOKT,
			proofParams.GetProofRequirementThreshold(),
		))
		return requirementReason, nil
	}

	earliestProofCommitBlockHash, err := k.getEarliestSupplierProofCommitBlockHash(ctx, claim)
	if err != nil {
		return requirementReason, err
	}

	proofRequirementSampleValue, err := claim.GetProofRequirementSampleValue(earliestProofCommitBlockHash)
	if err != nil {
		return requirementReason, err
	}

	// Require a proof probabilistically based on the proof_request_probability param.
	// NB: A random value between 0 and 1 will be less than or equal to proof_request_probability
	// with probability equal to the proof_request_probability.
	if proofRequirementSampleValue <= proofParams.GetProofRequestProbability() {
		requirementReason = prooftypes.ProofRequirementReason_PROBABILISTIC

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

// slashSupplierStake slashes the stake of a supplier slashingCount times and mints
// the total slashing amount to the tokenomics module account.
func (k Keeper) slashSupplierStake(
	ctx sdk.Context,
	supplierOperatorAddress string,
	slashingCount uint64,
) error {
	logger := k.logger.With("method", "slashSupplierStake")

	proofParams := k.proofKeeper.GetParams(ctx)
	slashingPenalty := proofParams.GetProofMissingPenalty()

	totalSlashingAmt := slashingPenalty.Amount.Mul(math.NewIntFromUint64(slashingCount))
	totalSlashingCoin := sdk.NewCoin(volatile.DenomuPOKT, totalSlashingAmt)

	supplierToSlash, supplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddress)
	if !supplierFound {
		return types.ErrTokenomicsSupplierNotFound
	}

	remainingStakeAmt := math.NewInt(0)
	if supplierToSlash.GetStake().Amount.GT(totalSlashingAmt) {
		remainingStakeAmt = supplierToSlash.GetStake().Amount.Sub(totalSlashingAmt)
	}

	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(totalSlashingCoin))
	if err != nil {
		return err
	}

	supplierToSlash.Stake.Amount = remainingStakeAmt

	logger.Warn(fmt.Sprintf(
		"slashing supplier %s stake by %s, remaining stake: %s",
		supplierToSlash.GetOperatorAddress(),
		totalSlashingCoin,
		remainingStakeAmt,
	))

	// Check if the supplier's stake is below the minimum and unstake it if necessary.
	// TODO_BETA(@red-0ne, #612): Use minimum stake governance parameter once available.
	// TODO_BETA(@red-0ne): Since SettlePendingClaims is not necessarily called
	// at session end height so the unstaked supplier may not be immediately removed.
	// Ensure that Gateways and Applications do not interact with a supplier that
	// is below the minimum stake.
	if remainingStakeAmt.LT(math.NewInt(1)) {
		sharedParams := k.sharedKeeper.GetParams(ctx)
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		currentHeight := sdkCtx.BlockHeight()
		unstakeSessionEndHeight := uint64(shared.GetSessionEndHeight(&sharedParams, currentHeight))

		logger.Warn(fmt.Sprintf(
			"unstaking supplier %s due to stake below the minimum",
			supplierOperatorAddress,
		))

		// TODO_CONSIDERATION: Should we just remove the supplier if the stake is
		// below the minimum, at the risk of making the off-chain actors have an
		// inconsistent session supplier list?
		supplierToSlash.UnstakeSessionEndHeight = unstakeSessionEndHeight

	}

	k.supplierKeeper.SetSupplier(ctx, supplierToSlash)

	// TODO_CONSIDERATION: Handle the case where the total slashing amount is
	// greater than the supplier's stake. The protocol could take the remaining
	// amount from the supplier's owner or operator balances.

	return nil
}

// getEarliestSupplierProofCommitBlockHash returns the block hash of the earliest
// block at which a claim might have its proof committed.
func (k Keeper) getEarliestSupplierProofCommitBlockHash(
	ctx context.Context,
	claim *prooftypes.Claim,
) (blockHash []byte, err error) {
	sharedParams, err := k.sharedQuerier.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	sessionEndHeight := claim.GetSessionHeader().GetSessionEndBlockHeight()
	supplierOperatorAddress := claim.GetSupplierOperatorAddress()

	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)
	proofWindowOpenBlockHash := k.sessionKeeper.GetBlockHash(ctx, proofWindowOpenHeight)

	// TODO_TECHDEBT: Update the method header of this function to accept (sharedParams, Claim, BlockHash).
	// After doing so, please review all calling sites and simplify them accordingly.
	earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddress,
	)

	return k.sessionKeeper.GetBlockHash(ctx, earliestSupplierProofCommitHeight), nil
}
