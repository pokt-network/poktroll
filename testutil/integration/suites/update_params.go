package suites

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/cases"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	MsgUpdateParamsName = "MsgUpdateParams"
	MsgUpdateParamName  = "MsgUpdateParam"
)

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
		migrationtypes.ModuleName:  MigrationModuleParamConfig,
	}

	// paramConfigsPath is the path, relative to the project root, to the go file
	// containing the ModuleParamConfig declaration. It is set in init() and is
	// interpolated into error messages when a module's ModuleParamConfig seems
	// misconfigured (e.g. missing expected values).
	paramConfigsPath string
)

var _ IntegrationSuite = (*ParamsSuite)(nil)

func init() {
	for moduleName, moduleParamCfg := range ModuleParamConfigMap {
		if moduleParamCfg.ParamsMsgs.MsgUpdateParam != nil {
			MsgUpdateParamEnabledModuleNames = append(MsgUpdateParamEnabledModuleNames, moduleName)
		}
	}

	// Take the package path of the ModuleParamConfig type and drop the github.com/<org>/<repo> prefix.
	paramConfigsPkgPath := reflect.TypeOf(ModuleParamConfig{}).PkgPath()
	paramConfigsPathParts := strings.Split(paramConfigsPkgPath, string(os.PathSeparator))
	paramConfigsDir := filepath.Join(paramConfigsPathParts[3:]...)
	// NB: The file name is not included in the package path and must be updated here if it changes.
	paramConfigsPath = filepath.Join(paramConfigsDir, "param_configs.go")
}

// ParamsSuite is an integration test suite that provides helper functions for
// querying module parameters and running parameter update messages. It is
// intended to be embedded in other integration test suites which are dependent
// on parameter queries or updates.
type ParamsSuite struct {
	AuthzIntegrationSuite

	// AuthorityAddr is the cosmos account address of the authority for the integration
	// app. It is used as the **granter** of authz grants for parameter update messages.
	// In practice, is an address sourced by an onchain string and no one has the private key.
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
func (s *ParamsSuite) SetupTestAuthzAccounts(t *testing.T) {
	t.Helper()

	// Set the authority, authorized, and unauthorized addresses.
	s.AuthorityAddr = cosmostypes.MustAccAddressFromBech32(s.GetApp().GetAuthority())

	nextAcct, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.True(t, ok, "insufficient pre-generated accounts available")
	s.AuthorizedAddr = nextAcct.Address
}

// SetupTestAuthzGrants creates onchain authz grants for the MsgUpdateUpdateParam and
// MsgUpdateParams message for each module. It is expected to be called after s.NewApp()
// as it depends on the authority and authorized addresses having been set.
func (s *ParamsSuite) SetupTestAuthzGrants(t *testing.T) {
	t.Helper()

	// Create authz grants for all pocket modules' MsgUpdateParams messages.
	s.RunAuthzGrantMsgForPocketModules(t,
		s.AuthorityAddr,
		s.AuthorizedAddr,
		MsgUpdateParamsName,
		s.GetPocketModuleNames()...,
	)

	// Create authz grants for all pocket modules' MsgUpdateParam messages.
	s.RunAuthzGrantMsgForPocketModules(t,
		s.AuthorityAddr,
		s.AuthorizedAddr,
		MsgUpdateParamName,
		// NB: only modules with params are expected to support MsgUpdateParam.
		MsgUpdateParamEnabledModuleNames...,
	)
}

// RunUpdateParams runs the given MsgUpdateParams message via an authz exec as the
// AuthorizedAddr and returns the response bytes and error. It is expected to be called
// after s.SetupTestAuthzGrants() as it depends on an onchain authz grant to AuthorizedAddr
// for MsgUpdateParams for the given module.
func (s *ParamsSuite) RunUpdateParams(
	t *testing.T,
	msgUpdateParams cosmostypes.Msg,
) (msgResponseBz []byte, err error) {
	t.Helper()

	return s.RunUpdateParamsAsSigner(t, msgUpdateParams, s.AuthorizedAddr)
}

// RunUpdateParamsAsSigner runs the given MsgUpdateParams message via an authz exec
// as signerAddr and returns the response bytes and error. It depends on an onchain
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
// as it depends on an onchain authz grant to AuthorizedAddr for MsgUpdateParam for the given module.
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
// the response bytes and error. It depends on an onchain authz grant to signerAddr for
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

	msgAsTypeStruct, hasParamType := moduleCfg.ParamTypes[paramType]
	require.Truef(t, hasParamType,
		"module %q does not include param type %q in its ModuleParamConfig#ParamTypes; consider updating %s",
		moduleName, paramType, paramConfigsPath,
	)
	msgAsTypeType := reflect.TypeOf(msgAsTypeStruct)
	msgAsTypeValue := reflect.New(msgAsTypeType)
	switch paramType {
	case ParamTypeUint64:
		// =~ msg.AsType.AsUint64 = paramReflectValue.Interface().(uint64)
		msgAsTypeValue.Elem().FieldByName("AsUint64").Set(paramReflectValue)
	case ParamTypeInt64:
		// =~ msg.AsType.AsInt64 = paramReflectValue.Interface().(int64)
		msgAsTypeValue.Elem().FieldByName("AsInt64").Set(paramReflectValue)
	case ParamTypeFloat64:
		// =~ msg.AsType.AsFloat = paramReflectValue.Interface().(float64)
		msgAsTypeValue.Elem().FieldByName("AsFloat").Set(paramReflectValue)
	case ParamTypeString:
		// =~ msg.AsType.AsString = paramReflectValue.Interface().(string)
		msgAsTypeValue.Elem().FieldByName("AsString").Set(paramReflectValue)
	case ParamTypeBytes:
		// =~ msg.AsType.AsBytes = paramReflectValue.Interface().([]byte)
		msgAsTypeValue.Elem().FieldByName("AsBytes").Set(paramReflectValue)
	case ParamTypeCoin:
		// =~ msg.AsType.AsCoin = paramReflectValue.Interface().(*cosmostypes.Coin)
		msgAsTypeValue.Elem().FieldByName("AsCoin").Set(paramReflectValue)
	case ParamTypeMintAllocationPercentages:
		// DEB_NOTE: Params.MintAllocationPercentages is a struct (not a pointer) because
		// it is not nullable. As a result, in this case, we need to create a pointer to
		// assign the paramValue to (because it won't be a pointer itself).
		asMintAllocationPercentagesField := msgAsTypeValue.Elem().FieldByName("AsMintAllocationPercentages")
		// =~ msg.AsType.AsMintAllocationPercentages = new(MsgUpdateParam_AsMintAllocationPercentages)
		asMintAllocationPercentagesField.Set(reflect.New(paramReflectValue.Type()))
		// =~ *msg.AsType.AsMintAllocationPercentages = paramReflectValue.Interface().(MintAllocationPercentages)
		asMintAllocationPercentagesField.Elem().Set(paramReflectValue)
	default:
		t.Fatalf("ERROR: unknown field type %q", paramType)
	}

	msgUpdateParamValue.Elem().FieldByName("AsType").Set(msgAsTypeValue)

	msgUpdateParam := msgUpdateParamValue.Interface().(cosmostypes.Msg)

	// Send an authz MsgExec from the authority address.
	execMsg := authz.NewMsgExec(signerAddr, []cosmostypes.Msg{msgUpdateParam})
	msgRespsBz, err := s.RunAuthzExecMsg(t, signerAddr, &execMsg)
	if err != nil {
		return nil, err
	}

	require.Equal(t, 1, len(msgRespsBz), "expected exactly 1 message response")
	return msgRespsBz[0], err
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
