package params

import (
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
)

var unauthorizedAddr cosmostypes.AccAddress

type MsgUpdateParamsSuite struct {
	suites.UpdateParamsSuite
}

func (s *MsgUpdateParamsSuite) SetupTest() {
	// Call the SetupTest() of the inherited UpdateParamsSuite.
	s.UpdateParamsSuite.SetupTest()

	// Allocate an address for unauthorized user.
	nextAcct, ok := s.GetApp(s.T()).GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	unauthorizedAddr = nextAcct.Address
}

func (s *MsgUpdateParamsSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range s.GetModuleNames() {
		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			s.RequireModuleHasDefaultParams(t, moduleName)

			msgIface, isMsgTypeFound := suites.MsgUpdateParamsByModule[moduleName]
			require.Truef(t, isMsgTypeFound, "unknown message type for module %q", moduleName)

			msgValue := reflect.ValueOf(msgIface)
			msgType := msgValue.Type()

			// Copy the message and set the authority field.
			msgValueCopy := reflect.New(msgType)
			msgValueCopy.Elem().Set(msgValue)
			msgValueCopy.Elem().
				FieldByName("Authority").
				SetString(suites.AuthorityAddr.String())
			msgUpdateParams := msgValueCopy.Interface().(cosmostypes.Msg)

			// Set up assertion that the MsgExec will fail.
			errAssertionOpt := integration.WithErrorAssertion(
				func(err error) {
					require.ErrorIs(t, err, authz.ErrNoAuthorizationFound)
				},
			)

			// Send an authz MsgExec from an unauthorized address.
			runOpts := integration.RunUntilNextBlockOpts.Append(errAssertionOpt)
			execMsg := authz.NewMsgExec(unauthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
			anyRes := s.GetApp(t).RunMsg(t, &execMsg, runOpts...)
			require.Nil(t, anyRes)
		})
	}
}

func (s *MsgUpdateParamsSuite) TestAuthorizedMsgUpdateParamsSucceeds() {
	for _, moduleName := range s.GetModuleNames() {
		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			s.RequireModuleHasDefaultParams(t, moduleName)

			msgIface, isMsgTypeFound := suites.MsgUpdateParamsByModule[moduleName]
			require.Truef(t, isMsgTypeFound, "unknown message type for module %q", moduleName)

			msgValue := reflect.ValueOf(msgIface)
			msgType := msgValue.Type()

			// Copy the message and set the authority field.
			msgValueCopy := reflect.New(msgType)
			msgValueCopy.Elem().Set(msgValue)
			msgValueCopy.Elem().
				FieldByName("Authority").
				SetString(suites.AuthorityAddr.String())
			expectedParamsValue := msgValueCopy.Elem().FieldByName("Params")

			msgUpdateParams := msgValueCopy.Interface().(cosmostypes.Msg)

			// Send an authz MsgExec from an unauthorized address.
			execMsg := authz.NewMsgExec(suites.AuthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
			anyRes := s.GetApp(t).RunMsg(t, &execMsg, integration.RunUntilNextBlockOpts...)
			require.NotNil(t, anyRes)

			// Query for the module's params.
			params, err := s.QueryModuleParams(t, moduleName)
			require.NoError(t, err)

			// Assert that the module's params are updated.
			require.True(t,
				reflect.DeepEqual(params, expectedParamsValue.Interface()),
				"expected:\n%+v\nto deeply equal:\n%+v",
				params, suites.ValidSharedParams,
			)
		})
	}
}

func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &MsgUpdateParamsSuite{})
}
