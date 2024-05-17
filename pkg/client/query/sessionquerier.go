package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.SessionQueryClient = (*sessionQuerier)(nil)

// sessionQuerier is a wrapper around the sessiontypes.QueryClient that enables the
// querying of on-chain session information through a single exposed method
// which returns an sessiontypes.Session struct
type sessionQuerier struct {
	clientConn     grpc.ClientConn
	sessionQuerier sessiontypes.QueryClient
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewSessionQuerier(deps depinject.Config) (client.SessionQueryClient, error) {
	sessq := &sessionQuerier{}

	if err := depinject.Inject(
		deps,
		&sessq.clientConn,
	); err != nil {
		return nil, err
	}

	sessq.sessionQuerier = sessiontypes.NewQueryClient(sessq.clientConn)

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
	service := &sharedtypes.Service{Id: serviceId}
	req := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		Service:            service,
		BlockHeight:        blockHeight,
	}
	res, err := sessq.sessionQuerier.GetSession(ctx, req)
	if err != nil {
		return nil, ErrQueryRetrieveSession.Wrapf(
			"address: %s; serviceId: %s; block height: %d; error: [%v]",
			appAddress, serviceId, blockHeight, err,
		)
	}
	return res.Session, nil
}

// TODO_IN_THIS_PR: godoc comments...
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	req := &sessiontypes.QueryParamsRequest{}
	res, err := sessq.sessionQuerier.Params(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}
	return &res.Params, nil
}

// TODO_TECHDEBT: We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last` method
// to get the most recently (asynchronously) observed (and cached) value.
//
// TODO_IN_THIS_PR: godoc comments...
func (sessq *sessionQuerier) GetSessionGracePeriodBlockCount(
	ctx context.Context,
	queryHeight int64,
) (uint64, error) {
	paramsRes, err := sessq.sessionQuerier.Params(ctx, &sessiontypes.QueryParamsRequest{})
	if err != nil {
		return 0, err
	}

	// TODO_BLOCKER: Use `queryHeight` & some alternate queries to retrieve
	// the value of session params at that height.
	_ = queryHeight

	numBlocksPerSession := paramsRes.GetParams().NumBlocksPerSession

	return sessionkeeper.SessionGracePeriod * numBlocksPerSession, nil
}

// IsWithinGracePeriod checks if the grace period for the session has ended
// and signals whether it is time to create a claim for it.
func (sessq *sessionQuerier) IsWithinGracePeriod(
	ctx context.Context,
	sessionEndBlockHeight,
	currentBlockHeight int64,
) (bool, error) {
	sessionGracePeriodEndBlocks, err := sessq.GetSessionGracePeriodBlockCount(ctx, currentBlockHeight)
	if err != nil {
		return false, err
	}

	sessionGracePeriodEndHeight := sessionEndBlockHeight + int64(sessionGracePeriodEndBlocks)
	return currentBlockHeight <= sessionGracePeriodEndHeight, nil
}
