package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ client.SessionQueryClient = (*sessionQuerier)(nil)

// sessionQuerier is a wrapper around the sessiontypes.QueryClient that enables the
// querying of onchain session information through a single exposed method
// which returns an sessiontypes.Session struct
type sessionQuerier struct {
	clientConn     grpc.ClientConn
	sessionQuerier sessiontypes.QueryClient

	blockClient        client.BlockClient
	sessionCache       map[string]*sessiontypes.Session
	sessionParamsCache *sessiontypes.Params
	sessionCacheMu     sync.Mutex
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewSessionQuerier(ctx context.Context, deps depinject.Config) (client.SessionQueryClient, error) {
	sessq := &sessionQuerier{}

	if err := depinject.Inject(
		deps,
		&sessq.blockClient,
		&sessq.clientConn,
	); err != nil {
		return nil, err
	}

	sessq.sessionQuerier = sessiontypes.NewQueryClient(sessq.clientConn)

	channel.ForEach(
		ctx,
		sessq.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			sessq.sessionCacheMu.Lock()
			defer sessq.sessionCacheMu.Unlock()

			sessq.sessionCache = make(map[string]*sessiontypes.Session)
			sessq.sessionParamsCache = nil
		},
	)

	return sessq, nil
}

// GetSession returns an sessiontypes.Session struct for a given appAddress,
// serviceId and blockHeight. It implements the SessionQueryClient#GetSession function.
func (sessq *sessionQuerier) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*sessiontypes.Session, error) {
	sessq.sessionCacheMu.Lock()
	defer sessq.sessionCacheMu.Unlock()

	sessionCacheKey := fmt.Sprintf("%s-%s", appAddress, serviceId)

	if foundSession, isSessionFound := sessq.sessionCache[sessionCacheKey]; isSessionFound {
		return foundSession, nil
	}

	req := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
		BlockHeight:        blockHeight,
	}
	res, err := sessq.sessionQuerier.GetSession(ctx, req)
	if err != nil {
		return nil, ErrQueryRetrieveSession.Wrapf(
			"address: %s; serviceId: %s; block height: %d; error: [%v]",
			appAddress, serviceId, blockHeight, err,
		)
	}

	sessq.sessionCache[sessionCacheKey] = res.Session
	return res.Session, nil
}

// GetParams queries & returns the session module onchain parameters.
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	sessq.sessionCacheMu.Lock()
	defer sessq.sessionCacheMu.Unlock()

	if sessq.sessionParamsCache != nil {
		return sessq.sessionParamsCache, nil
	}
	req := &sessiontypes.QueryParamsRequest{}
	res, err := sessq.sessionQuerier.Params(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}

	sessq.sessionParamsCache = &res.Params
	return &res.Params, nil
}
