//go:build integration

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

var unauthorizedAddr cosmostypes.AccAddress

type MsgUpdateParamsSuite struct {
	suites.UpdateParamsSuite
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
	unauthorizedAddr = nextAcct.Address
}

func (s *MsgUpdateParamsSuite) TestUnauthorizedMsgUpdateParamsFails() {
	for _, moduleName := range s.GetPoktrollModuleNames() {
		moduleCfg := suites.ModuleParamConfigMap[moduleName]

		s.T().Run(moduleName, func(t *testing.T) {
			// Assert that the module's params are set to their default values.
			s.RequireModuleHasDefaultParams(t, moduleName)

			// Construct a new MsgUpdateParams and set its authority and params fields.
			expectedParams := moduleCfg.ValidParams
			msgUpdateParamsValue := reflect.New(reflect.TypeOf(moduleCfg.MsgUpdateParams))
			msgUpdateParamsValue.Elem().
				FieldByName("Authority").
				SetString(suites.AuthorityAddr.String())
			msgUpdateParamsValue.Elem().
				FieldByName("Params").
				Set(reflect.ValueOf(expectedParams))

			msgUpdateParams := msgUpdateParamsValue.Interface().(cosmostypes.Msg)
			updateRes, err := s.RunUpdateParamsAsSigner(t, msgUpdateParams, unauthorizedAddr)
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
			msgUpdateParamsValue := reflect.New(reflect.TypeOf(moduleCfg.MsgUpdateParams))
			msgUpdateParamsValue.Elem().
				FieldByName("Authority").
				SetString(suites.AuthorityAddr.String())
			msgUpdateParamsValue.Elem().
				FieldByName("Params").
				Set(reflect.ValueOf(expectedParams))
			//expectedParams := reflect.ValueOf(moduleCfg.ValidParams).FieldByName("Params")

			// TODO_IMPROVE: add a Params field to the MsgUpdateParamsResponse
			// and assert that it reflects the updated params.
			msgUpdateParams := msgUpdateParamsValue.Interface().(cosmostypes.Msg)
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

func TestUpdateParamsSuite(t *testing.T) {
	suite.Run(t, &MsgUpdateParamsSuite{})
}
