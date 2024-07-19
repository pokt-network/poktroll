package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
)

var _ client.SessionQueryClient = (*sessionQuerier)(nil)

// sessionQuerier is a wrapper around the sessiontypes.QueryClient that enables the
// querying of on-chain session information through a single exposed method
// which returns an sessiontypes.Session struct
type sessionQuerier struct {
	clientConn     grpc.ClientConn
	sessionQuerier session.QueryClient
}

// NewSessionQuerier returns a new instance of a client.SessionQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewSessionQuerier(deps depinject.Config) (client.SessionQueryClient, error) {
	sessq := &sessionQuerier{}

	if err := depinject.Inject(
		deps,
		&sessq.clientConn,
	); err != nil {
		return nil, err
	}

	sessq.sessionQuerier = session.NewQueryClient(sessq.clientConn)

	return sessq, nil
}

// GetSession returns an sessiontypes.Session struct for a given appAddress,
// serviceId and blockHeight. It implements the SessionQueryClient#GetSession function.
func (sessq *sessionQuerier) GetSession(
	ctx context.Context,
	appAddress string,
	serviceId string,
	blockHeight int64,
) (*session.Session, error) {
	service := &shared.Service{Id: serviceId}
	req := &session.QueryGetSessionRequest{
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
