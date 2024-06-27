package shared

import (
	"math/rand"
	"sync"
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
		return GetEarliestSupplierClaimCommitHeight(
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
		sharedParams             = sharedtypes.DefaultParams()
	)

	test := func(queryHeight int64, supplierAddr string) int64 {
		return GetEarliestSupplierProofCommitHeight(
			&sharedParams,
			queryHeight,
			proofWindowOpenBlockHash[:],
			supplierAddr,
		)
	}

	wg := sync.WaitGroup{}
	wg.Add(1000)
	for randomizeIdx := 0; randomizeIdx < 1000; randomizeIdx++ {
		// NB: sample concurrently to save time.
		go func() {
			// Randomize queryHeight, proofWindowOpenBlockHash, and supplierAddr.
			queryHeight := rand.Int63()
			supplierAddr := sample.AccAddress()
			_, err := rand.Read(proofWindowOpenBlockHash[:])
			require.NoError(t, err)

			// Gompute expected value.
			expected := test(queryHeight, supplierAddr)

			// Ensure consecutive calls are deterministic.
			wg.Add(1000)
			for deterministicIdx := 0; deterministicIdx < 1000; deterministicIdx++ {
				// NB: sample concurrently to save time.
				go func() {

					require.Equalf(t, expected, test(queryHeight, supplierAddr), "on call number %d", deterministicIdx)
					wg.Done()
				}()
			}
			wg.Wait()
			wg.Done()
		}()
	}
}

func TestClaimProofWindows(t *testing.T) {
	var blockHash []byte

	// NB: arbitrary sample size intended to be large enough to
	sampleSize := 15000

	tests := []struct {
		desc         string
		sharedParams sharedtypes.Params
		queryHeight  int64
	}{
		{
			desc:         "default params",
			sharedParams: sharedtypes.DefaultParams(),
			queryHeight:  int64(1),
		},
		{
			desc: "minimal windows",
			sharedParams: sharedtypes.Params{
				NumBlocksPerSession:          1,
				ClaimWindowOpenOffsetBlocks:  0,
				ClaimWindowCloseOffsetBlocks: 1,
				ProofWindowOpenOffsetBlocks:  0,
				ProofWindowCloseOffsetBlocks: 1,
			},
			queryHeight: int64(1),
		},
	}

	wg := sync.WaitGroup{}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			wg.Add(sampleSize)
			for i := 0; i < sampleSize; i++ {
				// NB: sample concurrently to save time.
				go func() {
					// Randomize the supplier address for each sample.
					// This will produce different randomized earliest claim & proof offsets.
					supplierAddr := sample.AccAddress()

					claimWindowOpenHeight := GetClaimWindowOpenHeight(&test.sharedParams, test.queryHeight)
					claimWindowCloseHeight := GetClaimWindowCloseHeight(&test.sharedParams, test.queryHeight)

					require.Greater(t, claimWindowCloseHeight, claimWindowOpenHeight)

					proofWindowOpenHeight := GetProofWindowOpenHeight(&test.sharedParams, test.queryHeight)
					proofWindowCloseHeight := GetProofWindowCloseHeight(&test.sharedParams, test.queryHeight)

					require.GreaterOrEqual(t, proofWindowOpenHeight, claimWindowCloseHeight)
					require.Greater(t, proofWindowCloseHeight, proofWindowOpenHeight)

					earliestClaimCommitHeight := GetEarliestSupplierClaimCommitHeight(
						&test.sharedParams,
						test.queryHeight,
						blockHash,
						supplierAddr,
					)

					require.Greater(t, claimWindowCloseHeight, earliestClaimCommitHeight)

					earliestProofCommitHeight := GetEarliestSupplierProofCommitHeight(
						&test.sharedParams,
						test.queryHeight,
						blockHash,
						supplierAddr,
					)

					require.GreaterOrEqual(t, earliestProofCommitHeight, claimWindowCloseHeight)
					require.Greater(t, proofWindowCloseHeight, earliestProofCommitHeight)

					claimWindowSizeBlocks := GetClaimWindowSizeBlocks(&test.sharedParams)
					require.Greater(t, claimWindowSizeBlocks, uint64(0))

					proofWindowSizeBlocks := GetProofWindowSizeBlocks(&test.sharedParams)
					require.Greater(t, proofWindowSizeBlocks, uint64(0))

					wg.Done()
				}()
			}
		})
	}
	wg.Wait()
}
