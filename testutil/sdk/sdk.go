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
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
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
	Deps      map[string]depinject.Config
	Ctx       context.Context
}

func (tb *TestBehavior) WithDependencies(behavior func(*TestBehavior) map[string]depinject.Config) *TestBehavior {
	tb.Deps = behavior(tb)
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

func (tb *TestBehavior) BuildDeps() depinject.Config {
	deps := depinject.Configs()
	if len(tb.Deps) == 0 {
		return deps
	}

	for _, dep := range tb.Deps {
		deps = depinject.Configs(deps, dep)
	}

	ringCache, err := rings.NewRingCache(deps)
	require.NoError(tb.T, err)
	deps = depinject.Configs(deps, depinject.Supply(ringCache))

	return deps
}

func NewTestBehavior(t *testing.T) *TestBehavior {
	ctx := context.TODO()
	queryNodeGRPCURL, err := url.Parse(grpcURL)
	require.NoError(t, err)
	queryNodeRPCURL, err := url.Parse(rpcURL)
	require.NoError(t, err)
	decodedPrivateKey, err := hex.DecodeString(privateKey)
	require.NoError(t, err)

	deps := map[string]depinject.Config{}

	logger := polylog.Ctx(ctx)
	deps["logger"] = depinject.Supply(logger)

	blockClient := testblock.NewAnyTimeLastNBlocksBlockClient(t, []byte{}, BlockHeight)
	deps["blockClient"] = depinject.Supply(blockClient)

	accountQuerier := testqueryclients.NewTestAccountQueryClient(t)
	deps["accountQuerier"] = depinject.Supply(accountQuerier)

	applicationQuerier := testqueryclients.NewTestApplicationQueryClient(t)
	deps["applicationQuerier"] = depinject.Supply(applicationQuerier)

	testqueryclients.AddToExistingSessions(
		t,
		ValidAppAddress,
		ValidServiceID,
		BlockHeight,
		[]string{},
	)
	sessionQuerier := testqueryclients.NewTestSessionQueryClient(t)
	deps["sessionQuerier"] = depinject.Supply(sessionQuerier)

	redelegationObs, _ := channel.NewReplayObservable[client.Redelegation](ctx, 1)
	delegationClient := testdelegation.NewAnyTimesRedelegationsSequence(
		ctx,
		t,
		ValidAppAddress,
		redelegationObs,
	)
	deps["delegationClient"] = depinject.Supply(delegationClient)

	_ = testdelegation.NewAnyTimesRedelegation(t, "", "")

	config := &sdk.POKTRollSDKConfig{
		QueryNodeGRPCUrl: queryNodeGRPCURL,
		QueryNodeUrl:     queryNodeRPCURL,
		PrivateKey:       &secp256k1.PrivKey{Key: decodedPrivateKey},
	}

	return &TestBehavior{
		T:         t,
		SdkConfig: config,
		Deps:      deps,
		Ctx:       ctx,
	}
}

func InvalidDependencies(testBehavior *TestBehavior) map[string]depinject.Config {
	return map[string]depinject.Config{}
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

func NonDefaultLatestBlockHeight(testBehavior *TestBehavior) map[string]depinject.Config {
	blockClient := testblock.NewAnyTimeLastNBlocksBlockClient(
		testBehavior.T,
		[]byte{},
		BlockHeight+sessionkeeper.NumBlocksPerSession,
	)

	testBehavior.Deps["blockClient"] = depinject.Supply(blockClient)

	return testBehavior.Deps
}
