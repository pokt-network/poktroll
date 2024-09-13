//go:build integration

package suites

import (
	"fmt"
	"strings"
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
	granterAddr, granteeAddr types.AccAddress,
	msgName string,
	moduleNames ...string,
) {
	var runOpts []integration.RunOption
	for moduleIdx, moduleName := range moduleNames {
		// Commit and finalize the block after the last module's grant.
		if moduleIdx == len(moduleNames)-1 {
			runOpts = append(runOpts, integration.RunUntilNextBlockOpts...)
		}

		msgType := fmt.Sprintf(poktrollMsgTypeFormat, moduleName, msgName)
		authorization := &authz.GenericAuthorization{Msg: msgType}
		s.RunAuthzGrantMsg(granterAddr, granteeAddr, authorization, runOpts...)
	}

	authzQueryClient := authz.NewQueryClient(s.app.QueryHelper())
	grantsQueryRes, err := authzQueryClient.GranteeGrants(s.app.GetSdkCtx(), &authz.QueryGranteeGrantsRequest{
		Grantee: granteeAddr.String(),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), grantsQueryRes)

	require.Equalf(s.T(),
		len(allPoktrollModuleNames),
		len(grantsQueryRes.Grants),
		"expected %d grants but got %d: %+v",
		len(allPoktrollModuleNames),
		len(grantsQueryRes.Grants),
		grantsQueryRes.Grants,
	)

	foundModuleGrants := make(map[string]int)
	for _, grant := range grantsQueryRes.GetGrants() {
		require.Equal(s.T(), granterAddr.String(), grant.Granter)
		require.Equal(s.T(), granteeAddr.String(), grant.Grantee)

		for _, moduleName := range allPoktrollModuleNames {
			if strings.Contains(grant.Authorization.GetTypeUrl(), moduleName) {
				foundModuleGrants[moduleName]++
			}
		}
	}

	for _, foundTimes := range foundModuleGrants {
		require.Equal(s.T(), 1, foundTimes)
	}
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) RunAuthzGrantMsg(
	granterAddr,
	granteeAddr types.AccAddress,
	authorization authz.Authorization,
	runOpts ...integration.RunOption,
) {
	grantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, &defaultAuthzGrantExpiration)
	require.NoError(s.T(), err)

	anyRes := s.app.RunMsg(s.T(), grantMsg, runOpts...)
	require.NotNil(s.T(), anyRes)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AuthzIntegrationSuite) RunAuthzExecMsg(
	fromAddr types.AccAddress,
	msgs ...types.Msg,
) {
	execMsg := authz.NewMsgExec(fromAddr, msgs)
	anyRes := s.GetApp().RunMsg(s.T(), &execMsg, integration.RunUntilNextBlockOpts...)
	require.NotNil(s.T(), anyRes)

	execRes := new(authz.MsgExecResponse)
	err := s.GetApp().GetCodec().UnpackAny(anyRes, &execRes)
	require.NoError(s.T(), err)
}
