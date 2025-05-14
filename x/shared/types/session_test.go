package types_test

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

var sharedParamsHistory = sharedtypes.InitialParamsHistory(sharedtypes.DefaultParams())

func TestGetEarliestSupplierClaimCommitHeight_IsDeterministic(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		wg          = sync.WaitGroup{}
	)

	// Randomize queryHeight, claimWindowOpenBlockHash, and supplierOperatorAddr.
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
			supplierOperatorAddr := sample.AccAddress()
			var claimWindowOpenBlockHash [32]byte

			_, err := rand.Read(claimWindowOpenBlockHash[:]) //nolint:staticcheck // We need a deterministic pseudo-random source.
			require.NoError(t, err)

			expected := sharedtypes.GetEarliestSupplierClaimCommitHeight(
				sharedParamsHistory,
				queryHeight,
				claimWindowOpenBlockHash[:],
				supplierOperatorAddr,
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
					actual := sharedtypes.GetEarliestSupplierClaimCommitHeight(
						sharedParamsHistory,
						queryHeight,
						claimWindowOpenBlockHash[:],
						supplierOperatorAddr,
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
		ctx, cancel = context.WithCancel(context.Background())
		wg          = sync.WaitGroup{}
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
			// Randomize queryHeight, proofWindowOpenBlockHash, and supplierOperatorAddr.
			queryHeight := rand.Int63()
			supplierOperatorAddr := sample.AccAddress()
			var proofWindowOpenBlockHash [32]byte
			_, err := rand.Read(proofWindowOpenBlockHash[:]) //nolint:staticcheck // We need a deterministic pseudo-random source.

			if !assert.NoError(t, err) {
				cancel()
				return
			}

			// Compute expected value.
			expected := sharedtypes.GetEarliestSupplierProofCommitHeight(
				sharedParamsHistory,
				queryHeight,
				proofWindowOpenBlockHash[:],
				supplierOperatorAddr,
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
					actual := sharedtypes.GetEarliestSupplierProofCommitHeight(
						sharedParamsHistory,
						queryHeight,
						proofWindowOpenBlockHash[:],
						supplierOperatorAddr,
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
		desc                string
		sharedParamsHistory sharedtypes.ParamsHistory
		queryHeight         int64
	}{
		{
			desc:                "default params",
			sharedParamsHistory: sharedParamsHistory,
			queryHeight:         int64(1),
		},
		{
			desc: "minimal windows",
			sharedParamsHistory: sharedtypes.ParamsHistory{
				{
					Params: sharedtypes.Params{
						NumBlocksPerSession:          1,
						ClaimWindowOpenOffsetBlocks:  0,
						ClaimWindowCloseOffsetBlocks: 1,
						ProofWindowOpenOffsetBlocks:  0,
						ProofWindowCloseOffsetBlocks: 1,
					},
					ActivationHeight:   1,
					DeactivationHeight: 0,
				},
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
					// Randomize the supplier operator address for each sample.
					// This will produce different randomized earliest claim & proof offsets.
					supplierOperatorAddr := sample.AccAddress()

					claimWindowOpenHeight := test.sharedParamsHistory.GetClaimWindowOpenHeight(test.queryHeight)
					claimWindowCloseHeight := test.sharedParamsHistory.GetClaimWindowCloseHeight(test.queryHeight)

					require.Greater(t, claimWindowCloseHeight, claimWindowOpenHeight)

					proofWindowOpenHeight := test.sharedParamsHistory.GetProofWindowOpenHeight(test.queryHeight)
					proofWindowCloseHeight := test.sharedParamsHistory.GetProofWindowCloseHeight(test.queryHeight)

					require.GreaterOrEqual(t, proofWindowOpenHeight, claimWindowCloseHeight)
					require.Greater(t, proofWindowCloseHeight, proofWindowOpenHeight)

					earliestClaimCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
						test.sharedParamsHistory,
						test.queryHeight,
						blockHash,
						supplierOperatorAddr,
					)

					require.Greater(t, claimWindowCloseHeight, earliestClaimCommitHeight)

					earliestProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
						test.sharedParamsHistory,
						test.queryHeight,
						blockHash,
						supplierOperatorAddr,
					)

					require.GreaterOrEqual(t, earliestProofCommitHeight, claimWindowCloseHeight)
					require.Greater(t, proofWindowCloseHeight, earliestProofCommitHeight)

					sharedParams := test.sharedParamsHistory.GetParamsAtHeight(test.queryHeight)

					claimWindowSizeBlocks := sharedParams.GetClaimWindowCloseOffsetBlocks()
					require.Greater(t, claimWindowSizeBlocks, uint64(0))

					proofWindowSizeBlocks := sharedParams.GetProofWindowCloseOffsetBlocks()
					require.Greater(t, proofWindowSizeBlocks, uint64(0))

					wg.Done()
				}()
			}
		})
	}
	wg.Wait()
}
