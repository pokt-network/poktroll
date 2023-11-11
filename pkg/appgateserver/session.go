package appgateserver

import (
	"context"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// getCurrentSession gets the current session for the given service
// It returns the current session if it exists and is still valid, otherwise it
// queries for the latest session, caches and returns it.
func (app *appGateServer) getCurrentSession(
	ctx context.Context,
	appAddress, serviceId string,
) (*sessiontypes.Session, error) {
	app.sessionMu.RLock()
	defer app.sessionMu.RUnlock()

	latestBlock := app.blockClient.LatestBlock(ctx)
	if currentSession, ok := app.currentSessions[serviceId]; ok {
		sessionEndBlockHeight := currentSession.Header.SessionStartBlockHeight + currentSession.NumBlocksPerSession

		// Return the current session if it is still valid.
		if latestBlock.Height() < sessionEndBlockHeight {
			return currentSession, nil
		}
	}

	// Query for the current session.
	sessionQueryReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		Service:            &sharedtypes.Service{Id: serviceId},
		BlockHeight:        latestBlock.Height(),
	}
	sessionQueryRes, err := app.sessionQuerier.GetSession(ctx, &sessionQueryReq)
	if err != nil {
		return nil, err
	}

	session := sessionQueryRes.Session

	// Cache the current session.
	app.currentSessions[serviceId] = session

	return session, nil
}
