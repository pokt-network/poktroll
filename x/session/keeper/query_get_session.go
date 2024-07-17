package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/session"
)

// GetSession should be deterministic and always return the same session for
// the same block height.
func (k Keeper) GetSession(ctx context.Context, req *session.QueryGetSessionRequest) (*session.QueryGetSessionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Note that `GetSession` is called via the `Query` service rather than the `Msg` server.
	// The former is stateful but does not lead to state transitions, while the latter one
	// does. The request height depends on how much the node has synched and only acts as a read,
	// while the `Msg` server handles the code flow of the validator when a new block is being proposed.
	blockHeight := req.BlockHeight

	k.Logger().Info(fmt.Sprintf("Getting session for height: %d", blockHeight))

	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.Service.Id, blockHeight)
	hydratedSession, err := k.HydrateSession(ctx, sessionHydrator)
	if err != nil {
		return nil, err
	}

	res := &session.QueryGetSessionResponse{
		Session: hydratedSession,
	}
	return res, nil
}
