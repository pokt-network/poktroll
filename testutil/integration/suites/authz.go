//go:build integration

package suites

import (
	"fmt"
	"testing"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
)

const (
	// poktrollMsgTypeFormat is the format for a poktroll module's message type.
	// The first %s is the module name, and the second %s is the message name.
	poktrollMsgTypeFormat = "/poktroll.%s.%s"
)

var (
	defaultAuthzGrantExpiration = time.Now().Add(time.Hour)
)

// TODO_IN_THIS_COMMIT: move...
type AuthzIntegrationSuite struct {
	BaseIntegrationSuite
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) SendAuthzGrantMsgForPoktrollModules(
	t *testing.T,
	granterAddr, granteeAddr cosmostypes.AccAddress,
	msgName string,
	moduleNames ...string,
) {
	t.Helper()

	var (
		foundModuleGrants = make(map[string]int)
	)
	for _, moduleName := range moduleNames {
		msgType := fmt.Sprintf(poktrollMsgTypeFormat, moduleName, msgName)
		authorization := &authz.GenericAuthorization{Msg: msgType}
		s.RunAuthzGrantMsg(t, granterAddr, granteeAddr, authorization)

		// Query for the created grant to assert that they were created.
		authzQueryClient := authz.NewQueryClient(s.app.QueryHelper())
		queryGrantsReq := &authz.QueryGrantsRequest{
			Granter:    granterAddr.String(),
			Grantee:    granteeAddr.String(),
			MsgTypeUrl: msgType,
		}
		queryGrantsRes, err := authzQueryClient.Grants(s.app.GetSdkCtx(), queryGrantsReq)
		require.NoError(t, err)
		require.NotNil(t, queryGrantsRes)

		for range queryGrantsRes.GetGrants() {
			foundModuleGrants[moduleName]++
		}
	}

	// Assert that only one grant was created for each module.
	for _, foundTimes := range foundModuleGrants {
		require.Equal(t, 1, foundTimes)
	}
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) RunAuthzGrantMsg(
	t *testing.T,
	granterAddr,
	granteeAddr cosmostypes.AccAddress,
	authorization authz.Authorization,
) {
	t.Helper()

	grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &defaultAuthzGrantExpiration)
	require.NoError(t, err)

	grantResAny, err := s.app.RunMsg(s.T(), grantMsg)
	require.NoError(t, err)
	require.NotNil(t, grantResAny)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) RunAuthzExecMsg(
	t *testing.T,
	fromAddr cosmostypes.AccAddress,
	msgs ...cosmostypes.Msg,
) (msgRespsBz []tx.MsgResponse, err error) {
	t.Helper()

	execMsg := authz.NewMsgExec(fromAddr, msgs)
	execResAny, err := s.GetApp().RunMsg(s.T(), &execMsg)
	if err != nil {
		return nil, err
	}

	require.NotNil(t, execResAny)
	execRes := execResAny.(*authz.MsgExecResponse)
	for _, msgResBz := range execRes.Results {
		msgRespsBz = append(msgRespsBz, msgResBz)
	}

	return msgRespsBz, nil
}
