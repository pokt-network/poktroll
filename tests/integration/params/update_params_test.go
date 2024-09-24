package params

import (
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration/suites"
)

// MsgUpdateParamsSuite is a test suite which exercises the MsgUpdateParams message
// for each poktroll module via authz, as would be done in a live network in order
// to update **all** parameter values for a given module.
// NB: Not to be confused with MsgUpdateParam (singular), which updates a single
// parameter value for a module.
type MsgUpdateParamsSuite struct {
	suites.ParamsSuite

	unauthorizedAddr cosmostypes.AccAddress
}

// TestUpdateParamsSuite uses the ModuleParamConfig for each module to test the
// MsgUpdateParams message execution via authz on an integration app, as would be
// done in a live network in order to update module parameter values. It uses
// reflection to construct the messages and make assertions about the results to
// improve maintainability and reduce boilerplate.
func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &MsgUpdateParamsSuite{})
}

func (s *MsgUpdateParamsSuite) SetupTest() {
	// Create a fresh integration app for each test.
	s.NewApp(s.T())

	// Initialize the test accounts and create authz grants.
	s.SetupTestAuthzAccounts()
	s.SetupTestAuthzGrants()

	// Allocate an address for unauthorized user.
	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	s.unauthorizedAddr = nextAcct.Address
}

func (s *MsgUpdateParamsSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range s.GetPoktrollModuleNames() {
		moduleCfg := suites.ModuleParamConfigMap[moduleName]

		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			s.RequireModuleHasDefaultParams(t, moduleName)

			// Construct a new MsgUpdateParams and set its authority and params fields.
			expectedParams := moduleCfg.ValidParams
			msgUpdateParamsType := reflect.TypeOf(moduleCfg.ParamsMsgs.MsgUpdateParams)
			msgUpdateParams := suites.NewMsgUpdateParams(
				msgUpdateParamsType,
				s.AuthorityAddr.String(),
				expectedParams,
			)
			updateRes, err := s.RunUpdateParamsAsSigner(t, msgUpdateParams, s.unauthorizedAddr)
			require.ErrorContains(t, err, authz.ErrNoAuthorizationFound.Error())
			require.Nil(t, updateRes)
		})
	}
}

func (s *MsgUpdateParamsSuite) TestAuthorizedMsgUpdateParamsSucceeds() {
	for _, moduleName := range s.GetPoktrollModuleNames() {
		moduleCfg := suites.ModuleParamConfigMap[moduleName]

		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			s.RequireModuleHasDefaultParams(t, moduleName)

			// Construct a new MsgUpdateParams and set its authority and params fields.
			expectedParams := moduleCfg.ValidParams
			msgUpdateParamsType := reflect.TypeOf(moduleCfg.ParamsMsgs.MsgUpdateParams)
			msgUpdateParams := suites.NewMsgUpdateParams(
				msgUpdateParamsType,
				s.AuthorityAddr.String(),
				expectedParams,
			)
			// TODO_IMPROVE: add a Params field to the MsgUpdateParamsResponse
			// and assert that it reflects the updated params.
			_, err := s.RunUpdateParams(t, msgUpdateParams)
			require.NoError(t, err)

			// Query for the module's params.
			params, err := s.QueryModuleParams(t, moduleName)
			require.NoError(t, err)

			// Assert that the module's params are updated.
			require.True(t,
				reflect.DeepEqual(expectedParams, params),
				"expected:\n%+v\nto deeply equal:\n%+v",
				expectedParams, params,
			)
		})
	}
}
