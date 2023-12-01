package testqueryclients

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// sessionsMap is a map of:
//
//	sessionId -> Session.
//
// If a sessionId is not present in the map or if the Session associated
// with a sessionId is nil it is assumed that it does not exist on chain.
var sessionsMap map[string]*sessiontypes.Session

func init() {
	sessionsMap = make(map[string]*sessiontypes.Session)
}

// NewTestSessionQueryClient creates a mock of the SessionQueryClient
// which allows the caller to call GetSession any times and will return
// an application with the given address.
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
			sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", address, serviceId, blockHeight)))

			session, ok := sessionsMap[string(sessionId[:])]
			if !ok {
				return nil, fmt.Errorf("session not found")
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

	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, serviceId, blockHeight)))

	session := sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			Service:                 &sharedtypes.Service{Id: serviceId},
			ApplicationAddress:      appAddress,
			SessionStartBlockHeight: blockHeight,
		},
		SessionId: string(sessionId[:]),
		Suppliers: []*sharedtypes.Supplier{},
	}

	for _, supplierAddress := range suppliersAddress {
		supplier := &sharedtypes.Supplier{Address: supplierAddress}
		session.Suppliers = append(session.Suppliers, supplier)
	}

	sessionsMap[string(sessionId[:])] = &session

	t.Cleanup(func() {
		delete(sessionsMap, string(sessionId[:]))
	})
}
