package proof_test

import (
	"fmt"
	"testing"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	_ "github.com/pokt-network/pocket/testutil/testpolylog"
	proof "github.com/pokt-network/pocket/x/proof/module"
	"github.com/pokt-network/pocket/x/proof/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

func TestClaim_Show(t *testing.T) {
	numSessions := 1
	numSuppliers := 3
	numApps := 3

	sharedParams := sharedtypes.DefaultParams()
	net, claims, clientCtx := networkWithClaimObjects(t, numSessions, numApps, numSuppliers, &sharedParams)

	commonArgs := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}

	var wrongSupplierOperatorAddr = sample.AccAddress()
	tests := []struct {
		desc                 string
		sessionId            string
		supplierOperatorAddr string

		claim       types.Claim
		expectedErr error
	}{
		{
			desc:                 "claim found",
			sessionId:            claims[0].GetSessionHeader().GetSessionId(),
			supplierOperatorAddr: claims[0].GetSupplierOperatorAddress(),

			claim:       claims[0],
			expectedErr: nil,
		},
		{
			desc:                 "claim not found (wrong session ID)",
			sessionId:            "wrong_session_id",
			supplierOperatorAddr: claims[0].GetSupplierOperatorAddress(),

			expectedErr: status.Error(
				codes.NotFound,
				types.ErrProofClaimNotFound.Wrapf(
					"session ID %q and supplier %q",
					"wrong_session_id",
					claims[0].GetSupplierOperatorAddress(),
				).Error(),
			),
		},
		{
			desc:                 "claim not found (invalid bech32 supplier operator address)",
			sessionId:            claims[0].GetSessionHeader().GetSessionId(),
			supplierOperatorAddr: "invalid_bech32_supplier_operator_address",

			// NB: this is *NOT* a gRPC status error because the bech32 parse
			// error occurs during request validation (i.e. client-side).
			expectedErr: types.ErrProofInvalidAddress.Wrapf(
				// TODO_TECHDEBT: prefer using "%q" in error format strings
				// to disambiguate empty string from space or no output.
				"invalid supplier operator address for claim being retrieved %s; (decoding bech32 failed: invalid separator index -1)",
				"invalid_bech32_supplier_operator_address",
			),
		},
		{
			desc:                 "claim not found (wrong supplier operator address)",
			sessionId:            claims[0].GetSessionHeader().GetSessionId(),
			supplierOperatorAddr: wrongSupplierOperatorAddr,

			expectedErr: status.Error(
				codes.NotFound,
				types.ErrProofClaimNotFound.Wrapf(
					"session ID %q and supplier %q",
					claims[0].GetSessionHeader().GetSessionId(),
					wrongSupplierOperatorAddr,
				).Error(),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			args := []string{
				test.sessionId,
				test.supplierOperatorAddr,
			}
			args = append(args, commonArgs...)
			out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdShowClaim(), args)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)

				var resp types.QueryGetClaimResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Claim)

				require.Equal(t, test.claim.GetSupplierOperatorAddress(), resp.Claim.GetSupplierOperatorAddress())
				require.Equal(t, test.claim.GetRootHash(), resp.Claim.GetRootHash())
				require.Equal(t, test.claim.GetSessionHeader(), resp.Claim.GetSessionHeader())
			}
		})
	}
}

// TODO_HACK(@Olshansk): While working on #359, I uncovered that the configurations
// set at the beginning of this test cannot be set independently of how the helpers
// create claims. I'm adapting the tests in #448, in order to keep moving and not
// waste too much time on fixing the test for now but will revisit.
func TestClaim_List(t *testing.T) {
	numSuppliers := 4
	numApps := 1
	numBlocksPerSession := 4
	sharedParams := &sharedtypes.Params{
		NumBlocksPerSession:                uint64(numBlocksPerSession),
		GracePeriodEndOffsetBlocks:         1,
		ClaimWindowOpenOffsetBlocks:        1,
		ClaimWindowCloseOffsetBlocks:       4,
		ProofWindowOpenOffsetBlocks:        0,
		ProofWindowCloseOffsetBlocks:       4,
		SupplierUnbondingPeriodSessions:    1,
		ApplicationUnbondingPeriodSessions: 1,
		GatewayUnbondingPeriodSessions:     1,
		ComputeUnitsToTokensMultiplier:     42,
	}

	// TODO_HACK(@Olshansk): Due to the bug found in `networkWithClaimObjects`, this
	// is a temporary workaround instead of setting numSessions to its own
	// independent constant, which requires us to temporarily align the
	// with the num blocks per session. See the `forloop` in `networkWithClaimObjects`
	// that has a TODO_HACK as well.
	require.Equal(t, 0, numSuppliers*numApps%numBlocksPerSession)

	numSessions := numSuppliers * numApps / numBlocksPerSession

	// Submitting one claim per block for simplicity
	numClaimsPerSession := numBlocksPerSession
	totalClaims := numSessions * numClaimsPerSession

	// TODO_FLAKY(@bryanchriswhite): The `networkWithClaimObjects is flaky because
	// test configuration and management is hard (see details above). To verify if
	// it's functioning independently, please run:
	// 		$ make itest 2 5 ./x/proof/module -- -run TestClaim_List
	// NB: Adding the call to recover seemed to have mitigated the flakiness.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Test panicked: %s", r)
		}
	}()
	net, claims, clientCtx := networkWithClaimObjects(t, numSessions, numSuppliers, numApps, sharedParams)

	prepareArgs := func(next []byte, offset, limit uint64, total bool) []string {
		args := []string{
			fmt.Sprintf("--%s=json", cometcli.OutputFlag),
		}
		if next == nil {
			args = append(args, fmt.Sprintf("--%s=%d", flags.FlagOffset, offset))
		} else {
			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagPageKey, next))
		}
		args = append(args, fmt.Sprintf("--%s=%d", flags.FlagLimit, limit))
		if total {
			args = append(args, fmt.Sprintf("--%s", flags.FlagCountTotal))
		}
		return args
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < totalClaims; i += step {
			args := prepareArgs(nil, uint64(i), uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claims), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claims),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < totalClaims; i += step {
			args := prepareArgs(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claims), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claims),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("BySupplierOperatorAddress", func(t *testing.T) {
		supplierOperatorAddr := claims[0].SupplierOperatorAddress
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", proof.FlagSupplierOperatorAddress, supplierOperatorAddr))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.SupplierOperatorAddress == supplierOperatorAddr {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, numSessions*numApps, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claims),
		)
	})

	t.Run("BySession", func(t *testing.T) {
		sessionId := claims[0].GetSessionHeader().SessionId
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", proof.FlagSessionId, sessionId))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.GetSessionHeader().SessionId == sessionId {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, numSuppliers, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claims),
		)
	})

	t.Run("ByHeight", func(t *testing.T) {
		sessionEndHeight := claims[0].GetSessionHeader().GetSessionEndBlockHeight()
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%d", proof.FlagSessionEndHeight, sessionEndHeight))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.GetSessionHeader().GetSessionEndBlockHeight() == sessionEndHeight {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, numClaimsPerSession, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claims),
		)
	})

	t.Run("Total", func(t *testing.T) {
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		out, err := clitestutil.ExecTestCLICmd(clientCtx, proof.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, totalClaims, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claims),
		)
	})
}
