package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/x/session"
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

// GetParams queries & returns the session module on-chain parameters.
//
// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last` method
// to get the most recently (asynchronously) observed (and cached) value.
func (sessq *sessionQuerier) GetParams(ctx context.Context) (*sessiontypes.Params, error) {
	req := &sessiontypes.QueryParamsRequest{}
	res, err := sessq.sessionQuerier.Params(ctx, req)
	if err != nil {
		return nil, ErrQuerySessionParams.Wrapf("[%v]", err)
	}
	return &res.Params, nil
}

// GetSessionGracePeriodBlockCount returns the number of blocks in the grace period
// for the session which includes queryHeight.
func (sessq *sessionQuerier) GetSessionGracePeriodBlockCount(
	ctx context.Context,
	sessionEndHeight int64,
) (uint64, error) {
	params, err := sessq.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	// TODO_BLOCKER(#543): Use the values of session params at `sessionEndHeight`.
	_ = sessionEndHeight

	numBlocksPerSession := params.GetNumBlocksPerSession()
	return session.GetSessionGracePeriodBlockCount(numBlocksPerSession), nil
}

// IsWithinGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has not yet elapsed, given currentHeight.
func (sessq *sessionQuerier) IsWithinGracePeriod(
	ctx context.Context,
	sessionEndHeight,
	currentHeight int64,
) (bool, error) {
	params, err := sessq.GetParams(ctx)
	if err != nil {
		return false, err
	}

	// TODO_BLOCKER(#543): Use the values of session params at `sessionEndHeight`.
	_ = sessionEndHeight

	numBlocksPerSession := params.GetNumBlocksPerSession()
	return session.IsWithinGracePeriod(numBlocksPerSession, sessionEndHeight, currentHeight), nil
}

// IsPastGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func (sessq *sessionQuerier) IsPastGracePeriod(
	ctx context.Context,
	sessionEndHeight,
	currentHeight int64,
) (bool, error) {
	paramsRes, err := sessq.sessionQuerier.Params(ctx, &sessiontypes.QueryParamsRequest{})
	if err != nil {
		return false, err
	}

	// TODO_BLOCKER(#543): Use the values of session params at `sessionEndHeight`.
	_ = sessionEndHeight

	numBlocksPerSession := paramsRes.GetParams().NumBlocksPerSession
	return session.IsPastGracePeriod(numBlocksPerSession, sessionEndHeight, currentHeight), nil
}
