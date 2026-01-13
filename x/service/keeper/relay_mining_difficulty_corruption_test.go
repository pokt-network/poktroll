package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/service/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestRelayMiningDifficulty_RepairCorruptedDifficulties tests the upgrade repair logic
// that detects and fixes corrupted relay mining difficulties.
//
// Context: In v0.0.10, relay mining difficulties were moved from tokenomics to service module
// but the data migration was skipped, leaving services without proper difficulties initialized.
// On mainnet, this manifests as empty difficulty objects that break proof validation.
//
// This test verifies that the v0.1.31 upgrade repair logic:
// 1. Detects corrupted difficulties (empty service_id or target_hash)
// 2. Creates proper default difficulties with valid target hashes
// 3. Enables proof validation to work correctly
func TestRelayMiningDifficulty_RepairCorruptedDifficulties(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	targetNumRelays := k.GetParams(ctx).TargetNumRelays

	// Create test services
	services := []sharedtypes.Service{
		{Id: "gnosis", Name: "Gnosis", ComputeUnitsPerRelay: 1118, OwnerAddress: "pokt1owner"},
		{Id: "eth", Name: "Ethereum", ComputeUnitsPerRelay: 1579, OwnerAddress: "pokt1owner"},
		{Id: "base", Name: "Base", ComputeUnitsPerRelay: 1232, OwnerAddress: "pokt1owner"},
	}

	for _, svc := range services {
		k.SetService(ctx, svc)
	}

	// SCENARIO: Services exist but no difficulties were initialized
	// This is the state after v0.0.10 migration that skipped difficulty data

	// TEST 1: Verify difficulties don't exist initially
	t.Run("Services without initialized difficulties return defaults", func(t *testing.T) {
		for _, svc := range services {
			diff, found := k.GetRelayMiningDifficulty(ctx, svc.Id)

			// GetRelayMiningDifficulty returns a default when not found
			require.False(t, found, "difficulty should not be found for %s", svc.Id)
			require.Equal(t, svc.Id, diff.ServiceId, "default should have correct service_id")
			require.NotEmpty(t, diff.TargetHash, "default should have target_hash")
			require.Len(t, diff.TargetHash, 32, "default target_hash should be 32 bytes")
		}
	})

	// TEST 2: Simulate the upgrade repair logic from v0.1.31
	t.Run("Upgrade repair logic initializes missing difficulties", func(t *testing.T) {
		// This is the logic from app/upgrades/v0.1.31.go:initializeDifficultyHistory
		repairedCount := 0

		for _, svc := range services {
			diff, found := k.GetRelayMiningDifficulty(ctx, svc.Id)

			// Detect corrupted/missing difficulty
			isCorrupted := !found || diff.ServiceId == "" || len(diff.TargetHash) == 0

			if isCorrupted {
				// Create proper default difficulty
				diff = keeper.NewDefaultRelayMiningDifficulty(
					ctx,
					k.Logger(),
					svc.Id,
					targetNumRelays,
					targetNumRelays,
				)

				// Save the repaired difficulty
				k.SetRelayMiningDifficulty(ctx, diff)

				repairedCount++
			}
		}

		require.Equal(t, 3, repairedCount, "should have repaired all 3 services")

		// Verify all difficulties are now properly initialized
		for _, svc := range services {
			diff, found := k.GetRelayMiningDifficulty(ctx, svc.Id)

			require.True(t, found, "difficulty for %s should now be found", svc.Id)
			require.Equal(t, svc.Id, diff.ServiceId, "ServiceId should match")
			require.NotEmpty(t, diff.TargetHash, "TargetHash should not be empty")
			require.Len(t, diff.TargetHash, 32, "TargetHash should be 32 bytes (sha256)")
			require.Equal(t, targetNumRelays, diff.NumRelaysEma, "NumRelaysEma should equal targetNumRelays")
			require.Equal(t, sdkCtx.BlockHeight(), diff.BlockHeight, "BlockHeight should be set")
		}
	})

	// TEST 3: Verify repaired difficulties pass proof validation requirements
	t.Run("Repaired difficulties meet proof validation requirements", func(t *testing.T) {
		for _, svc := range services {
			diff, found := k.GetRelayMiningDifficulty(ctx, svc.Id)

			require.True(t, found, "difficulty for %s should be found", svc.Id)

			// This is the critical check from proof_validation.go:validateRelayDifficulty (line 472)
			// Without this fix, proof validation fails with:
			// "invalid RelayDifficultyTargetHash: length wanted: 32; got: 0"
			require.Len(t, diff.TargetHash, 32,
				"TargetHash must be 32 bytes for proof validation (service: %s)", svc.Id)

			// Verify other required fields for proof validation
			require.NotEmpty(t, diff.ServiceId, "ServiceId required for difficulty lookups")
			require.Greater(t, diff.NumRelaysEma, uint64(0), "NumRelaysEma required for difficulty adjustments")
		}
	})

	// TEST 4: Verify GetAllRelayMiningDifficulty returns initialized difficulties
	t.Run("GetAllRelayMiningDifficulty returns valid difficulties after repair", func(t *testing.T) {
		allDifficulties := k.GetAllRelayMiningDifficulty(ctx)

		require.Len(t, allDifficulties, 3, "should have 3 difficulty entries")

		// Verify none are corrupted/empty
		for i, diff := range allDifficulties {
			require.NotEmpty(t, diff.ServiceId, "difficulty[%d].ServiceId should be populated", i)
			require.Len(t, diff.TargetHash, 32, "difficulty[%d].TargetHash should be 32 bytes", i)
			require.Greater(t, diff.NumRelaysEma, uint64(0), "difficulty[%d].NumRelaysEma should be > 0", i)
		}
	})
}

// TestRelayMiningDifficulty_MixedValidAndMissingDifficulties tests the upgrade repair logic
// when some services have valid difficulties and others don't.
func TestRelayMiningDifficulty_MixedValidAndMissingDifficulties(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	cosmostypes.UnwrapSDKContext(ctx)
	targetNumRelays := k.GetParams(ctx).TargetNumRelays

	// Create test services
	services := []sharedtypes.Service{
		{Id: "missing1", Name: "Missing Service 1", ComputeUnitsPerRelay: 1000, OwnerAddress: "pokt1owner"},
		{Id: "valid1", Name: "Valid Service 1", ComputeUnitsPerRelay: 2000, OwnerAddress: "pokt1owner"},
		{Id: "missing2", Name: "Missing Service 2", ComputeUnitsPerRelay: 3000, OwnerAddress: "pokt1owner"},
	}

	for _, svc := range services {
		k.SetService(ctx, svc)
	}

	// Initialize difficulty for "valid1" only
	validDiff := keeper.NewDefaultRelayMiningDifficulty(ctx, k.Logger(), "valid1", targetNumRelays, targetNumRelays)
	k.SetRelayMiningDifficulty(ctx, validDiff)

	// Apply upgrade repair logic
	repairedCount := 0
	for _, svc := range services {
		diff, found := k.GetRelayMiningDifficulty(ctx, svc.Id)
		isCorrupted := !found || diff.ServiceId == "" || len(diff.TargetHash) == 0

		if isCorrupted {
			diff = keeper.NewDefaultRelayMiningDifficulty(ctx, k.Logger(), svc.Id, targetNumRelays, targetNumRelays)
			k.SetRelayMiningDifficulty(ctx, diff)
			repairedCount++
		}
	}

	require.Equal(t, 2, repairedCount, "should have repaired exactly 2 missing difficulties")

	// Verify all difficulties are now valid
	allDifficulties := k.GetAllRelayMiningDifficulty(ctx)
	require.Len(t, allDifficulties, 3, "should have 3 difficulty entries")

	for _, diff := range allDifficulties {
		require.NotEmpty(t, diff.ServiceId, "all ServiceIds should be populated")
		require.Len(t, diff.TargetHash, 32, "all TargetHashes should be 32 bytes")
		require.Greater(t, diff.NumRelaysEma, uint64(0), "all NumRelaysEma should be > 0")
	}
}
