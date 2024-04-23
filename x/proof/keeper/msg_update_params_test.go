package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

const authorizedUid = "authorized"

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		params         *prooftypes.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid authority",
			params: &prooftypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "send enabled param",
			params: &prooftypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    prooftypes.Params{},
			},
			shouldError: false,
		},
		{
			desc: "all good",
			params: &prooftypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, test.params)

			if test.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgServer_UpdateParams_Authz(t *testing.T) {
	cmd.InitSDKConfig()

	// Construct a proof keeper & its dependencies.
	proofKeeperOpts := []keepertest.ProofKeepersOpt{
		// Set block hash so we can have a deterministic expected on-chain proof requested by the protocol.
		keepertest.WithBlockHash(blockHeaderHash),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, proofKeeperOpts...)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	authorizedAddr := createAccount(ctx, t, authorizedUid, keyRing, keepers).GetAddress().String()

	modules := keepers.NewModules()
	integrationApp := modules.NewIntegrationApp(ctx)

	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	// TODO_IN_THIS_COMMIT: use the local keyring...
	//foundationAddress := "xxx"
	proofUpdateParams := &prooftypes.MsgUpdateParams{
		Authority: authority,
		//Authority: authorizedAddr,
		// TODO: set new params..
		Params: prooftypes.DefaultParams(),
	}

	//proofUpdateParamsDelegated := &prooftypes.MsgUpdateParams{
	//	Authority: authorizedAddr,
	//	// TODO: set new params..
	//	Params: prooftypes.DefaultParams(),
	//}

	// Attempt to update the proof module params as unauthorized user
	//result, err := integrationApp.RunMsg(proofUpdateParams)
	//require.Error(t, err)
	//t.Log("unauthorized:")
	//t.Log(result)

	// repeat this for every `MsgUpdatePrams` method
	authorization := authz.NewGenericAuthorization("/" + proto.MessageName(proofUpdateParams))
	t.Log("authorization:")
	t.Logf("%+v", authorization)
	grant, err := authz.NewGrant(time.Now(), authorization, nil)
	if err != nil {
		panic(err)
	}
	grantMsg := &authz.MsgGrant{
		Granter: authority,
		Grantee: authorizedAddr,
		Grant:   grant,
	}
	// grantMsg needs to be included in genesis

	result, err := integrationApp.RunMsg(grantMsg)
	require.NoError(t, err)

	t.Log("granting:")
	t.Log(result)

	_, err = integrationApp.Commit()
	require.NoError(t, err)

	// --

	// Attempt to update the proof module params as authorized user
	//result, err = integrationApp.RunMsg(proofUpdateParams)
	//require.NoError(t, err)
	//msg, err := codectypes.NewAnyWithValue(proofUpdateParams)
	//require.NoError(t, err)

	pnfAccount, err := types.AccAddressFromBech32("pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw")
	require.NoError(t, err)

	execMsg := authz.NewMsgExec(pnfAccount, []cosmostypes.Msg{proofUpdateParams})
	result, err = integrationApp.RunMsg(&execMsg)
	require.NoError(t, err)

	// TODO_IN_THIS_COMMIT: assert on result...
	_ = result
	t.Log("updating:")
	t.Log(result)
}
