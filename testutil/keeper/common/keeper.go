package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AddServiceActors adds a supplier and an application for a specific
// service so a successful session can be generated for testing purposes.
func AddServiceActors(
	ctx context.Context,
	t *testing.T,
	supplierKeeper SupplierKeeper,
	appKeeper ApplicationKeeper,
	service *sharedtypes.Service,
	supplierAddr string,
	appAddr string,
) {
	t.Helper()

	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})
}

// CreateClaimAndStoreBlockHash creates a valid claim, submits it on-chain,
// and on success, stores the block hash for retrieval at future heights.
// TODO_TECHDEBT(@bryanchriswhite): Consider if we could/should split
// this into two functions.
func CreateClaimAndStoreBlockHash(
	ctx context.Context,
	t *testing.T,
	sessionKeeper SessionKeeper,
	proofMsgServer prooftypes.MsgServer,
	sessionStartHeight int64,
	supplierAddr, appAddr string,
	service *sharedtypes.Service,
	sessionTree relayer.SessionTree,
	sessionHeader *sessiontypes.SessionHeader,
) {
	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Create a create claim message.
	claimMsg := NewTestClaimMsg(t,
		sessionStartHeight,
		sessionHeader.GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		merkleRootBz,
	)
	_, err = proofMsgServer.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)

	// TODO_TECHDEBT(@red-0ne): Centralize the business logic that involves taking
	// into account the heights, windows and grace periods into helper functions.
	proofSubmissionHeight :=
		claimMsg.GetSessionHeader().GetSessionEndBlockHeight() +
			shared.SessionGracePeriodBlocks

	// Set block height to be after the session grace period.
	blockHeightCtx := SetBlockHeight(ctx, proofSubmissionHeight)

	// Store the current context's block hash for future height, which is currently an EndBlocker operation.
	sessionKeeper.StoreBlockHash(blockHeightCtx)
}

// GetSessionHeader is a helper to retrieve the session header
// for a specific (app, service, height).
func GetSessionHeader(
	ctx context.Context,
	t *testing.T,
	sessionKeeper SessionKeeper,
	appAddr string,
	service *sharedtypes.Service,
	blockHeight int64,
) *sessiontypes.SessionHeader {
	t.Helper()

	sessionRes, err := sessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        blockHeight,
		},
	)
	require.NoError(t, err)

	return sessionRes.GetSession().GetHeader()
}
