package types

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
)

// SharedQueryClient is an interface which adapts generated (concrete) shared query client
// to paramsQuerierIface (see: pkg/client/query/paramsquerier.go) such that implementors
// (i.e. the generated shared query client) is compliant with client.ParamsQuerier for the
// shared module's params type. This is required to resolve generic type constraints.
type SharedQueryClient interface {
	QueryClient
	GetParams(context.Context) (*Params, error)
}

// NewSharedQueryClient is a wrapper for the shared query client constructor which
// returns a new shared query client as a SharedQueryClient interface type.
func NewSharedQueryClient(conn gogogrpc.ClientConn) SharedQueryClient {
	return NewQueryClient(conn).(SharedQueryClient)
}

// GetParams returns the shared module's params as a pointer, which is critical to
// resolve related generic type constraints between client.ParamsQuerier and it's usages.
func (c *queryClient) GetParams(ctx context.Context) (*Params, error) {
	res, err := c.Params(ctx, &QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	params := res.GetParams()
	return &params, nil
}
