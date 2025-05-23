package suites

import (
	"fmt"
	"testing"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
)

const (
	// pocketMsgTypeFormat is the format for a pocket module's message type.
	// The first %s is the module name, and the second %s is the message name.
	pocketMsgTypeFormat = "/pocket.%s.%s"
)

var defaultAuthzGrantExpiration = time.Now().Add(time.Hour)

// AuthzIntegrationSuite is an integration test suite that provides helper functions for
// running authz grant and exec messages. It is intended to be embedded in other integration
// test suites which are dependent on authz.
type AuthzIntegrationSuite struct {
	BaseIntegrationSuite
}

// RunAuthzGrantMsgForPocketModules creates an onchain authz grant for the given
// granter and grantee addresses for the specified message name in each of the pocket
// modules present in the integration app.
func (s *AuthzIntegrationSuite) RunAuthzGrantMsgForPocketModules(
	t *testing.T,
	granterAddr, granteeAddr cosmostypes.AccAddress,
	msgName string,
	moduleNames ...string,
) {
	t.Helper()

	var foundModuleGrants = make(map[string]int)
	for _, moduleName := range moduleNames {
		msgType := fmtPocketMsgType(moduleName, msgName)
		authorization := &authz.GenericAuthorization{Msg: msgType}
		s.RunAuthzGrantMsg(t, granterAddr, granteeAddr, authorization)

		// Query for the created grant to assert that they were created.
		authzQueryClient := authz.NewQueryClient(s.GetApp().QueryHelper())
		queryGrantsReq := &authz.QueryGrantsRequest{
			Granter:    granterAddr.String(),
			Grantee:    granteeAddr.String(),
			MsgTypeUrl: msgType,
		}
		queryGrantsRes, err := authzQueryClient.Grants(s.GetApp().GetSdkCtx(), queryGrantsReq)
		require.NoError(t, err)
		require.NotNil(t, queryGrantsRes)

		// Count the number of grants found for each module.
		for range queryGrantsRes.GetGrants() {
			foundModuleGrants[moduleName]++
		}
	}

	// Assert that only one grant was created for each module.
	for _, foundTimes := range foundModuleGrants {
		require.Equal(t, 1, foundTimes)
	}
}

// RunAuthzGrantMsg creates an onchain authz grant from the given granter to the
// grantee addresses for the authorization object provided.
func (s *AuthzIntegrationSuite) RunAuthzGrantMsg(
	t *testing.T,
	granterAddr,
	granteeAddr cosmostypes.AccAddress,
	authorization authz.Authorization,
) {
	t.Helper()

	grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &defaultAuthzGrantExpiration)
	require.NoError(t, err)

	grantResAny, err := s.GetApp().RunMsg(t, grantMsg)
	require.NoError(t, err)
	require.NotNil(t, grantResAny)
}

// RunAuthzExecMsg executes the given messag(es) using authz. It assumes that an
// authorization exists for which signerAdder is the grantee.
func (s *AuthzIntegrationSuite) RunAuthzExecMsg(
	t *testing.T,
	signerAddr cosmostypes.AccAddress,
	msgs ...cosmostypes.Msg,
) (msgRespsBz [][]byte, err error) {
	t.Helper()

	execMsg := authz.NewMsgExec(signerAddr, msgs)
	execResAny, err := s.GetApp().RunMsg(t, &execMsg)
	if err != nil {
		return nil, err
	}

	require.NotNil(t, execResAny)
	return execResAny.(*authz.MsgExecResponse).Results, nil
}

// fmtPocketMsgType returns the formatted message type for a pocket module.
func fmtPocketMsgType(moduleName, msgName string) string {
	return fmt.Sprintf(pocketMsgTypeFormat, moduleName, msgName)
}
