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
	s.NewApp(s.T())

	// Call the SetupTest() of the inherited UpdateParamsSuite.
	s.SetupTestAccounts()
	s.SetupTestAuthzGrants()

	// Allocate an address for unauthorized user.
	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	unauthorizedAddr = nextAcct.Address
}

func (s *MsgUpdateParamSuite) TestUnauthorizedMsgUpdateParamFails() {
	for _, moduleName := range suites.MsgUpdateParamEnabledModuleNames {
		// TODO_IN_THIS_COMMIT: improve comment...
		// iterate over each field in the current module's MsgUpdateParam...
		// for each field, send a new MsgUpdateParam populated with the corresopnding field from that module's "validParams" struct...
		defaultParamsValue := reflect.ValueOf(suites.DefaultParamsByModule[moduleName])
		for fieldIdx := 0; fieldIdx < defaultParamsValue.NumField(); fieldIdx++ {
			fieldValue := defaultParamsValue.Field(fieldIdx)
			fieldName := defaultParamsValue.Type().Field(fieldIdx).Name
			fieldType := defaultParamsValue.Type().Field(fieldIdx).Type.Name()
			if fieldType == "" {
				fieldType = defaultParamsValue.Type().Field(fieldIdx).Type.Elem().Name()
			}

			testName := fmt.Sprintf("%s_%s", moduleName, fieldName)
			s.T().Run(testName, func(t *testing.T) {
				// Assert that the module's params are set to their default values.
				s.RequireModuleHasDefaultParams(t, moduleName)

				updateParamResAny, err := s.RunUpdateParamAsSigner(t,
					moduleName,
					fieldName,
					fieldValue.Interface(),
					unauthorizedAddr,
				)
				require.ErrorContains(t, err, authz.ErrNoAuthorizationFound.Error())
				require.Nil(t, updateParamResAny)
			})
		}
	}
}

func (s *MsgUpdateParamSuite) TestAuthorizedMsgUpdateParamSucceeds() {
	for _, moduleName := range suites.MsgUpdateParamEnabledModuleNames {
		// TODO_IN_THIS_COMMIT: improve comment...
		// iterate over each field in the current module's MsgUpdateParam...
		// for each field, send a new MsgUpdateParam populated with the corresopnding field from that module's "validParams" struct...
		defaultParamsValue := reflect.ValueOf(suites.DefaultParamsByModule[moduleName])
		for fieldIdx := 0; fieldIdx < defaultParamsValue.NumField(); fieldIdx++ {
			fieldValue := defaultParamsValue.Field(fieldIdx)
			fieldName := defaultParamsValue.Type().Field(fieldIdx).Name
			fieldType := defaultParamsValue.Type().Field(fieldIdx).Type.Name()
			if fieldType == "" {
				fieldType = defaultParamsValue.Type().Field(fieldIdx).Type.Elem().Name()
			}

			testName := fmt.Sprintf("%s_%s", moduleName, fieldName)
			s.T().Run(testName, func(t *testing.T) {
				// Reset the app state in order to assert that each module
				// param is updated correctly.
				s.SetupTest()

				// Assert that the module's params are set to their default values.
				s.RequireModuleHasDefaultParams(t, moduleName)

				updateParamResAny, err := s.RunUpdateParamAsSigner(t,
					moduleName,
					fieldName,
					fieldValue.Interface(),
					suites.AuthorizedAddr,
				)
				require.NoError(t, err)
				require.NotNil(t, updateParamResAny)

				// Query for the module's params.
				params, err := s.QueryModuleParams(t, moduleName)
				require.NoError(t, err)

				// Assert that the module's params are updated.
				// TODO_IN_THIS_COMMIT: update...
				_ = params
				//require.True(t,
				//	reflect.DeepEqual(params, expectedParamValue.Interface()),
				//	"expected:\n%+v\nto deeply equal:\n%+v",
				//	params, suites.ValidSharedParams,
				//)
			})
		}
	}
}

func TestUpdateParamSuite(t *testing.T) {
	suite.Run(t, new(MsgUpdateParamSuite))
}
