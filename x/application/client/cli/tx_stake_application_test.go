package cli_test

import (
	"context"
	"fmt"
	"io"
	"pocket/testutil/network"
	appmodule "pocket/x/application"
	"pocket/x/application/client/cli"
	"pocket/x/application/types"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"
)

func TestStakeAppCLI(t *testing.T) {
	net := network.New(t)
	val := net.Validators[0]
	ctx := val.ClientCtx

	fields := []string{}

	tests := []struct {
		desc    string
		idIndex string

		args []string
		err  error
		code uint32
	}{
		{
			idIndex: strconv.Itoa(0),

			desc: "valid",
			args: []string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			require.NoError(t, net.WaitForNextBlock())

			args := []string{
				tc.idIndex,
			}
			args = append(args, fields...)
			args = append(args, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)

			var resp sdk.TxResponse
			require.NoError(t, ctx.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.NoError(t, clitestutil.CheckTxCode(net, ctx, resp.TxHash, tc.code))
		})
	}
}

type CLITestSuite struct {
	suite.Suite

	kr        keyring.Keyring
	encCfg    testutilmod.TestEncodingConfig
	baseCtx   client.Context
	clientCtx client.Context
	ctx       context.Context

	owner sdk.AccAddress

	// ac address.Codec
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) SetupSuite() {
	s.encCfg = testutilmod.MakeTestEncodingConfig(appmodule.AppModuleBasic{})
	s.kr = keyring.NewInMemory(s.encCfg.Codec)
	s.baseCtx = client.Context{}.
		WithKeyring(s.kr).
		WithTxConfig(s.encCfg.TxConfig).
		WithCodec(s.encCfg.Codec).
		// WithClient(clitestutil.MockCometRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithOutput(io.Discard).
		WithChainID("test-chain")
		// WithAddressCodec(addresscodec.NewBech32Codec("cosmos")).
		// WithValidatorAddressCodec(addresscodec.NewBech32Codec("cosmosvaloper")).
		// WithConsensusAddressCodec(addresscodec.NewBech32Codec("cosmosvalcons"))

	s.ctx = svrcmd.CreateExecuteContext(context.Background())
	// ctxGen := func() client.Context {
	// 	bz, _ := s.encCfg.Codec.Marshal(&sdk.TxResponse{})
	// 	c := clitestutil.NewMockCometRPC(abci.ResponseQuery{
	// 		Value: bz,
	// 	})
	// 	return s.baseCtx.WithClient(c)
	// }
	// s.clientCtx = ctxGen()

	cfg := network.DefaultConfig()
	genesisState := cfg.GenesisState

	appGenesis := applicationModuleGenesis(2)
	appDataBz, err := s.encCfg.Codec.MarshalJSON(appGenesis)
	s.Require().NoError(err)
	genesisState[types.ModuleName] = appDataBz

	// s.ac = addresscodec.NewBech32Codec("cosmos")

	fmt.Println("OLSH", "HERE")
	s.initAccount()
}

func (s *CLITestSuite) TestCLITxSend() {
	fmt.Println("OLSH", s.owner)
}

// TODO_IN_THIS_COMMIT(@Olshansk): Finish off these tests.
func TestCLI_StakeApplication(t *testing.T) {
	// net, _ := networkWithApplicationObjects(t, 2)
	net := network.New(t)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// cfg := testutilmod.MakeTestEncodingConfig()
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 2)
	appAccount := accounts[0]
	// otherAccount := accounts[1]
	// ctx = ctx.WithFromAddress(appAccount.Address).WithKeyring(kr).WithKeyringOptions()
	ctx = ctx.WithKeyring(kr)

	tests := []struct {
		desc        string
		address     string
		stakeAmount string
		err         *errorsmod.Error
	}{
		// {
		// 	desc:        "stake application - invalid address",
		// 	address:     "invalid",
		// 	stakeAmount: "1000upokt",
		// 	err:         types.ErrAppInvalidAddress,
		// },
		// {
		// 	desc: "stake application - invalid stake amount",
		// 	// address:     apps[0].Address,
		// 	stakeAmount: "1000invalid",
		// 	err:         types.ErrAppInvalidStake,
		// },
		{
			desc: "stake application - valid",

			address:     appAccount.Address.String(),
			stakeAmount: "1000upokt",
		},
	}
	commonArgs := []string{
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
	_, err := clitestutil.MsgSendExec(ctx, net.Validators[0].Address, appAccount.Address, amount, commonArgs...)
	require.NoError(t, err)
	// commonFlags := []string{fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			args := []string{
				tt.stakeAmount,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
				// fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
			}
			// args = append(args, commonArgs...)

			require.NoError(t, net.WaitForNextBlock())

			// args := []string{
			// 	tc.idIndex,
			// }
			// args = append(args, fields...)
			// args = append(args, tc.args...)
			// out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)
			// if tc.err != nil {
			// 	require.ErrorIs(t, err, tc.err)
			// 	return
			// }
			// require.NoError(t, err)

			// var resp sdk.TxResponse
			// require.NoError(t, ctx.Codec.UnmarshalJSON(out.Bytes(), &resp))
			// require.NoError(t, clitestutil.CheckTxCode(net, ctx, resp.TxHash, tc.code))

			// fmt.Println("OLSH", ctx.FromAddress, "~~~~~", ctx.FromAddress.String(), "~~~~~")

			// argsSend := []string{
			// 	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			// 	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
			// 	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
			// }

			// amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
			// _, err := clitestutil.MsgSendExec(ctx, net.Validators[0].Address, net.Validators[0].Address, amount, commonArgs...)
			// require.NoError(t, err)

			// res, err := clitestutil.MsgSendExec(
			// 	ctx,
			// 	net.Validators[0].Address,
			// 	appAccount.Address,
			// 	sdk.NewCoins(
			// 		sdk.NewCoin("upokt", sdk.NewInt(10)),
			// 	),
			// )
			// require.NoError(t, err)
			// fmt.Println("OLSH", res)

			// create a new account

			_, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)
			if tt.err != nil {
				stat, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, int32(stat.Code()), int32(tt.err.ABCICode()))
				return
			}

			require.NoError(t, err)
			// var resp types.MsgStakeApplicationResponse
			// require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			// require.NotNil(t, resp)

		})
	}
}

const (
	OwnerName  = "owner"
	Owner      = "cosmos1kznrznww4pd6gx0zwrpthjk68fdmqypjpkj5hp"
	OwnerArmor = `-----BEGIN TENDERMINT PRIVATE KEY-----
salt: C3586B75587D2824187D2CDA22B6AFB6
type: secp256k1
kdf: bcrypt

1+15OrCKgjnwym1zO3cjo/SGe3PPqAYChQ5wMHjdUbTZM7mWsH3/ueL6swgjzI3b
DDzEQAPXBQflzNW6wbne9IfT651zCSm+j1MWaGk=
=wEHs
-----END TENDERMINT PRIVATE KEY-----`

	testClassID          = "kitty"
	testClassName        = "Crypto Kitty"
	testClassSymbol      = "kitty"
	testClassDescription = "Crypto Kitty"
	testClassURI         = "class uri"
	testID               = "kitty1"
	testURI              = "kitty uri"
)

func (s *CLITestSuite) initAccount() {
	ctx := s.clientCtx
	err := ctx.Keyring.ImportPrivKey(OwnerName, OwnerArmor, "1234567890")
	s.Require().NoError(err)
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 1)

	fmt.Println("OLSH", accounts[0].Address)
	// keyinfo, err := ctx.Keyring.Key(OwnerName)
	// s.Require().NoError(err)

	// args := []string{
	// 	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	// 	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	// 	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10))).String()),
	// }

	// s.owner, err = keyinfo.GetAddress()
	// s.Require().NoError(err)

	// amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
	// fmt.Println("OLSH", accounts[0].Address, s.owner, amount, args)
	// _, err = clitestutil.MsgSendExec(ctx, accounts[0].Address, s.owner, amount, args...)
	// s.Require().NoError(err)
	// fmt.Println("OLSH")
}
