package params

import (
	"fmt"
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/cases"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
)

type MsgUpdateParamSuite struct {
	suites.UpdateParamsSuite
}

func (s *MsgUpdateParamSuite) SetupTest() {
	// Call the SetupTest() of the inherited UpdateParamsSuite.
	s.UpdateParamsSuite.SetupTest()

	// Allocate an address for unauthorized user.
	nextAcct, ok := s.GetApp(s.T()).GetPreGeneratedAccounts().Next()
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

				msgIface, isMsgTypeFound := suites.MsgUpdateParamByModule[moduleName]
				require.Truef(t, isMsgTypeFound, "unknown message type for module %q: %T", moduleName, msgIface)

				msgValue := reflect.ValueOf(msgIface)
				msgType := msgValue.Type()

				// Copy the message and set the authority field.
				msgValueCopy := reflect.New(msgType)
				msgValueCopy.Elem().Set(msgValue)
				msgValueCopy.Elem().
					FieldByName("Authority").
					SetString(suites.AuthorityAddr.String())

				msgValueCopy.Elem().FieldByName("Name").SetString(cases.ToSnakeCase(fieldName))

				msgAsTypeStruct := suites.MsgUpdateParamTypesByModuleName[moduleName][fieldType]
				msgAsTypeType := reflect.TypeOf(msgAsTypeStruct)
				t.Logf("fieldType: %q", fieldType)
				t.Logf("msgAsTypeType: %+v", msgAsTypeType)
				msgAsTypeValue := reflect.New(msgAsTypeType)
				switch fieldType {
				case "uint64":
					msgAsTypeValue.Elem().FieldByName("AsInt64").SetInt(int64(fieldValue.Interface().(uint64)))
				case "int64":
					msgAsTypeValue.Elem().FieldByName("AsInt64").Set(fieldValue)
				case "float32":
					msgAsTypeValue.Elem().FieldByName("AsFloat").Set(fieldValue)
				case "string":
					msgAsTypeValue.Elem().FieldByName("AsString").Set(fieldValue)
				case "uint8":
					msgAsTypeValue.Elem().FieldByName("AsBytes").Set(fieldValue)
				// TODO_IN_THIS_COMMIT: check type name...
				case "Coin":
					msgAsTypeValue.Elem().FieldByName("AsCoin").Set(fieldValue)
				default:
					t.Fatalf("ERROR: unknown field type %q", fieldType)
				}

				msgValueCopy.Elem().FieldByName("AsType").Set(msgAsTypeValue)

				msgUpdateParam := msgValueCopy.Interface().(cosmostypes.Msg)

				// Set up assertion that the MsgExec will fail.
				errAssertionOpt := integration.WithErrorAssertion(
					func(err error) {
						require.ErrorIs(t, err, authz.ErrNoAuthorizationFound)
					},
				)

				// Send an authz MsgExec from an unauthorized address.
				runOpts := integration.RunUntilNextBlockOpts.Append(errAssertionOpt)
				execMsg := authz.NewMsgExec(unauthorizedAddr, []cosmostypes.Msg{msgUpdateParam})
				anyRes := s.GetApp(t).RunMsg(t, &execMsg, runOpts...)
				require.Nil(t, anyRes)
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

				msgIface, isMsgTypeFound := suites.MsgUpdateParamByModule[moduleName]
				require.Truef(t, isMsgTypeFound, "unknown message type for module %q: %T", moduleName, msgIface)

				msgValue := reflect.ValueOf(msgIface)
				msgType := msgValue.Type()

				// Copy the message and set the authority field.
				msgValueCopy := reflect.New(msgType)
				msgValueCopy.Elem().Set(msgValue)
				msgValueCopy.Elem().
					FieldByName("Authority").
					SetString(suites.AuthorityAddr.String())

				msgValueCopy.Elem().FieldByName("Name").SetString(cases.ToSnakeCase(fieldName))
				// TODO_IN_THIS_COMMIT: merge expected param value with defaults...
				//expectedParamsValue := msgValueCopy.Elem().FieldByName("Params")

				msgAsTypeStruct := suites.MsgUpdateParamTypesByModuleName[moduleName][fieldType]
				msgAsTypeType := reflect.TypeOf(msgAsTypeStruct)
				msgAsTypeValue := reflect.New(msgAsTypeType)
				switch fieldType {
				case "uint64":
					msgAsTypeValue.Elem().FieldByName("AsInt64").SetInt(int64(fieldValue.Interface().(uint64)))
				case "int64":
					msgAsTypeValue.Elem().FieldByName("AsInt64").Set(fieldValue)
				case "float32":
					msgAsTypeValue.Elem().FieldByName("AsFloat").Set(fieldValue)
				case "string":
					msgAsTypeValue.Elem().FieldByName("AsString").Set(fieldValue)
				case "uint8":
					msgAsTypeValue.Elem().FieldByName("AsBytes").Set(fieldValue)
				// TODO_IN_THIS_COMMIT: check type name...
				case "Coin":
					msgAsTypeValue.Elem().FieldByName("AsCoin").Set(fieldValue)
				default:
					t.Fatalf("ERROR: unknown field type %q", fieldType)
				}

				msgValueCopy.Elem().FieldByName("AsType").Set(msgAsTypeValue)

				msgUpdateParam := msgValueCopy.Interface().(cosmostypes.Msg)

				// Send an authz MsgExec from an unauthorized address.
				execMsg := authz.NewMsgExec(suites.AuthorizedAddr, []cosmostypes.Msg{msgUpdateParam})
				anyRes := s.GetApp(t).RunMsg(t, &execMsg, integration.RunUntilNextBlockOpts...)
				require.NotNil(t, anyRes)

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
