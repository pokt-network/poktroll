package suites

import (
	"reflect"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/cases"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// ParamType is a type alias for a module parameter type. It is the string that
// is returned when calling reflect.Value#Type()#Name() on a module parameter.
type ParamType = string

const (
	ParamTypeInt64   ParamType = "int64"
	ParamTypeUint64  ParamType = "uint64"
	ParamTypeFloat32 ParamType = "float32"
	ParamTypeString  ParamType = "string"
	ParamTypeBytes   ParamType = "uint8"
	ParamTypeCoin    ParamType = "Coin"
)

const (
	MsgUpdateParamsName = "MsgUpdateParams"
	MsgUpdateParamName  = "MsgUpdateParam"
)

// ModuleParamConfig holds type information about a module's parameters update
// message(s) along with default and valid non-default values and a query constructor
// function for the module. It is used by ParamsSuite to construct and send
// parameter update messages and assert on their results.
type ModuleParamConfig struct {
	ParamsMsgs ModuleParamsMessages
	// ParamTypes is a map of parameter types to their respective MsgUpdateParam_As*
	// types which satisfy the oneof for the MsgUpdateParam#AsType field. Each AsType
	// type which the module supports should be included in this map.
	ParamTypes map[ParamType]any
	// ValidParams is a set of parameters which are expected to be valid when used
	// together AND when used individually, where the reamining parameters are set
	// to their default values.
	ValidParams      any
	DefaultParams    any
	NewParamClientFn any
}

// ModuleParamsMessages holds a reference to each of the params-related message
// types for a given module. The values are only used for their type information
// which is obtained via reflection. The values are not used for their actual
// message contents and MAY be the zero value.
// If MsgUpdateParam is omitted (i.e. nil), ParamsSuite will assume that
// this module does not support individual parameter updates (i.e. MsgUpdateParam).
// In this case, MsgUpdateParamResponse SHOULD also be omitted.
type ModuleParamsMessages struct {
	MsgUpdateParams         any
	MsgUpdateParamsResponse any
	MsgUpdateParam          any
	MsgUpdateParamResponse  any
	QueryParamsRequest      any
	QueryParamsResponse     any
}

var (
	// MsgUpdateParamEnabledModuleNames is a list of module names which support
	// individual parameter updates (i.e. MsgUpdateParam). It is initialized in
	// init().
	MsgUpdateParamEnabledModuleNames []string

	// ModuleParamConfigMap is a map of module names to their respective parameter
	// configurations. It is used by the ParamsSuite, mostly via reflection,
	// to construct and send parameter update messages and assert on their results.
	ModuleParamConfigMap = map[string]ModuleParamConfig{
		sharedtypes.ModuleName:     SharedModuleParamConfig,
		sessiontypes.ModuleName:    SessionModuleParamConfig,
		servicetypes.ModuleName:    ServiceModuleParamConfig,
		apptypes.ModuleName:        ApplicationModuleParamConfig,
		gatewaytypes.ModuleName:    GatewayModuleParamConfig,
		suppliertypes.ModuleName:   SupplierModuleParamConfig,
		prooftypes.ModuleName:      ProofModuleParamConfig,
		tokenomicstypes.ModuleName: TokenomicsModuleParamConfig,
	}

	_ IntegrationSuite = (*ParamsSuite)(nil)
)

func init() {
	for moduleName, moduleParamCfg := range ModuleParamConfigMap {
		if moduleParamCfg.ParamsMsgs.MsgUpdateParam != nil {
			MsgUpdateParamEnabledModuleNames = append(MsgUpdateParamEnabledModuleNames, moduleName)
		}
	}
}

// ParamsSuite is an integration test suite that provides helper functions for
// querying module parameters and running parameter update messages. It is
// intended to be embedded in other integration test suites which are dependent
// on parameter queries or updates.
type ParamsSuite struct {
	AuthzIntegrationSuite

	// AuthorityAddr is the cosmos account address of the authority for the integration
	// app. It is used as the **granter** of authz grants for parameter update messages.
	// In practice, is an address sourced by an on-chain string and no one has the private key.
	AuthorityAddr cosmostypes.AccAddress
	// AuthorizedAddr is the cosmos account address which is the **grantee** of authz
	// grants for parameter update messages.
	// In practice, it is the address of the foundation or the DAO.
	AuthorizedAddr cosmostypes.AccAddress
}

// NewMsgUpdateParams constructs a new concrete pointer of msgUpdateParams type
// with the given param values set on it. It is returned as a cosmostypes.Msg.
func NewMsgUpdateParams(
	msgUpdateParamsType reflect.Type,
	authorityBech32 string,
	params any,
) cosmostypes.Msg {
	msgUpdateParamsValue := reflect.New(msgUpdateParamsType)
	msgUpdateParamsValue.Elem().
		FieldByName("Authority").
		SetString(authorityBech32)
	msgUpdateParamsValue.Elem().
		FieldByName("Params").
		Set(reflect.ValueOf(params))

	return msgUpdateParamsValue.Interface().(cosmostypes.Msg)
}

// SetupTestAuthzAccounts sets AuthorityAddr for the suite by getting the authority
// from the integration app. It also assigns a new pre-generated identity to be used
// as the AuthorizedAddr for the suite. It is expected to be called after s.NewApp()
// as it depends on the integration app and its pre-generated account iterator.
func (s *ParamsSuite) SetupTestAuthzAccounts() {
	// Set the authority, authorized, and unauthorized addresses.
	s.AuthorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(s.T(), ok, "insufficient pre-generated accounts available")
	s.AuthorizedAddr = nextAcct.Address
}

// SetupTestAuthzGrants creates on-chain authz grants for the MsgUpdateUpdateParam and
// MsgUpdateParams message for each module. It is expected to be called after s.NewApp()
// as it depends on the authority and authorized addresses having been set.
func (s *ParamsSuite) SetupTestAuthzGrants() {
	// Create authz grants for all poktroll modules' MsgUpdateParams messages.
	s.RunAuthzGrantMsgForPoktrollModules(s.T(),
		s.AuthorityAddr,
		s.AuthorizedAddr,
		MsgUpdateParamsName,
		s.GetPoktrollModuleNames()...,
	)

	// Create authz grants for all poktroll modules' MsgUpdateParam messages.
	s.RunAuthzGrantMsgForPoktrollModules(s.T(),
		s.AuthorityAddr,
		s.AuthorizedAddr,
		MsgUpdateParamName,
		// NB: only modules with params are expected to support MsgUpdateParam.
		MsgUpdateParamEnabledModuleNames...,
	)
}

// RunUpdateParams runs the given MsgUpdateParams message via an authz exec as the
// AuthorizedAddr and returns the response bytes and error. It is expected to be called
// after s.SetupTestAuthzGrants() as it depends on an on-chain authz grant to AuthorizedAddr
// for MsgUpdateParams for the given module.
func (s *ParamsSuite) RunUpdateParams(
	t *testing.T,
	msgUpdateParams cosmostypes.Msg,
) (msgResponseBz []byte, err error) {
	t.Helper()

	return s.RunUpdateParamsAsSigner(t, msgUpdateParams, s.AuthorizedAddr)
}

// RunUpdateParamsAsSigner runs the given MsgUpdateParams message via an authz exec
// as signerAddr and returns the response bytes and error. It depends on an on-chain
// authz grant to signerAddr for MsgUpdateParams for the given module.
func (s *ParamsSuite) RunUpdateParamsAsSigner(
	t *testing.T,
	msgUpdateParams cosmostypes.Msg,
	signerAddr cosmostypes.AccAddress,
) (msgResponseBz []byte, err error) {
	t.Helper()

	// Send an authz MsgExec from an unauthorized address.
	execMsg := authz.NewMsgExec(s.AuthorizedAddr, []cosmostypes.Msg{msgUpdateParams})
	msgRespsBz, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(msgRespsBz), "expected exactly 1 message response")
	return msgRespsBz[0], err
}

// RunUpdateParam constructs and runs an MsgUpdateParam message via an authz exec
// as the AuthorizedAddr for the given module, parameter name, and value. It returns
// the response bytes and error. It is expected to be called after s.SetupTestAuthzGrants()
// as it depends on an on-chain authz grant to AuthorizedAddr for MsgUpdateParam for the given module.
func (s *ParamsSuite) RunUpdateParam(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
) (msgResponseBz []byte, err error) {
	t.Helper()

	return s.RunUpdateParamAsSigner(t,
		moduleName,
		paramName,
		paramValue,
		s.AuthorizedAddr,
	)
}

// RunUpdateParamAsSigner constructs and runs an MsgUpdateParam message via an authz exec
// as the given signerAddr for the given module, parameter name, and value. It returns
// the response bytes and error. It depends on an on-chain authz grant to signerAddr for
// MsgUpdateParam for the given module.
func (s *ParamsSuite) RunUpdateParamAsSigner(
	t *testing.T,
	moduleName string,
	paramName string,
	paramValue any,
	signerAddr cosmostypes.AccAddress,
) (msgResponseBz []byte, err error) {
	t.Helper()

	moduleCfg := ModuleParamConfigMap[moduleName]

	paramReflectValue := reflect.ValueOf(paramValue)
	paramType := paramReflectValue.Type().Name()
	switch paramReflectValue.Kind() {
	case reflect.Pointer:
		paramType = paramReflectValue.Elem().Type().Name()
	case reflect.Slice:
		paramType = paramReflectValue.Type().Elem().Name()
	}

	msgIface := moduleCfg.ParamsMsgs.MsgUpdateParam
	msgValue := reflect.ValueOf(msgIface)
	msgType := msgValue.Type()

	// Copy the message and set the authority field.
	msgUpdateParamValue := reflect.New(msgType)
	msgUpdateParamValue.Elem().
		FieldByName("Authority").
		SetString(s.AuthorityAddr.String())

	msgUpdateParamValue.Elem().FieldByName("Name").SetString(cases.ToSnakeCase(paramName))

	msgAsTypeStruct := moduleCfg.ParamTypes[paramType]
	msgAsTypeType := reflect.TypeOf(msgAsTypeStruct)
	msgAsTypeValue := reflect.New(msgAsTypeType)
	switch paramType {
	case ParamTypeUint64:
		// NB: MsgUpdateParam doesn't currently support uint64 param type.
		msgAsTypeValue.Elem().FieldByName("AsInt64").SetInt(int64(paramReflectValue.Interface().(uint64)))
	case ParamTypeInt64:
		msgAsTypeValue.Elem().FieldByName("AsInt64").Set(paramReflectValue)
	case ParamTypeFloat32:
		msgAsTypeValue.Elem().FieldByName("AsFloat").Set(paramReflectValue)
	case ParamTypeString:
		msgAsTypeValue.Elem().FieldByName("AsString").Set(paramReflectValue)
	case ParamTypeBytes:
		msgAsTypeValue.Elem().FieldByName("AsBytes").Set(paramReflectValue)
	case ParamTypeCoin:
		msgAsTypeValue.Elem().FieldByName("AsCoin").Set(paramReflectValue)
	default:
		t.Fatalf("ERROR: unknown field type %q", paramType)
	}

	msgUpdateParamValue.Elem().FieldByName("AsType").Set(msgAsTypeValue)

	msgUpdateParam := msgUpdateParamValue.Interface().(cosmostypes.Msg)

	// Send an authz MsgExec from the authority address.
	execMsg := authz.NewMsgExec(signerAddr, []cosmostypes.Msg{msgUpdateParam})
	execResps, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(execResps), "expected exactly 1 message response")
	return execResps[0], err
}

// RequireModuleHasDefaultParams asserts that the given module's parameters are set
// to their default values.
func (s *ParamsSuite) RequireModuleHasDefaultParams(t *testing.T, moduleName string) {
	t.Helper()

	params, err := s.QueryModuleParams(t, moduleName)
	require.NoError(t, err)

	moduleCfg := ModuleParamConfigMap[moduleName]
	require.EqualValues(t, moduleCfg.DefaultParams, params)
}

// QueryModuleParams queries the given module's parameters and returns them. It is
// expected to be called after s.NewApp() as it depends on the app's query helper.
func (s *ParamsSuite) QueryModuleParams(t *testing.T, moduleName string) (params any, err error) {
	t.Helper()

	moduleCfg := ModuleParamConfigMap[moduleName]

	// Construct a new param client.
	newParamClientFn := reflect.ValueOf(moduleCfg.NewParamClientFn)
	newParamClientFnArgs := []reflect.Value{
		reflect.ValueOf(s.GetApp().QueryHelper()),
	}
	paramClient := newParamClientFn.Call(newParamClientFnArgs)[0]

	// Query for the module's params.
	paramsQueryReqValue := reflect.New(reflect.TypeOf(moduleCfg.ParamsMsgs.QueryParamsRequest))
	callParamsArgs := []reflect.Value{
		reflect.ValueOf(s.GetApp().GetSdkCtx()),
		paramsQueryReqValue,
	}
	callParamsReturnValues := paramClient.MethodByName("Params").Call(callParamsArgs)
	paramsResParamsValue := callParamsReturnValues[0]
	paramResErrValue := callParamsReturnValues[1].Interface()

	isErr := false
	err, isErr = paramResErrValue.(error)
	if !isErr {
		require.Nil(t, callParamsReturnValues[1].Interface())
	}

	paramsValue := paramsResParamsValue.Elem().FieldByName("Params")
	return paramsValue.Interface(), err
}
