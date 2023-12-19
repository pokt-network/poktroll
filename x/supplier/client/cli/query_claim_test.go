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

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/sessionnet"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestClaim_Show(t *testing.T) {
	// TODO_IN_THIS_COMMIT: consolidate / re-evaluate DefaultNetworkWithSessions
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		&network.InMemoryNetworkConfig{
			NumSessions:     1,
			NumSuppliers:    3,
			NumApplications: 3,
		},
	)
	//memnet := sessionnet.DefaultNetworkWithSessions(t)
	memnet.Start(t)

	claims, _ := memnet.CreateClaims(t)
	net := memnet.GetNetwork(t)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}

	var wrongSupplierAddr = sample.AccAddress()
	tests := []struct {
		desc         string
		sessionId    string
		supplierAddr string

		args        []string
		expectedErr error
		claim       types.Claim
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

			expectedErr: status.Error(
				codes.NotFound,
				types.ErrSupplierClaimNotFound.Wrapf(
					// TODO_CONSIDERATION: factor out error message format strings to constants.
					"session ID %q and supplier %q",
					"wrong_session_id",
					claims[0].GetSupplierAddress(),
				).Error(),
			),
		},
		{
			desc:         "claim not found (invalid bech32 supplier address)",
			sessionId:    claims[0].GetSessionHeader().GetSessionId(),
			supplierAddr: "invalid_bech32_supplier_address",

			args: common,
			// NB: this is *NOT* a gRPC status error because the bech32 parse
			// error occurs during request validation (i.e. client-side).
			expectedErr: types.ErrSupplierInvalidAddress.Wrapf(
				// TODO_CONSIDERATION: prefer using "%q" in error format strings
				// to disambiguate empty string from space or no output.
				"invalid supplier address for claim being retrieved %s; (decoding bech32 failed: invalid separator index -1)",
				"invalid_bech32_supplier_address",
			),
		},
		{
			desc:         "claim not found (wrong supplier address)",
			sessionId:    claims[0].GetSessionHeader().GetSessionId(),
			supplierAddr: wrongSupplierAddr,

			args: common,
			expectedErr: status.Error(
				codes.NotFound,
				types.ErrSupplierClaimNotFound.Wrapf(
					// TODO_CONSIDERATION: factor out error message format strings to constants.
					"session ID %q and supplier %q",
					claims[0].GetSessionHeader().GetSessionId(),
					wrongSupplierAddr,
				).Error(),
			),
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
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
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
	numSessions := 2
	numSuppliers := 4
	numApps := 3
	serviceCount := 1
	// Each supplier will submit a claim for each app x service combination (per session).
	numClaimsPerSession := numSuppliers * numApps * serviceCount
	totalClaims := numSessions * numClaimsPerSession

	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		&network.InMemoryNetworkConfig{
			NumSessions:         numSessions,
			NumRelaysPerSession: 5,
			NumSuppliers:        numSuppliers,
			NumApplications:     numApps,
		},
	)
	memnet.Start(t)

	claims, _ := memnet.CreateClaims(t)
	net := memnet.GetNetwork(t)

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
		require.Equal(t, numSessions*numApps, int(resp.Pagination.Total))
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
		require.Equal(t, numSuppliers, int(resp.Pagination.Total))
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
