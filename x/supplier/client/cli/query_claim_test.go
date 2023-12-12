package cli_test

import (
	"fmt"
	"testing"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestClaim_Show(t *testing.T) {
	sessionCount := 1
	supplierCount := 3
	appCount := 3

	net, claims := networkWithClaimObjects(
		t, sessionCount,
		appCount,
		supplierCount,
	)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	tests := []struct {
		desc         string
		sessionId    string
		supplierAddr string

		args  []string
		err   error
		claim types.Claim
	}{
		{
			desc:         "claim found",
			sessionId:    claims[0].GetSessionHeader().GetSessionId(),
			supplierAddr: claims[0].GetSupplierAddress(),

			args:  common,
			claim: claims[0],
		},
		{
			desc:         "claim not found (wrong session ID)",
			sessionId:    "wrong_session_id",
			supplierAddr: claims[0].GetSupplierAddress(),

			args: common,
			err:  status.Error(codes.NotFound, "not found"),
		},
		{
			desc:         "claim not found (wrong supplier address)",
			sessionId:    claims[0].GetSessionHeader().GetSessionId(),
			supplierAddr: "wrong_supplier_address",

			args: common,
			err:  status.Error(codes.NotFound, "not found"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			args := []string{
				tc.sessionId,
				tc.supplierAddr,
			}
			args = append(args, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdShowClaim(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetClaimResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Claim)

				require.Equal(t, tc.claim.GetSupplierAddress(), resp.Claim.GetSupplierAddress())
				require.Equal(t, tc.claim.GetRootHash(), resp.Claim.GetRootHash())
				require.Equal(t, tc.claim.GetSessionHeader(), resp.Claim.GetSessionHeader())
			}
		})
	}
}

func TestClaim_List(t *testing.T) {
	sessionCount := 2
	supplierCount := 4
	appCount := 3
	serviceCount := 1
	// Each supplier will submit a claim for each app x service combination (per session).
	numClaimsPerSession := supplierCount * appCount * serviceCount
	totalClaims := sessionCount * numClaimsPerSession

	net, claims := networkWithClaimObjects(
		t, sessionCount,
		supplierCount,
		appCount,
	)

	ctx := net.Validators[0].ClientCtx
	prepareArgs := func(next []byte, offset, limit uint64, total bool) []string {
		args := []string{
			fmt.Sprintf("--%s=json", tmcli.OutputFlag),
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
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claim), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claim),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < totalClaims; i += step {
			args := prepareArgs(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claim), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claim),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("ByAddress", func(t *testing.T) {
		supplierAddr := claims[0].SupplierAddress
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", cli.FlagSupplierAddress, supplierAddr))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.SupplierAddress == supplierAddr {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
		require.Equal(t, sessionCount*appCount, int(resp.Pagination.Total))
	})

	t.Run("BySession", func(t *testing.T) {
		sessionId := claims[0].GetSessionHeader().SessionId
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", cli.FlagSessionId, sessionId))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.GetSessionHeader().SessionId == sessionId {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
		require.Equal(t, supplierCount, int(resp.Pagination.Total))
	})

	t.Run("ByHeight", func(t *testing.T) {
		sessionEndHeight := claims[0].GetSessionHeader().GetSessionEndBlockHeight()
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%d", cli.FlagSessionEndHeight, sessionEndHeight))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.GetSessionHeader().GetSessionEndBlockHeight() == sessionEndHeight {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, numClaimsPerSession, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
	})

	t.Run("Total", func(t *testing.T) {
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, totalClaims, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claim),
		)
	})
}
