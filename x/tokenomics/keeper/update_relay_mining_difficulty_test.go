package keeper_test

import (
	"bytes"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// This is a "base" test for updating relay mining difficulty to go through
// a flow testing a few different scenarios, but does not cover the full range
// of edge or use cases.
func TestUpdateRelayMiningDifficulty_Base(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Introduce svc1 for the first time
	relaysPerServiceMap := map[string]uint64{
		"svc1": 1e3, // new service
	}
	_, err := keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	// The first time svc1 difficulty is updated, the relay EMA will be equal
	// to the first value provided.
	difficultySvc11, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	require.Equal(t, uint64(1e3), difficultySvc11.NumRelaysEma)

	// Update svc1 and introduce svc2 for the first time
	relaysPerServiceMap = map[string]uint64{
		"svc1": 1e10, // higher than the first value above
		"svc2": 1e5,  // new service
	}
	_, err = keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	difficultySvc12, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	// Verify that the svc1 relay ema is strictly higher than the first value
	// above, but strictly lower than the second value because of the rolling average.
	require.Greater(t, difficultySvc12.NumRelaysEma, difficultySvc11.NumRelaysEma)
	require.Less(t, difficultySvc12.NumRelaysEma, uint64(1e10))
	// Because the number of relays went up, there are more leading zeroes in the
	// target hash, so the number is lower than it was before.
	require.Less(t, difficultySvc12.TargetHash, difficultySvc11.TargetHash)

	// The first time svc2 difficulty is updated, so the num relays ema is
	// equal to the first value provided.
	difficultySvc21, found := keeper.GetRelayMiningDifficulty(ctx, "svc2")
	require.True(t, found)
	require.Equal(t, uint64(1e5), difficultySvc21.NumRelaysEma)

	// Update svc1 and svc2 and introduce svc3 for the first time
	relaysPerServiceMap = map[string]uint64{
		"svc1": 1e12, // higher than the second value above
		"svc2": 1e2,  // lower than the first value above
		"svc3": 1e10, // new service
	}
	_, err = keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	// svc1 relays went up so the target hash is now a smaller number (more leading zeroes)
	// because the difficulty is higher.
	difficultySvc13, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	require.Greater(t, difficultySvc13.NumRelaysEma, difficultySvc12.NumRelaysEma)
	require.Less(t, difficultySvc13.TargetHash, difficultySvc12.TargetHash)

	// svc2 relay ema went down so the target hash is now a larger number (less leading zeroes)
	difficultySvc22, found := keeper.GetRelayMiningDifficulty(ctx, "svc2")
	require.True(t, found)
	require.Less(t, difficultySvc22.NumRelaysEma, difficultySvc21.NumRelaysEma)
	// Since the relays EMA is lower than the target, the difficulty hash is all 1s
	require.Less(t, difficultySvc22.NumRelaysEma, tokenomicskeeper.TargetNumRelays)
	require.Equal(t, difficultySvc22.TargetHash, makeBytesFullOfOnes(32))

	// svc3 is new so the relay ema is equal to the first value provided
	difficultySvc31, found := keeper.GetRelayMiningDifficulty(ctx, "svc3")
	require.True(t, found)
	require.Equal(t, uint64(1e10), difficultySvc31.NumRelaysEma)

	// Confirm a relay mining difficulty update event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventRelayMiningDifficultyUpdated](t,
		events, "poktroll.tokenomics.EventRelayMiningDifficultyUpdated")
	require.Len(t, expectedEvents, 6) // 3 for svc1, 2 for svc2, 1 for svc3
}

func TestUpdateRelayMiningDifficulty_FirstDifficulty(t *testing.T) {
	tests := []struct {
		desc                          string
		numRelays                     uint64
		expectedRelayMiningDifficulty types.RelayMiningDifficulty
	}{
		{
			desc:      "First Difficulty way below target",
			numRelays: keeper.TargetNumRelays / 1e3,
			expectedRelayMiningDifficulty: types.RelayMiningDifficulty{
				ServiceId:    "svc1",
				BlockHeight:  1,
				NumRelaysEma: keeper.TargetNumRelays / 1e3,
				TargetHash:   defaultDifficulty(), // default difficulty without any leading 0 bits
			},
		}, {
			desc:      "First Difficulty equal to target",
			numRelays: keeper.TargetNumRelays,
			expectedRelayMiningDifficulty: types.RelayMiningDifficulty{
				ServiceId:    "svc1",
				BlockHeight:  1,
				NumRelaysEma: keeper.TargetNumRelays,
				TargetHash:   defaultDifficulty(), // default difficulty without any leading 0 bits
			},
		}, {
			desc:      "First Difficulty way above target",
			numRelays: keeper.TargetNumRelays * 1e3,
			expectedRelayMiningDifficulty: types.RelayMiningDifficulty{
				ServiceId:    "svc1",
				BlockHeight:  1,
				NumRelaysEma: keeper.TargetNumRelays * 1e3,
				TargetHash: append(
					[]byte{0b00000000}, // at least 8 leading 0 bits
					makeBytesFullOfOnes(31)...,
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			keeper, ctx := keepertest.TokenomicsKeeper(t)
			relaysPerServiceMap := map[string]uint64{
				"svc1": tt.numRelays,
			}
			_, err := keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
			require.NoError(t, err)

			relayDifficulty, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
			require.True(t, found)

			require.Equal(t, tt.numRelays, relayDifficulty.NumRelaysEma)
			require.Equal(t, tt.expectedRelayMiningDifficulty.NumRelaysEma, relayDifficulty.NumRelaysEma)

			// NB: An increase in difficulty is indicated by a decrease in the target hash
			didDifficultyIncrease := bytes.Compare(relayDifficulty.TargetHash, tt.expectedRelayMiningDifficulty.TargetHash) < 1
			require.True(t, didDifficultyIncrease,
				"expected difficulty.TargetHash (%x) to be less than or equal to expectedRelayMiningDifficulty.TargetHash (%x)",
				relayDifficulty.TargetHash, tt.expectedRelayMiningDifficulty.TargetHash,
			)
		})
	}
}

func TestComputeNewDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                        string
		numRelaysTarget             uint64
		relaysEma                   uint64
		expectedRelayDifficultyHash []byte
	}{
		{
			desc:                        "Relays Target > Relays EMA",
			numRelaysTarget:             100,
			relaysEma:                   50,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:                        "Relays Target == Relays EMA",
			numRelaysTarget:             100,
			relaysEma:                   100,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:            "Relays Target < Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       100,
			expectedRelayDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       200,
			expectedRelayDifficultyHash: append(
				[]byte{0b00111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       1000,
			expectedRelayDifficultyHash: append(
				[]byte{0b00001111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       10000,
			expectedRelayDifficultyHash: append(
				[]byte{0b00000001},
				makeBytesFullOfOnes(31)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			newRelayDifficultyTargetHash := keeper.ComputeNewDifficultyTargetHash(prooftypes.DefaultRelayDifficultyTargetHash, tt.numRelaysTarget, tt.relaysEma)

			// NB: An increase in difficulty is indicated by a decrease in the target hash
			didDifficultyIncrease := bytes.Compare(newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash) < 1
			require.True(t, didDifficultyIncrease,
				"expected difficulty.TargetHash (%x) to be less than or equal to expectedRelayMiningDifficulty.TargetHash (%x)",
				newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash,
			)
		})
	}
}

func makeBytesFullOfOnes(length int) []byte {
	output := make([]byte, length)
	for i := range output {
		output[i] = 0b11111111
	}
	return output
}

func defaultDifficulty() []byte {
	return makeBytesFullOfOnes(32)
}
