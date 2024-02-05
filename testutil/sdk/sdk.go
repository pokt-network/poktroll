package testsdk

import (
	"context"
	"encoding/hex"
	"net/url"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/sdk"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
)

const (
	privateKey        = "2d00ef074d9b51e46886dc9a1df11e7b986611d0f336bdcf1f0adce3e037ec0a"
	BlockHeight       = 4
	InvalidAppAddress = "invalidAppAddress"
	ValidAppAddress   = "validAppAddress"
	InvalidServiceID  = "invalidServiceID"
	ValidServiceID    = "validServiceID"
	rpcURL            = "https://localhost:8080"
	grpcURL           = "https://localhost:8081"
)

type TestBehavior struct {
	T         *testing.T
	SdkConfig *sdk.POKTRollSDKConfig
	Ctx       context.Context
}

func (tb *TestBehavior) WithDependencies(behavior func(*TestBehavior) depinject.Config) *TestBehavior {
	tb.SdkConfig.Deps = behavior(tb)
	return tb
}

func (tb *TestBehavior) WithQueryNodeGRPCURL(behavior func(*TestBehavior) *url.URL) *TestBehavior {
	tb.SdkConfig.QueryNodeGRPCUrl = behavior(tb)
	return tb
}

func (tb *TestBehavior) WithQueryNodeRPCURL(behavior func(*TestBehavior) *url.URL) *TestBehavior {
	tb.SdkConfig.QueryNodeUrl = behavior(tb)
	return tb
}

func (tb *TestBehavior) WithPrivateKey(behavior func(*TestBehavior) cryptotypes.PrivKey) *TestBehavior {
	tb.SdkConfig.PrivateKey = behavior(tb)
	return tb
}

func NewTestBehavior(t *testing.T) *TestBehavior {
	ctx := context.TODO()
	queryNodeGRPCURL, err := url.Parse(grpcURL)
	require.NoError(t, err)
	queryNodeRPCURL, err := url.Parse(rpcURL)
	require.NoError(t, err)
	decodedPrivateKey, err := hex.DecodeString(privateKey)
	require.NoError(t, err)

	deps := depinject.Configs()

	logger := polylog.Ctx(ctx)
	deps = depinject.Configs(deps, depinject.Supply(logger))

	blockClient := testblock.NewAnyTimeLastNBlocksBlockClient(t, []byte{}, BlockHeight)
	deps = depinject.Configs(deps, depinject.Supply(blockClient))

	accountQuerier := testqueryclients.NewTestAccountQueryClient(t)
	deps = depinject.Configs(deps, depinject.Supply(accountQuerier))

	applicationQuerier := testqueryclients.NewTestApplicationQueryClient(t)
	deps = depinject.Configs(deps, depinject.Supply(applicationQuerier))

	testqueryclients.AddToExistingSessions(
		t,
		ValidAppAddress,
		ValidServiceID,
		BlockHeight,
		[]string{},
	)
	sessionQuerier := testqueryclients.NewTestSessionQueryClient(t)
	deps = depinject.Configs(deps, depinject.Supply(sessionQuerier))

	redelegationObs, _ := channel.NewReplayObservable[client.Redelegation](ctx, 1)
	delegationClient := testdelegation.NewAnyTimesRedelegationsSequence(
		ctx,
		t,
		ValidAppAddress,
		redelegationObs,
	)
	deps = depinject.Configs(deps, depinject.Supply(delegationClient))

	_ = testdelegation.NewAnyTimesRedelegation(t, "", "")

	ringCache, err := rings.NewRingCache(deps)
	require.NoError(t, err)
	deps = depinject.Configs(deps, depinject.Supply(ringCache))

	config := &sdk.POKTRollSDKConfig{
		QueryNodeGRPCUrl: queryNodeGRPCURL,
		QueryNodeUrl:     queryNodeRPCURL,
		PrivateKey:       &secp256k1.PrivKey{Key: decodedPrivateKey},
		Deps:             deps,
	}

	return &TestBehavior{
		T:         t,
		SdkConfig: config,
		Ctx:       ctx,
	}
}

func InvalidDependencies(testBehavior *TestBehavior) depinject.Config {
	return depinject.Configs()
}

func MissingPrivateKey(testBehavior *TestBehavior) cryptotypes.PrivKey {
	return nil
}

func MissingGRPCURL(testBehavior *TestBehavior) *url.URL {
	return nil
}

func MissingRPCURL(testBehavior *TestBehavior) *url.URL {
	return nil
}
