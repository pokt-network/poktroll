package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/session/types"
)

// GetSession should be deterministic and always return the same session for
// the same block height.
func (k Keeper) GetSession(ctx context.Context, req *types.QueryGetSessionRequest) (*types.QueryGetSessionResponse, error) {
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
	var blockHeight int64
	// If the request specifies a block height, use it. Otherwise, use the current
	// block height.
	// Requesting a session with a block height of 0 allows to get the current session,
	// which is useful for querying from CLI.
	blockHeight = sdk.UnwrapSDKContext(ctx).BlockHeight()
	if req.BlockHeight > 0 {
		blockHeight = req.BlockHeight
	}

	k.Logger().Info(fmt.Sprintf("Getting session for height: %d", blockHeight))

	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.ServiceId, blockHeight)
	session, err := k.HydrateSession(ctx, sessionHydrator)
	if err != nil {
		return nil, err
	}

	res := &types.QueryGetSessionResponse{
		Session: session,
	}
	return res, nil
}
