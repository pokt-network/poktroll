package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
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
	numClaimsSettled, numClaimsExpired uint64,
	relaysPerServiceMap map[string]uint64,
	computeUnitsPerServiceMap map[string]uint64,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing expiringClaims appropriately
	// and only retrieving the expiringClaims that need to be settled rather than all
	// of them and iterating through them one by one.
	expiringClaims := k.getExpiringClaims(ctx)

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	relaysPerServiceMap = make(map[string]uint64)
	computeUnitsPerServiceMap = make(map[string]uint64)

	for _, claim := range expiringClaims {

		// NB: Note that not every (Req, Res) pair in the session is inserted in
		// the tree for scalability reasons. This is the count of non-empty leaves
		// that matched the necessary difficulty and is therefore an estimation
		// of the total number of relays serviced and work done.
		numClaimComputeUnits, err := claim.GetNumComputeUnits()
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		numRelaysInSessionTree, err := claim.GetNumRelays()
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		sessionId := claim.SessionHeader.SessionId

		_, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		isProofRequiredForClaim, err := k.isProofRequiredForClaim(ctx, &claim)
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}
		if isProofRequiredForClaim {
			// If a proof is not found, the claim will expire and never be settled.
			if !isProofFound {
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:        &claim,
					ComputeUnits: numClaimComputeUnits,
				}
				if err := ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
				}
				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)

				numClaimsExpired++
				continue
			}
			// NB: If a proof is found, it is valid because verification is done
			// at the time of submission.
		}

		// Manage the mint & burn accounting for the claim.
		if err := k.SettleSessionAccounting(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error settling session accounting for claim %q: %v", claim.SessionHeader.SessionId, err))
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:         &claim,
			ComputeUnits:  numClaimComputeUnits,
			ProofRequired: isProofRequiredForClaim,
		}
		if err := ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)
		// Whether or not the proof is required, the supplier may have submitted one
		// so we need to delete it either way. If we don't have the if structure,
		// a safe error will be printed, but it can be confusing to the operator
		// or developer.
		if isProofFound {
			k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierAddress)
		}

		relaysPerServiceMap[claim.SessionHeader.Service.Id] += numRelaysInSessionTree
		computeUnitsPerServiceMap[claim.SessionHeader.Service.Id] += numClaimComputeUnits

		numClaimsSettled++
		logger.Info(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	logger.Info(fmt.Sprintf("settled %d and expired %d claims at block height %d", numClaimsSettled, numClaimsExpired, blockHeight))

	return numClaimsSettled, numClaimsExpired, relaysPerServiceMap, computeUnitsPerServiceMap, nil
}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim) {
	blockHeight := ctx.BlockHeight()

	// TODO_TECHDEBT: Optimize this by indexing claims appropriately
	// and only retrieving the claims that need to be settled rather than all
	// of them and iterating through them one by one.
	claims := k.proofKeeper.GetAllClaims(ctx)

	// Loop over all claims we need to check for expiration
	for _, claim := range claims {
		claimSessionStartHeight := claim.GetSessionHeader().GetSessionStartBlockHeight()
		expirationHeight := k.sharedKeeper.GetProofWindowCloseHeight(ctx, claimSessionStartHeight)
		if blockHeight >= expirationHeight {
			expiringClaims = append(expiringClaims, claim)
		}
	}

	// Return the actually expiring claims
	return expiringClaims
}

// isProofRequiredForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_BLOCKER(@bryanchriswhite, #419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) isProofRequiredForClaim(ctx sdk.Context, claim *prooftypes.Claim) (_ bool, err error) {
	logger := k.logger.With("method", "isProofRequiredForClaim")

	var requirementReason = telemetry.ProofNotRequired

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		telemetry.ProofRequirementCounter(requirementReason, err)
	}()

	// NB: Assumption that claim is non-nil and has a valid root sum because it
	// is retrieved from the store and validated, on-chain, at time of creation.
	claimComputeUnits, err := claim.GetNumComputeUnits()
	if err != nil {
		return true, err
	}

	proofParams := k.proofKeeper.GetParams(ctx)

	// Require a proof if the claim's compute units meets or exceeds the threshold.
	//
	// TODO_BLOCKER(@bryanchriswhite, #419): This is just VERY BASIC placeholder logic to have something
	// in place while we implement proper probabilistic proofs. If you're reading it,
	// do not overthink it and look at the documents linked in #419.
	//
	// TODO_IMPROVE(@bryanchriswhite, @red-0ne): It might make sense to include
	// whether there was a proof submission error downstream from here. This would
	// require a more comprehensive metrics API.
	if claimComputeUnits >= proofParams.GetProofRequirementThreshold() {
		requirementReason = telemetry.ProofRequirementReasonThreshold

		logger.Info(fmt.Sprintf(
			"claim requires proof due to compute units (%d) exceeding threshold (%d)",
			claimComputeUnits,
			proofParams.GetProofRequirementThreshold(),
		))
		return true, nil
	}

	claimSeed, err := computeClaimSeed(claim)
	if err != nil {
		return true, err
	}

	// Sample a pseudo-random value between 0 and 1 to determine if a proof is required probabilistically.
	randFloat, err := randProbability(claimSeed)
	if err != nil {
		return true, err
	}

	// Require a proof probabilistically based on the proof_request_probability param.
	// NB: A random value between 0 and 1 will be less than or equal to proof_request_probability
	// with probability equal to the proof_request_probability.
	if randFloat <= proofParams.GetProofRequestProbability() {
		requirementReason = telemetry.ProofRequirementReasonProbabilistic

		logger.Info(fmt.Sprintf(
			"claim requires proof due to random sample (%.2f) being less than or equal to probability (%.2f)",
			randFloat,
			proofParams.GetProofRequestProbability(),
		))
		return true, nil
	}

	logger.Info(fmt.Sprintf(
		"claim does not require proof due to compute units (%d) being less than the threshold (%d) and random sample (%.2f) being greater than probability (%.2f)",
		claimComputeUnits,
		proofParams.GetProofRequirementThreshold(),
		randFloat,
		proofParams.GetProofRequestProbability(),
	))
	return false, nil
}

// computeClaimSeed returns an int64 intended for seeding the math/rand pseudo-random
// number generator. The seed is derived by truncating the serialized claim's hash to
// the first 8 bytes and converting it to an int64.
func computeClaimSeed(claim *prooftypes.Claim) (int64, error) {
	var seedBz [8]byte

	claimHash, err := computeClaimHash(claim)
	if err != nil {
		return 0, err
	}

	copy(seedBz[:], claimHash[:])

	// Convert the seed bytes to an int64.
	// NB: little endian is more conventional in this context.
	seed := int64(binary.LittleEndian.Uint64(seedBz[:]))

	return seed, nil
}

// computeClaimHash returns the SHA-256 hash of the serialized claim.
func computeClaimHash(claim *prooftypes.Claim) ([32]byte, error) {
	claimBz, err := claim.Marshal()
	if err != nil {
		return [32]byte{}, err
	}

	return sha256.Sum256(claimBz), nil
}

// randProbability generates a pseudo random float32 between 0 and 1 deterministically given a seed.
//
// TODO_MAINNET: To support other language implementations of the protocol, the
// pseudo-random number generator used here should be language-agnostic (i.e. not
// golang specific).
func randProbability(seed int64) (float32, error) {
	// Construct a pseudo-random number generator with the seed.
	pseudoRand := rand.New(rand.NewSource(seed))

	// Generate a random uint32.
	randUint32 := pseudoRand.Uint32()

	// Clamp the random float32 between [0,1]. This is achieved by dividing the random uint32
	// by the most significant digit of a float32, which is 2^32, guaranteeing an output between
	// 0 and 1, inclusive.
	oneMostSignificantDigitFloat32 := float32(1 << 32)
	randClampedFloat32 := float32(randUint32) / oneMostSignificantDigitFloat32

	return randClampedFloat32, nil
}
