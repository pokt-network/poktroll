package query

import (
	"context"

	"cosmossdk.io/depinject"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
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
