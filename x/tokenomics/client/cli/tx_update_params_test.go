package cli_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/pokt-network/poktroll/x/tokenomics/client/cli"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	"github.com/stretchr/testify/require"
)

func TestCLI_UpdateParams(t *testing.T) {
	net := networkWithDefaultConfig(t)
	ctx := net.Validators[0].ClientCtx

	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, net.Validators[0].Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
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

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			args := append(common, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdUpdateParams(), args)
			if tc.expectedErr != nil {
				_, ok := status.FromError(tc.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, err, tc.expectedErr)
				require.Contains(t, err.Error(), tc.expectedExtraErrMsg)
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
