package types

import (
	"context"

	gogogrpc "github.com/cosmos/gogoproto/grpc"
)

// TODO_IN_THIS_COMMIT: godoc...
type ServiceQueryClient interface {
	QueryClient

	GetParams(context.Context) (*Params, error)
}

// TODO_IN_THIS_COMMIT: godoc...
func NewServiceQueryClient(conn gogogrpc.ClientConn) ServiceQueryClient {
	return NewQueryClient(conn).(ServiceQueryClient)
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
