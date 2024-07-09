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

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func networkWithProofObjects(t *testing.T, n int) (*network.Network, []types.Proof) {
	t.Helper()
	cfg := network.DefaultConfig()
	state := types.GenesisState{}
	for i := 0; i < n; i++ {
		proof := types.Proof{
			SupplierAddress: sample.AccAddress(),
			SessionHeader: &sessiontypes.SessionHeader{
				SessionId: "mock_session_id",
				// Other fields omitted and unused for these tests.
			},
			// CloseseMerkleProof not required for these tests.
			ClosestMerkleProof: nil,
		}
		nullify.Fill(&proof)
		state.ProofList = append(state.ProofList, proof)
	}
	buf, err := cfg.Codec.MarshalJSON(&state)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), state.ProofList
}

func TestShowProof(t *testing.T) {
	net, proofs := networkWithProofObjects(t, 2)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	tests := []struct {
		desc         string
		sessionId    string
		supplierAddr string

		args        []string
		expectedErr error
		proof       types.Proof
	}{
		{
			desc:         "found",
			supplierAddr: proofs[0].SupplierAddress,
			sessionId:    proofs[0].SessionHeader.SessionId,

			args:  common,
			proof: proofs[0],
		},
		{
			desc:         "not found",
			supplierAddr: sample.AccAddress(),
			sessionId:    proofs[0].SessionHeader.SessionId,

			args:        common,
			expectedErr: status.Error(codes.NotFound, "not found"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			args := []string{
				test.sessionId,
				test.supplierAddr,
			}
			args = append(args, test.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, proof.CmdShowProof(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), test.expectedErr)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetProofResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Proof)
				require.Equal(t,
					nullify.Fill(&test.proof),
					nullify.Fill(&resp.Proof),
				)
			}
		})
	}
}

func TestListProof(t *testing.T) {
	net, proofs := networkWithProofObjects(t, 5)

	ctx := net.Validators[0].ClientCtx
	request := func(next []byte, offset, limit uint64, total bool) []string {
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
		for i := 0; i < len(proofs); i += step {
			args := request(nil, uint64(i), uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, proof.CmdListProof(), args)
			require.NoError(t, err)
			var resp types.QueryAllProofsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Proofs), step)
			require.Subset(t,
				nullify.Fill(proofs),
				nullify.Fill(resp.Proofs),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(proofs); i += step {
			args := request(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, proof.CmdListProof(), args)
			require.NoError(t, err)
			var resp types.QueryAllProofsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Proofs), step)
			require.Subset(t,
				nullify.Fill(proofs),
				nullify.Fill(resp.Proofs),
			)
			next = resp.Pagination.NextKey
		}
	})

	//TODO_TEST: add "BySupplierAddress", "BySession", "ByHeight" tests.

	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(proofs)), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, proof.CmdListProof(), args)
		require.NoError(t, err)
		var resp types.QueryAllProofsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(proofs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(proofs),
			nullify.Fill(resp.Proofs),
		)
	})
}
