package testqueryclients

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// blockHashBz is the []byte representation of the block hash used in the tests.
var blockHashBz []byte

// sessionsMap is a map of: sessionId -> Session.
// If a sessionId is not present in the map, it implies we have not encountered
// that session yet.
var sessionsMap map[string]*sessiontypes.Session

func init() {
	sessionsMap = make(map[string]*sessiontypes.Session)

	var err error
	if blockHashBz, err = hex.DecodeString("1B1051B7BF236FEA13EFA65B6BE678514FA5B6EA0AE9A7A4B68D45F95E4F18E0"); err != nil {
		panic(fmt.Errorf("error while trying to decode block hash: %w", err))
	}
}

// NewTestSessionQueryClient creates a mock of the SessionQueryClient
// which allows the caller to call GetSession any times and will return
// the session matching the app address, serviceID and the blockHeight passed.
func NewTestSessionQueryClient(
	t *testing.T,
) *mockclient.MockSessionQueryClient {
	ctrl := gomock.NewController(t)

	sessionQuerier := mockclient.NewMockSessionQueryClient(ctrl)
	sessionQuerier.EXPECT().GetSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			address string,
			serviceId string,
			blockHeight int64,
		) (session *sessiontypes.Session, err error) {
			sessionId, _ := testsession.GetSessionIdWithDefaultParams(address, serviceId, blockHashBz, blockHeight)

			session, ok := sessionsMap[sessionId]
			if !ok {
				return nil, fmt.Errorf("error while trying to retrieve a session")
			}

			return session, nil
		}).
		AnyTimes()

	return sessionQuerier
}

// AddToExistingSessions adds the given session to the sessionsMap to mock it "existing"
// on chain, it will also remove the sessions from the map when the test is cleaned up.
func AddToExistingSessions(
	t *testing.T,
	appAddress string,
	serviceId string,
	blockHeight int64,
	suppliersAddress []string,
) {
	t.Helper()

	sessionId, _ := testsession.GetSessionIdWithDefaultParams(appAddress, serviceId, blockHashBz, blockHeight)

	session := sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			Service:                 &sharedtypes.Service{Id: serviceId},
			ApplicationAddress:      appAddress,
			SessionId:               sessionId,
			SessionStartBlockHeight: testsession.GetSessionStartHeightWithDefaultParams(blockHeight),
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(blockHeight),
		},
		NumBlocksPerSession: sharedtypes.DefaultNumBlocksPerSession,
		SessionNumber:       testsession.GetSessionNumberWithDefaultParams(blockHeight),
		SessionId:           sessionId,
		Suppliers:           []*sharedtypes.Supplier{},
	}

	for _, supplierAddress := range suppliersAddress {
		supplier := &sharedtypes.Supplier{Address: supplierAddress}
		session.Suppliers = append(session.Suppliers, supplier)
	}

	sessionsMap[sessionId] = &session

	t.Cleanup(func() {
		delete(sessionsMap, sessionId)
	})
}
