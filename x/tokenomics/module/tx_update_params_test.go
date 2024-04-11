package tokenomics_test

import (
	"fmt"
	"testing"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestCLI_UpdateParams(t *testing.T) {
	net := networkWithDefaultConfig(t)
	ctx := net.Validators[0].ClientCtx

	common := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, net.Validators[0].Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, network.NewBondDenomCoins(t, net, 10)),
	}

	tests := []struct {
		desc                string
		args                []string
		expectedErr         error
		expectedExtraErrMsg string
	}{
		{
			desc:        "valid update of all params",
			args:        []string{"42"},
			expectedErr: nil,
		},
		{
			desc:                "invalid compute_units_to_tokens_multiplier update",
			args:                []string{"0"},
			expectedErr:         types.ErrTokenomicsParamsInvalid,
			expectedExtraErrMsg: "invalid ComputeUnitsToTokensMultiplier",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			args := append(common, test.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, tokenomics.CmdUpdateParams(), args)
			if test.expectedErr != nil {
				_, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, err, test.expectedErr)
				require.Contains(t, err.Error(), test.expectedExtraErrMsg)
			} else {
				require.NoError(t, err)
				var resp sdk.TxResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.TxHash)
				require.Equal(t, uint32(0), resp.Code)
			}
		})
	}
}
