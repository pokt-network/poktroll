package cli_test

import (
	"strconv"
)

// Prevent strconv unused error
var _ = strconv.IntSize

//func TestShowProof(t *testing.T) {
//	ctx := context.Background()
//	memnet := sessionnet.DefaultNetworkWithSessions(t)
//	memnet.Start(ctx, t)
//
//	claims, sessionTrees := memnet.CreateClaims(t)
//	proofs := memnet.SubmitProofs(t, claims, sessionTrees)
//
//	wrongAppAddrSessionHeader := copy.DeepCopyJSON(t, claims[0].GetSessionHeader())
//	wrongAppAddrSessionHeader.ApplicationAddress = sample.AccAddress()
//
//	invalidAppAddrSessionHeader := copy.DeepCopyJSON(t, claims[0].GetSessionHeader())
//	invalidAppAddrSessionHeader.ApplicationAddress = "invalid_application_bech32"
//
//	wrongServiceSessionHeader := copy.DeepCopyJSON(t, claims[0].GetSessionHeader())
//	wrongServiceSessionHeader.Service.Id = "wrong_service_id"
//
//	wrongSessionIdSessionHeader := copy.DeepCopyJSON(t, claims[0].GetSessionHeader())
//	wrongSessionIdSessionHeader.SessionId = "wrong_session_id"
//
//	common := []string{
//		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
//	}
//	tests := []struct {
//		desc          string
//		sessionHeader *sessiontypes.SessionHeader
//		supplierAddr  string
//
//		args        []string
//		expectedErr error
//		proof       suppliertypes.Proof
//	}{
//		{
//			desc:          "found first",
//			sessionHeader: proofs[0].GetSessionHeader(),
//			supplierAddr:  proofs[0].GetSupplierAddress(),
//
//			args:  common,
//			proof: proofs[0],
//		},
//		{
//			desc:          "found second",
//			sessionHeader: proofs[1].GetSessionHeader(),
//			supplierAddr:  proofs[1].GetSupplierAddress(),
//
//			args:  common,
//			proof: proofs[1],
//		},
//		{
//			desc:          "proof not found (wrong session ID)",
//			sessionHeader: wrongSessionIdSessionHeader,
//			supplierAddr:  sample.AccAddress(),
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc:          "proof not found (invalid application address)",
//			sessionHeader: invalidAppAddrSessionHeader,
//			supplierAddr:  proofs[0].GetSupplierAddress(),
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc:          "proof not found (wrong application address)",
//			sessionHeader: wrongAppAddrSessionHeader,
//			supplierAddr:  proofs[0].GetSupplierAddress(),
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc:          "proof not found (wrong service)",
//			sessionHeader: wrongServiceSessionHeader,
//			supplierAddr:  claims[0].GetSupplierAddress(),
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc:          "proof not found (invalid bech32 supplier address)",
//			sessionHeader: proofs[0].GetSessionHeader(),
//			supplierAddr:  "invalid_supplier_bech32",
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc:          "proof not found (wrong supplier address)",
//			sessionHeader: proofs[0].GetSessionHeader(),
//			supplierAddr:  sample.AccAddress(),
//
//			args: common,
//			// TODO_IN_THIS_COMMIT: assert against sentinel error.
//			expectedErr: status.Error(codes.NotFound, "not found"),
//		},
//	}
//	for _, tc := range tests {
//		t.Run(tc.desc, func(t *testing.T) {
//			args := []string{
//				tc.sessionHeader.GetSessionId(),
//				tc.supplierAddr,
//			}
//			args = append(args, tc.args...)
//			out, cliError := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdShowProof(), args)
//
//			if tc.expectedErr != nil {
//				stat, ok := status.FromError(tc.expectedErr)
//				require.True(t, ok)
//				require.ErrorIs(t, stat.Err(), tc.expectedErr)
//			} else {
//				require.NoError(t, cliError)
//				var resp suppliertypes.QueryGetProofResponse
//				t.Logf("out: %s", out.String())
//				require.NoError(t, memnet.GetNetwork(t).Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
//				require.NotNil(t, resp.Proof)
//				require.EqualValues(t, tc.proof, resp.Proof)
//			}
//		})
//	}
//}
//
//func TestListProof(t *testing.T) {
//	ctx := context.Background()
//	memnet := sessionnet.DefaultNetworkWithSessions(t)
//	memnet.Start(ctx, t)
//
//	claims, sessionTrees := memnet.CreateClaims(t)
//	proofs := memnet.SubmitProofs(t, claims, sessionTrees)
//
//	clientCtx := memnet.GetClientCtx(t)
//	request := func(next []byte, offset, limit uint64, total bool) []string {
//		args := []string{
//			fmt.Sprintf("--%s=json", tmcli.OutputFlag),
//		}
//		if next == nil {
//			args = append(args, fmt.Sprintf("--%s=%d", flags.FlagOffset, offset))
//		} else {
//			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagPageKey, next))
//		}
//		args = append(args, fmt.Sprintf("--%s=%d", flags.FlagLimit, limit))
//		if total {
//			args = append(args, fmt.Sprintf("--%s", flags.FlagCountTotal))
//		}
//		return args
//	}
//	t.Run("ByOffset", func(t *testing.T) {
//		step := 2
//		for i := 0; i < len(proofs); i += step {
//			args := request(nil, uint64(i), uint64(step), false)
//			out, expectedErr := testcli.ExecTestCLICmd(clientCtx, cli.CmdListProof(), args)
//			require.NoError(t, expectedErr)
//			var resp suppliertypes.QueryAllProofsResponse
//			require.NoError(t, memnet.GetNetwork(t).Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
//			require.LessOrEqual(t, len(resp.Proof), step)
//			require.Subset(t,
//				nullify.Fill(proofs),
//				nullify.Fill(resp.Proof),
//			)
//		}
//	})
//	t.Run("ByKey", func(t *testing.T) {
//		step := 2
//		var next []byte
//		for i := 0; i < len(proofs); i += step {
//			args := request(next, 0, uint64(step), false)
//			out, expectedErr := testcli.ExecTestCLICmd(clientCtx, cli.CmdListProof(), args)
//			require.NoError(t, expectedErr)
//			var resp suppliertypes.QueryAllProofsResponse
//			require.NoError(t, memnet.GetNetwork(t).Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
//			require.LessOrEqual(t, len(resp.Proof), step)
//			require.Subset(t,
//				nullify.Fill(proofs),
//				nullify.Fill(resp.Proof),
//			)
//			next = resp.Pagination.NextKey
//		}
//	})
//	t.Run("Total", func(t *testing.T) {
//		args := request(nil, 0, uint64(len(proofs)), true)
//		out, expectedErr := testcli.ExecTestCLICmd(clientCtx, cli.CmdListProof(), args)
//		require.NoError(t, expectedErr)
//		var resp suppliertypes.QueryAllProofsResponse
//		require.NoError(t, memnet.GetNetwork(t).Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
//		require.NoError(t, expectedErr)
//		require.Equal(t, len(proofs), int(resp.Pagination.Total))
//		require.ElementsMatch(t,
//			nullify.Fill(proofs),
//			nullify.Fill(resp.Proof),
//		)
//	})
//}
