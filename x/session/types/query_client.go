package types

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
)

// TODO_IN_THIS_COMMIT: godoc...
type SessionQueryClient interface {
	QueryClient

	GetParams(context.Context) (*Params, error)
}

// TODO_IN_THIS_COMMIT: godoc...
func NewSessionQueryClient(conn gogogrpc.ClientConn) SessionQueryClient {
	return NewQueryClient(conn).(SessionQueryClient)
}

// TODO_IN_THIS_COMMIT: investigate generalization...
// TODO_IN_THIS_COMMIT: godoc...
func (c *queryClient) GetParams(ctx context.Context) (*Params, error) {
	res, err := c.Params(ctx, &QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	params := res.GetParams()
	return &params, nil
}
