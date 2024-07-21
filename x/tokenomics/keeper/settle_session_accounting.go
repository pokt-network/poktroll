package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettleSessionAccounting is responsible for all of the post-session accounting
// necessary to burn, mint or transfer tokens depending on the amount of work
// done. The amount of "work done" complete is dictated by `sum` of `root`.
//
// ASSUMPTION: It is assumed the caller of this function validated the claim
// against a proof BEFORE calling this function.
//
// TODO_MAINNET(@Olshansk): Research if there's a way to limit who can call this function?
func (k Keeper) SettleSessionAccounting(
	ctx context.Context,
	claim *prooftypes.Claim,
) error {
	logger := k.Logger().With("method", "SettleSessionAccounting")

	settlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
	isSuccessful := false
	// This is emitted only when the function returns.
	defer telemetry.EventSuccessCounter(
		"settle_session_accounting",
		func() float32 { return float32(settlementCoin.Amount.Int64()) },
		func() bool { return isSuccessful },
	)

	// Make sure the claim is not nil
	if claim == nil {
		logger.Error("received a nil claim")
		return tokenomicstypes.ErrTokenomicsClaimNil
	}

	// Retrieve & validate the session header
	sessionHeader := claim.GetSessionHeader()
	if sessionHeader == nil {
		logger.Error("received a nil session header")
		return tokenomicstypes.ErrTokenomicsSessionHeaderNil
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return tokenomicstypes.ErrTokenomicsSessionHeaderInvalid
	}

	supplierAddr, err := cosmostypes.AccAddressFromBech32(claim.GetSupplierAddress())
	if err != nil || supplierAddr == nil {
		return tokenomicstypes.ErrTokenomicsSupplierAddressInvalid
	}

	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid
	}

	// Retrieve the sum of the root as a proxy into the amount of work done
	root := (smt.MerkleRoot)(claim.GetRootHash())

	// TODO_BLOCKER(@Olshansk): This check should be the responsibility of the SMST package
	// since it's used to get compute units from the root hash.
	if root == nil || len(root) != smt.SmstRootSizeBytes {
		logger.Error(fmt.Sprintf("received an invalid root hash of size: %d", len(root)))
		return tokenomicstypes.ErrTokenomicsRootHashInvalid
	}
	claimComputeUnits := root.Sum()

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"compute_units", claimComputeUnits,
		"session_id", sessionHeader.GetSessionId(),
		"supplier", supplierAddr,
		"application", applicationAddress,
	)

	logger.Info("About to start session settlement accounting")

	// Retrieve the staked application record
	application, foundApplication := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !foundApplication {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the staked application record
	supplier, foundSupplier := k.supplierKeeper.GetSupplier(ctx, supplierAddr.String())
	if !foundSupplier {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	logger.Info(fmt.Sprintf("About to start settling claim for %d compute units", claimComputeUnits))

	// Calculate the amount of tokens to mint & burn
	settlementCoin = k.getCoinFromComputeUnits(ctx, root)
	settlementCoins := cosmostypes.NewCoins(settlementCoin)

	logger.Info(fmt.Sprintf(
		"%d compute units equate to %s for session %s",
		claimComputeUnits,
		settlementCoin,
		sessionHeader.SessionId,
	))

	if err := k.TLMRelayBurnEqualsMint(ctx, &application, &supplier, settlementCoins); err != nil {
		logger.Warn(fmt.Sprintf("failed to trigger the token-logic-module settle session accounting", supplierAddr))
		return err
	}

	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated stake for application with address %q to %s", applicationAddress, application.Stake))

	isSuccessful = true
	return nil
}

// getCoinFromComputeUnits calculates the amount of tokens to mint based on the
// sum of the merkle tree.
func (k Keeper) getCoinFromComputeUnits(ctx context.Context, root smt.MerkleRoot) cosmostypes.Coin {
	// Retrieve the existing tokenomics params
	params := k.GetParams(ctx)
	merkleRootSum := root.Sum()
	cuttm := params.ComputeUnitsToTokensMultiplier
	upokt := math.NewInt(int64(merkleRootSum * cuttm))
	return cosmostypes.NewCoin("upokt", upokt)
}
