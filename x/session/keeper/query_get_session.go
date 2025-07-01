package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/session/types"
)

// GetSession
// - Deterministically returns the same session for the same block height.
// - Always produces consistent results for identical inputs.
func (k Keeper) GetSession(ctx context.Context, req *types.QueryGetSessionRequest) (*types.QueryGetSessionResponse, error) {
	logger := k.Logger().With("method", "GetSession")

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := req.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Notes on usage:
	// - Called via the Query service (stateful, no state transitions).
	// - Msg server is used for state transitions (e.g., block proposals).
	// - The request height reflects node sync status; Query only reads state.
	var blockHeight int64
	// Block height selection:
	// - Use the specified block height if provided.
	// - If block height is 0, use the current block height (useful for CLI queries).
	blockHeight = sdk.UnwrapSDKContext(ctx).BlockHeight()
	if req.BlockHeight > 0 {
		blockHeight = req.BlockHeight
	}

	logger.Debug(fmt.Sprintf("Getting session for height: %d", blockHeight))

	sessionHydrator := NewSessionHydrator(req.ApplicationAddress, req.ServiceId, blockHeight)
	session, err := k.HydrateSession(ctx, sessionHydrator)
	if err != nil {
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	res := &types.QueryGetSessionResponse{
		Session: session,
	}
	return res, nil
}
