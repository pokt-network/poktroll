package shared

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGetEarliestSupplierClaimCommitHeight_IsDeterministic(t *testing.T) {
	var (
		sharedParams = sharedtypes.DefaultParams()
		ctx, cancel  = context.WithCancel(context.Background())
		wg           = sync.WaitGroup{}
	)

	// Randomize queryHeight, claimWindowOpenBlockHash, and supplierAddr.
	for randomizeIdx := 0; randomizeIdx < 100; randomizeIdx++ {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
		}

		wg.Add(1)

		// NB: sample concurrently to save time.
		go func() {
			queryHeight := rand.Int63()
			supplierAddr := sample.AccAddress()
			var claimWindowOpenBlockHash [32]byte

			_, err := rand.Read(claimWindowOpenBlockHash[:]) //nolint:staticcheck // We need a deterministic pseudo-random source.
			require.NoError(t, err)

			expected := GetEarliestSupplierClaimCommitHeight(
				&sharedParams,
				queryHeight,
				claimWindowOpenBlockHash[:],
				supplierAddr,
			)

			// Ensure consecutive calls are deterministic.
			for deterministicIdx := 0; deterministicIdx < 500; deterministicIdx++ {
				select {
				case <-ctx.Done():
					cancel()
					return
				default:
				}

				wg.Add(1)
				go func(deterministicIdx int) {
					actual := GetEarliestSupplierClaimCommitHeight(
						&sharedParams,
						queryHeight,
						claimWindowOpenBlockHash[:],
						supplierAddr,
					)
					require.Equalf(t, expected, actual, "on call number %d", deterministicIdx)
					wg.Done()
				}(deterministicIdx)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	cancel()
}

func TestGetEarliestSupplierProofCommitHeight_IsDeterministic(t *testing.T) {
	var (
		sharedParams = sharedtypes.DefaultParams()
		ctx, cancel  = context.WithCancel(context.Background())
		wg           = sync.WaitGroup{}
	)

	for randomizeIdx := 0; randomizeIdx < 100; randomizeIdx++ {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
		}

		wg.Add(1)

		// NB: sample concurrently to save time.
		go func() {
			// Randomize queryHeight, proofWindowOpenBlockHash, and supplierAddr.
			queryHeight := rand.Int63()
			supplierAddr := sample.AccAddress()
			var proofWindowOpenBlockHash [32]byte
			_, err := rand.Read(proofWindowOpenBlockHash[:]) //nolint:staticcheck // We need a deterministic pseudo-random source.

			if !assert.NoError(t, err) {
				cancel()
				return
			}

			// Compute expected value.
			expected := GetEarliestSupplierProofCommitHeight(
				&sharedParams,
				queryHeight,
				proofWindowOpenBlockHash[:],
				supplierAddr,
			)

			// Ensure consecutive calls are deterministic.
			for deterministicIdx := 0; deterministicIdx < 500; deterministicIdx++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				wg.Add(1)

				// NB: sample concurrently to save time.
				go func(deterministicIdx int) {
					actual := GetEarliestSupplierProofCommitHeight(
						&sharedParams,
						queryHeight,
						proofWindowOpenBlockHash[:],
						supplierAddr,
					)

					if !assert.Equalf(t, expected, actual, "on call number %d", deterministicIdx) {
						cancel()
					}
					wg.Done()
				}(deterministicIdx)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	cancel()
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

					claimWindowSizeBlocks := test.sharedParams.GetClaimWindowCloseOffsetBlocks()
					require.Greater(t, claimWindowSizeBlocks, uint64(0))

					proofWindowSizeBlocks := test.sharedParams.GetProofWindowCloseOffsetBlocks()
					require.Greater(t, proofWindowSizeBlocks, uint64(0))

					wg.Done()
				}()
			}
		})
	}
	wg.Wait()
}
