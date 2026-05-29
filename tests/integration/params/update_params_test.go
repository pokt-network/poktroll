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

// msgUpdateParamsTestSuite is a test suite which exercises the MsgUpdateParams message
// for each pocket module via authz, as would be done in a live network in order
// to update **all** parameter values for a given module.
// NB: Not to be confused with MsgUpdateParam (singular), which updates a single
// parameter value for a module.
type msgUpdateParamsTestSuite struct {
	suites.ParamsSuite

	unauthorizedAddr cosmostypes.AccAddress
}

// TestUpdateParamsSuite uses the ModuleParamConfig for each module to test the
// MsgUpdateParams message execution via authz on an integration app, as would be
// done in a live network in order to update module parameter values. It uses
// reflection to construct the messages and make assertions about the results to
// improve maintainability and reduce boilerplate.
func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &msgUpdateParamsTestSuite{})
}

func (s *msgUpdateParamsTestSuite) SetupTest() {
	// Create a fresh integration app for each test.
	s.NewApp(s.T())

	// Initialize the test accounts and create authz grants.
	s.SetupTestAuthzAccounts(s.T())
	s.SetupTestAuthzGrants(s.T())

	// Allocate an address for unauthorized user.
	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	s.unauthorizedAddr = nextAcct.Address
}

func (s *msgUpdateParamsTestSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range s.GetPocketModuleNames() {
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

func (s *msgUpdateParamsTestSuite) TestAuthorizedMsgUpdateParamsSucceeds() {
	for _, moduleName := range s.GetPocketModuleNames() {
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

			// Compares all SETTABLE params fields, ignoring derived metadata (e.g. the
			// anchored-session-grid anchor, #543) which the handler stamps rather than
			// echoing from the request.
			equalIgnoringDerived := func(expected, actual any) bool {
				expectedVal := reflect.ValueOf(expected)
				actualVal := reflect.ValueOf(actual)
				for i := 0; i < expectedVal.NumField(); i++ {
					name := expectedVal.Type().Field(i).Name
					if suites.IsNonSettableParamField(name) {
						continue
					}
					if !reflect.DeepEqual(expectedVal.Field(i).Interface(), actualVal.Field(i).Interface()) {
						return false
					}
				}
				return true
			}

			// A shared num_blocks_per_session change takes effect on live params only at the
			// next session boundary under the anchored grid (#543); advance blocks until the
			// settable params reflect the update (a no-op for modules with immediate effect).
			const maxAdvanceBlocks = 30
			for i := 0; i < maxAdvanceBlocks && !equalIgnoringDerived(expectedParams, params); i++ {
				s.GetApp().NextBlock(t)
				params, err = s.QueryModuleParams(t, moduleName)
				require.NoError(t, err)
			}

			// Assert that the module's params are updated.
			require.True(t,
				equalIgnoringDerived(expectedParams, params),
				"expected (ignoring derived fields):\n%+v\nto deeply equal:\n%+v",
				expectedParams, params,
			)
		})
	}
}
