//go:build integration

package suites

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/integration"
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
	granterAddr, granteeAddr types.AccAddress,
	msgName string,
	moduleNames ...string,
) {
	t.Helper()

	var (
		runOpts           []integration.RunOption
		foundModuleGrants = make(map[string]int)
	)
	for moduleIdx, moduleName := range moduleNames {
		// Commit and finalize the block after the last module's grant.
		if moduleIdx == len(moduleNames)-1 {
			runOpts = append(runOpts, integration.RunUntilNextBlockOpts...)
		}

		msgType := fmt.Sprintf(poktrollMsgTypeFormat, moduleName, msgName)
		authorization := &authz.GenericAuthorization{Msg: msgType}
		s.RunAuthzGrantMsg(t, granterAddr, granteeAddr, authorization, runOpts...)

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
	granteeAddr types.AccAddress,
	authorization authz.Authorization,
	runOpts ...integration.RunOption,
) {
	t.Helper()

	grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &defaultAuthzGrantExpiration)
	require.NoError(t, err)

	anyRes := s.app.RunMsg(s.T(), grantMsg, runOpts...)
	require.NotNil(t, anyRes)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) RunAuthzExecMsg(
	t *testing.T,
	fromAddr types.AccAddress,
	msgs ...types.Msg,
) {
	t.Helper()

	execMsg := authz.NewMsgExec(fromAddr, msgs)
	anyRes := s.GetApp(t).RunMsg(s.T(), &execMsg, integration.RunUntilNextBlockOpts...)
	require.NotNil(t, anyRes)

	execRes := new(authz.MsgExecResponse)
	err := s.GetApp(t).GetCodec().UnpackAny(anyRes, &execRes)
	require.NoError(t, err)
}
