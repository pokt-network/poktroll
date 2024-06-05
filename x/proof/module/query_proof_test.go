package proof_test

// TODO_BLOCKER(@Olshansk): Add these tests back in after merging on-chain Proof persistence.
// Prevent strconv unused error
// var _ = strconv.IntSize
//
// func networkWithProofObjects(t *testing.T, n int) (*network.Network, []types.Proof) {
// 	t.Helper()
// 	cfg := network.DefaultConfig()
// 	state := types.GenesisState{}
// 	for i := 0; i < n; i++ {
// 	proof := types.Proof{
// 			Index: strconv.Itoa(i),
//
// 		}
// 		nullify.Fill(&proof)
// 		state.ProofList = append(state.ProofList, proof)
// 	}
// 	buf, err := cfg.Codec.MarshalJSON(&state)
// 	require.NoError(t, err)
// 	cfg.GenesisState[types.ModuleName] = buf
// 	return network.New(t, cfg), state.ProofList
// }
//
// func TestShowProof(t *testing.T) {
// 	net, proofs := networkWithProofObjects(t, 2)
//
// 	ctx := net.Validators[0].ClientCtx
// 	common := []string{
// 		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
// 	}
// 	tests := []struct {
// 		desc string
// 		idIndex string
//
// 		args []string
// 		expectedErr  error
// 		proof  types.Proof
// 	}{
// 		{
// 			desc: "found",
// 			idIndex: proofs[0].Index,
//
// 			args: common,
// 			proof:  proofs[0],
// 		},
// 		{
// 			desc: "not found",
// 			idIndex: strconv.Itoa(100000),
//
// 			args: common,
// 			expectedErr:  status.Error(codes.NotFound, "not found"),
// 		},
// 	}
// 	for _, test := range tests {
// 		t.Run(test.desc, func(t *testing.T) {
// 			args := []string{
// 			    test.idIndex,
//
// 			}
// 			args = append(args, test.args...)
// 			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdShowProof(), args)
// 			if test.expectedErr != nil {
// 				stat, ok := status.FromError(test.expectedErr)
// 				require.True(t, ok)
// 				require.ErrorIs(t, stat.Err(), test.expectedErr)
// 			} else {
// 				require.NoError(t, err)
// 				var resp types.QueryGetProofResponse
// 				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
// 				require.NotNil(t, resp.Proof)
// 				require.Equal(t,
// 					nullify.Fill(&test.proof),
// 					nullify.Fill(&resp.Proof),
// 				)
// 			}
// 		})
// 	}
// }
//
// func TestListProof(t *testing.T) {
// 	net, proofs := networkWithProofObjects(t, 5)
//
// 	ctx := net.Validators[0].ClientCtx
// 	request := func(next []byte, offset, limit uint64, total bool) []string {
// 		args := []string{
// 			fmt.Sprintf("--%s=json", tmcli.OutputFlag),
// 		}
// 		if next == nil {
// 			args = append(args, fmt.Sprintf("--%s=%d", flags.FlagOffset, offset))
// 		} else {
// 			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagPageKey, next))
// 		}
// 		args = append(args, fmt.Sprintf("--%s=%d", flags.FlagLimit, limit))
// 		if total {
// 			args = append(args, fmt.Sprintf("--%s", flags.FlagCountTotal))
// 		}
// 		return args
// 	}
//
// 	t.Run("ByOffset", func(t *testing.T) {
// 		step := 2
// 		for i := 0; i < len(proofs); i += step {
// 			args := request(nil, uint64(i), uint64(step), false)
// 			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListProof(), args)
// 			require.NoError(t, err)
// 			var resp types.QueryAllProofsResponse
// 			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
// 			require.LessOrEqual(t, len(resp.Proof), step)
// 			require.Subset(t,
//             	nullify.Fill(proofs),
//             	nullify.Fill(resp.Proof),
//             )
// 		}
// 	})
//
// 	t.Run("ByKey", func(t *testing.T) {
// 		step := 2
// 		var next []byte
// 		for i := 0; i < len(proofs); i += step {
// 			args := request(next, 0, uint64(step), false)
// 			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListProof(), args)
// 			require.NoError(t, err)
// 			var resp types.QueryAllProofsResponse
// 			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
// 			require.LessOrEqual(t, len(resp.Proof), step)
// 			require.Subset(t,
//             	nullify.Fill(proofs),
//             	nullify.Fill(resp.Proof),
//             )
// 			next = resp.Pagination.NextKey
// 		}
// 	})
//
//  TODO_TEST: add "BySupplierAddress", "BySession", "ByHeight" tests.
//
// 	t.Run("Total", func(t *testing.T) {
// 		args := request(nil, 0, uint64(len(proofs)), true)
// 		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListProof(), args)
// 		require.NoError(t, err)
// 		var resp types.QueryAllProofsResponse
// 		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
// 		require.NoError(t, err)
// 		require.Equal(t, len(proofs), int(resp.Pagination.Total))
// 		require.ElementsMatch(t,
// 			nullify.Fill(proofs),
// 			nullify.Fill(resp.Proof),
// 		)
// 	})
// }
