//go:build integration

package params

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration/suites"
)

type MsgUpdateParamSuite struct {
	suites.UpdateParamsSuite
}

func (s *MsgUpdateParamSuite) SetupTest() {
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

func (s *MsgUpdateParamSuite) TestUnauthorizedMsgUpdateParamFails() {
	for _, moduleName := range suites.MsgUpdateParamEnabledModuleNames {
		moduleCfg := suites.ModuleParamConfigMap[moduleName]

		// Iterate over each field in the current module's MsgUpdateParam, for each
		// field, send a new MsgUpdateParam which would update the corresponding param
		// to that field's value.
		validParamsValue := reflect.ValueOf(moduleCfg.ValidParams)
		for fieldIdx := 0; fieldIdx < validParamsValue.NumField(); fieldIdx++ {
			fieldValue := validParamsValue.Field(fieldIdx)
			fieldName := validParamsValue.Type().Field(fieldIdx).Name

			testName := fmt.Sprintf("%s_%s", moduleName, fieldName)
			s.T().Run(testName, func(t *testing.T) {
				// Assert that the module's params are set to their default values.
				s.RequireModuleHasDefaultParams(t, moduleName)

				updateResBz, err := s.RunUpdateParamAsSigner(t,
					moduleName,
					fieldName,
					fieldValue.Interface(),
					unauthorizedAddr,
				)
				require.ErrorContains(t, err, authz.ErrNoAuthorizationFound.Error())
				require.Nil(t, updateResBz)
			})
		}
	}
}

func (s *MsgUpdateParamSuite) TestAuthorizedMsgUpdateParamSucceeds() {
	for _, moduleName := range suites.MsgUpdateParamEnabledModuleNames {
		moduleCfg := suites.ModuleParamConfigMap[moduleName]

		// Iterate over each field in the current module's MsgUpdateParam, for each
		// field, send a new MsgUpdateParam which would update the corresponding param
		// to that field's value.
		validParamsValue := reflect.ValueOf(moduleCfg.ValidParams)
		for fieldIdx := 0; fieldIdx < validParamsValue.NumField(); fieldIdx++ {
			fieldExpectedValue := validParamsValue.Field(fieldIdx)
			fieldName := validParamsValue.Type().Field(fieldIdx).Name

			testName := fmt.Sprintf("%s_%s", moduleName, fieldName)
			s.T().Run(testName, func(t *testing.T) {
				// Reset the app state in order to assert that each module
				// param is updated correctly.
				s.SetupTest()

				// Assert that the module's params are set to their default values.
				s.RequireModuleHasDefaultParams(t, moduleName)

				updateResBz, err := s.RunUpdateParam(t,
					moduleName,
					fieldName,
					fieldExpectedValue.Interface(),
				)
				require.NoError(t, err)
				require.NotNil(t, updateResBz)

				// TODO_TECHDEBT / TODO_TEST: It seems like the response objects are encoded in
				// an unexpected way. It's unclear whether this is the result of being
				// executed via authz. Looking at the code, it seems like authz utilizes the
				// sdk.Result#Data field of the result which is returned from the message handler.
				// These result byte slices are accumulated for each message in the MsgExec and
				// set on the MsgExecResponse#Results field.
				//
				//I would've expected the following to work, but it does not:
				//
				// updateResValue := reflect.New(reflect.TypeOf(moduleCfg.MsgUpdateParamResponse))
				// // NB: using proto.Unmarshal here because authz seems to use
				// // proto.Marshal to serialize each message response.
				// err = proto.Unmarshal(updateResBz, updateResValue.Interface().(cosmostypes.Msg))
				// require.NoError(t, err)
				// updateResParamValue := updateResValue.Elem().FieldByName("Params").Elem().FieldByName(fieldName)
				// require.Equal(t, fieldExpectedValue.Interface(), updateResParamValue.Interface())

				// Query for the module's params.
				params, err := s.QueryModuleParams(t, moduleName)
				require.NoError(t, err)

				paramsValue := reflect.ValueOf(params)
				paramValue := paramsValue.FieldByName(fieldName)
				require.Equal(t, fieldExpectedValue.Interface(), paramValue.Interface())
			})
		}
	}
}

func TestUpdateParamSuite(t *testing.T) {
	suite.Run(t, new(MsgUpdateParamSuite))
}
