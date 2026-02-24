package keeper

import (
	"testing"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/proof/mocks"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestKeeper_GetProofRequirementSeedBlockHash_UsesHistoricalParams verifies that
// getProofRequirementSeedBlockHash uses shared params at sessionEndHeight (historical)
// rather than current params.
//
// If shared params change between a session's end and the proof window,
// the seed block hash must still be calculated using the params that were
// effective when the session ended, to maintain consistency with other on-chain
// proof validation (validateProofWindow, validateClosestPath in EndBlocker).
//
// This is a regression test for the bug where getProofRequirementSeedBlockHash
// called k.sharedQuerier.GetParams(ctx) (current params) instead of
// k.sharedKeeper.GetParamsAtHeight(ctx, sessionEndHeight) (historical params).
func TestKeeper_GetProofRequirementSeedBlockHash_UsesHistoricalParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)

	// Build a minimal keeper with only the dependencies needed by
	// getProofRequirementSeedBlockHash: sharedKeeper and sessionKeeper.
	k := Keeper{
		sharedKeeper:  mockSharedKeeper,
		sessionKeeper: mockSessionKeeper,
		logger:        log.NewNopLogger(),
	}

	supplierAddr := sample.AccAddressBech32()
	sessionEndHeight := int64(10)

	claim := prooftypes.Claim{
		SupplierOperatorAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			SessionEndBlockHeight: sessionEndHeight,
		},
	}

	// --- Scenario 1: default params ---
	// Historical params at sessionEndHeight = default params.
	defaultParams := sharedtypes.DefaultParams()

	// Block hashes — distinct per height so we can distinguish them.
	blockHashA := []byte("block_hash_A_proof_window_open_default")
	seedBlockHashDefault := []byte("seed_block_hash_default")

	// Expect GetParamsAtHeight to be called with sessionEndHeight.
	mockSharedKeeper.EXPECT().
		GetParamsAtHeight(gomock.Any(), sessionEndHeight).
		Return(defaultParams)

	// Compute the expected proof window open height with default params.
	proofWindowOpenHeightDefault := sharedtypes.GetProofWindowOpenHeight(&defaultParams, sessionEndHeight)

	// GetBlockHash is called twice:
	// 1. For proofWindowOpenHeight (to get the block hash used as seed for random offset)
	// 2. For earliestSupplierProofCommitHeight - 1 (the actual seed block hash)
	mockSessionKeeper.EXPECT().
		GetBlockHash(gomock.Any(), proofWindowOpenHeightDefault).
		Return(blockHashA)

	// The earliest proof commit height depends on a random offset seeded by
	// (proofWindowOpenBlockHash, supplierAddr). We capture the exact height
	// by computing it ourselves.
	earliestCommitHeightDefault := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&defaultParams, sessionEndHeight, blockHashA, supplierAddr,
	)
	mockSessionKeeper.EXPECT().
		GetBlockHash(gomock.Any(), earliestCommitHeightDefault-1).
		Return(seedBlockHashDefault)

	ctx := cosmostypes.Context{}.WithBlockHeight(proofWindowOpenHeightDefault + 1)
	resultDefault, err := k.getProofRequirementSeedBlockHash(ctx, &claim)
	require.NoError(t, err)
	require.Equal(t, seedBlockHashDefault, resultDefault)

	// --- Scenario 2: current params changed, but historical params unchanged ---
	// After a governance update, current ProofWindowOpenOffsetBlocks is larger.
	// The function must still use historical params at sessionEndHeight.
	modifiedParams := defaultParams
	modifiedParams.ProofWindowOpenOffsetBlocks = defaultParams.ProofWindowOpenOffsetBlocks + 5

	// GetParamsAtHeight should still return DEFAULT (historical) params.
	mockSharedKeeper.EXPECT().
		GetParamsAtHeight(gomock.Any(), sessionEndHeight).
		Return(defaultParams)

	// Same expectations as scenario 1, because historical params are the same.
	mockSessionKeeper.EXPECT().
		GetBlockHash(gomock.Any(), proofWindowOpenHeightDefault).
		Return(blockHashA)
	mockSessionKeeper.EXPECT().
		GetBlockHash(gomock.Any(), earliestCommitHeightDefault-1).
		Return(seedBlockHashDefault)

	resultAfterParamChange, err := k.getProofRequirementSeedBlockHash(ctx, &claim)
	require.NoError(t, err)

	// The seed block hash MUST be identical because getProofRequirementSeedBlockHash
	// uses historical params (unchanged for this session), not current params.
	require.Equal(t, resultDefault, resultAfterParamChange,
		"seed block hash must be identical when historical params are unchanged, "+
			"regardless of current param changes",
	)

	// --- Verify: using current (modified) params would produce different heights ---
	proofWindowOpenHeightModified := sharedtypes.GetProofWindowOpenHeight(&modifiedParams, sessionEndHeight)
	require.NotEqual(t, proofWindowOpenHeightDefault, proofWindowOpenHeightModified,
		"modified params should produce a different proof window open height, "+
			"confirming the test is meaningful",
	)
}
