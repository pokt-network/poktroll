package keeper_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IN_THIS_PR: Finish this test and consider others we should add
func TestUpdateRelayMiningDifficulty_General(t *testing.T) {
	keeper, ctx := keepertest.TokenomicsKeeper(t)

	relaysPerServiceMap := map[string]uint64{
		"svc1": 1e3,
	}
	err := keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	difficultySvc1, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	require.Equal(t, uint64(1e3), difficultySvc1.NumRelaysEma)

	relaysPerServiceMap = map[string]uint64{
		"svc1": 1e10,
		"svc2": 1e5,
	}
	err = keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	difficultySvc1, found = keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	// require.Equal(t, uint64(1e10), difficultySvc1.NumRelaysEma)

	difficultySvc2, found := keeper.GetRelayMiningDifficulty(ctx, "svc2")
	require.True(t, found)
	require.Equal(t, uint64(1e10), difficultySvc2.NumRelaysEma)

	relaysPerServiceMap = map[string]uint64{
		"svc1": 1e10,
		"svc2": 1e5,
		"svc3": 1e10,
	}
	err = keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
	require.NoError(t, err)

	difficultySvc1, found = keeper.GetRelayMiningDifficulty(ctx, "svc1")
	require.True(t, found)
	require.Equal(t, uint64(1e10), difficultySvc1.NumRelaysEma)

	difficultySvc2, found = keeper.GetRelayMiningDifficulty(ctx, "svc2")
	require.True(t, found)
	require.Equal(t, uint64(1e10), difficultySvc2.NumRelaysEma)

	difficultySvc3, found := keeper.GetRelayMiningDifficulty(ctx, "svc3")
	require.True(t, found)
	require.Equal(t, uint64(1e10), difficultySvc3.NumRelaysEma)
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
					[]byte{0b00000000, 0b01111111}, // 9 leading 0 bits
					makeBytesFullOfOnes(30)...,
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
			err := keeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
			require.NoError(t, err)

			difficulty, found := keeper.GetRelayMiningDifficulty(ctx, "svc1")
			require.True(t, found)

			require.Equal(t, difficulty.NumRelaysEma, tt.numRelays)
			require.Equal(t, difficulty.NumRelaysEma, tt.expectedRelayMiningDifficulty.NumRelaysEma)

			require.Equal(t, difficulty.TargetHash, tt.expectedRelayMiningDifficulty.TargetHash)
		})
	}
}

func TestComputeNewDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                   string
		numRelaysTarget        uint64
		relaysEma              uint64
		expectedDifficultyHash []byte
	}{
		{
			desc:                   "Relays Target > Relays EMA",
			numRelaysTarget:        100,
			relaysEma:              50,
			expectedDifficultyHash: defaultDifficulty(),
		},
		{
			desc:                   "Relays Target == Relays EMA",
			numRelaysTarget:        100,
			relaysEma:              100,
			expectedDifficultyHash: defaultDifficulty(),
		},
		{
			desc:            "Relays Target < Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       100,
			expectedDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       200,
			expectedDifficultyHash: append(
				[]byte{0b00111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       1000,
			expectedDifficultyHash: append(
				[]byte{0b00001111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       10000,
			expectedDifficultyHash: append(
				[]byte{0b00000001},
				makeBytesFullOfOnes(31)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := keeper.ComputeNewDifficultyTargetHash(tt.numRelaysTarget, tt.relaysEma)
			require.Equal(t, result, tt.expectedDifficultyHash)
		})
	}
}

func TestLeadingZeroBitsToTargetDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                   string
		numLeadingZeroBits     int
		numBytes               int
		expectedDifficultyHash []byte
	}{
		{
			desc:                   "0 leading 0 bits in 1 byte",
			numLeadingZeroBits:     0,
			numBytes:               1,
			expectedDifficultyHash: []byte{0b11111111},
		},
		{
			desc:               "full zero bytes (16 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 16,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			desc:               "partial byte (20 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 20,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000, 0b00001111},
				makeBytesFullOfOnes(29)...,
			),
		},
		{
			desc:               "another partial byte (10 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 10,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00111111},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			desc:               "edge case 1 bit (1 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 1,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:               "exact byte boundary (24 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 24,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000, 0b00000000},
				makeBytesFullOfOnes(29)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := keeper.LeadingZeroBitsToTargetDifficultyHash(tt.numLeadingZeroBits, tt.numBytes)
			if !bytes.Equal(result, tt.expectedDifficultyHash) {
				t.Errorf("got %x, expected %x", result, tt.expectedDifficultyHash)
			}
		})
	}
}

func makeBytesFullOfOnes(length int) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = 0b11111111
	}
	return result
}

func defaultDifficulty() []byte {
	return makeBytesFullOfOnes(32)
}
