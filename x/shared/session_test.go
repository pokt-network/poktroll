package shared

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGetEarliestClaimCommitHeight_IsDeterministic(t *testing.T) {
	var (
		claimWindowOpenBlockHash [32]byte
		queryHeight              = int64(1)
		supplierAddr             = sample.AccAddress()
		sharedParams             = sharedtypes.DefaultParams()
	)

	test := func() int64 {
		return GetEarliestClaimCommitHeight(
			&sharedParams,
			queryHeight,
			claimWindowOpenBlockHash[:],
			supplierAddr,
		)
	}

	// Randomize queryHeight, claimWindowOpenBlockHash, and supplierAddr.
	for randomizeIdx := 0; randomizeIdx < 20; randomizeIdx++ {
		queryHeight = rand.Int63()

		supplierAddr = sample.AccAddress()

		_, err := rand.Read(claimWindowOpenBlockHash[:])
		require.NoError(t, err)

		expected := test()

		// Ensure consecutive calls are deterministic.
		for deterministicIdx := 0; deterministicIdx < 1000; deterministicIdx++ {
			require.Equalf(t, expected, test(), "on call number %d", deterministicIdx)
		}
	}
}

func TestGetEarliestProofCommitHeight_IsDeterministic(t *testing.T) {
	var (
		proofWindowOpenBlockHash [32]byte
		queryHeight              = int64(1)
		supplierAddr             = sample.AccAddress()
		sharedParams             = sharedtypes.DefaultParams()
	)

	test := func() int64 {
		return GetEarliestProofCommitHeight(
			&sharedParams,
			queryHeight,
			proofWindowOpenBlockHash[:],
			supplierAddr,
		)
	}

	// Randomize queryHeight, proofWindowOpenBlockHash, and supplierAddr.
	for randomizeIdx := 0; randomizeIdx < 20; randomizeIdx++ {
		queryHeight = rand.Int63()

		supplierAddr = sample.AccAddress()

		_, err := rand.Read(proofWindowOpenBlockHash[:])
		require.NoError(t, err)

		expected := test()

		// Ensure consecutive calls are deterministic.
		for deterministicIdx := 0; deterministicIdx < 1000; deterministicIdx++ {
			require.Equalf(t, expected, test(), "on call number %d", deterministicIdx)
		}
	}
}
