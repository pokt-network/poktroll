//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// txDelaySeconds is the number of seconds to wait for a tx to be committed before making assertions.
	txDelaySeconds = 3
	// txFeesCoinStr is the string representation of the amount & denom of tokens
	// which are sufficient to pay for tx fees in the test.
	txFeesCoinStr = "1000000upokt"
	// PNF is the account that acts on behalf of the DAO and is therefore the only
	// one authorized to perform certain actions such as updating params.
	pnfKeyName = "pnf"
)

// AllModuleParamsAreSetToTheirDefaultValues asserts that all module params are set to their default values.
func (s *suite) AllModuleParamsAreSetToTheirDefaultValues(moduleName string) {
	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		var tokenomicsParamsRes tokenomicstypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &tokenomicsParamsRes)
		require.Equal(s, tokenomicstypes.DefaultParams(), tokenomicsParamsRes.GetParams())

	case prooftypes.ModuleName:
		var proofParamsRes prooftypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &proofParamsRes)
		require.Equal(s, prooftypes.DefaultParams(), proofParamsRes.GetParams())

	case sessiontypes.ModuleName:
		var sessionParamsRes sessiontypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &sessionParamsRes)
		require.Equal(s, sessiontypes.DefaultParams(), sessionParamsRes.GetParams())

	case apptypes.ModuleName:
		var appParamsRes apptypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &appParamsRes)
		require.Equal(s, apptypes.DefaultParams(), appParamsRes.GetParams())

	case gatewaytypes.ModuleName:
		var gatewayParamsRes gatewaytypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &gatewayParamsRes)
		require.Equal(s, gatewaytypes.DefaultParams(), gatewayParamsRes.GetParams())

	case suppliertypes.ModuleName:
		var supplierParamsRes suppliertypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &supplierParamsRes)
		require.Equal(s, suppliertypes.DefaultParams(), supplierParamsRes.GetParams())

	case servicetypes.ModuleName:
		var serviceParamsRes servicetypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &serviceParamsRes)
		require.Equal(s, servicetypes.DefaultParams(), serviceParamsRes.GetParams())

	case sharedtypes.ModuleName:
		var sharedParamsRes sharedtypes.QueryParamsResponse
		s.cdc.MustUnmarshalJSON([]byte(res.Stdout), &sharedParamsRes)
		require.Equal(s, sharedtypes.DefaultParams(), sharedParamsRes.GetParams())

	default:
		s.Fatalf("ERROR: unexpected module name: (%v)", moduleName)
	}
}

// AnAuthzGrantFromTheAccountToTheAccountForTheMessage queries the authz module for grants
// with the expected granter & grantee (authz.QueryGrantsRequest) & asserts that the expected
// grant is found in the response.
func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForTheMessageExists(
	granterName string,
	granterAddrType string,
	granteeName string,
	granteeAddrType string,
	msgType string,
) {
	// Declare granter & grantee addresses for use in the authz CLI query sub-command.
	var granterAddr, granteeAddr string
	nameToAddrMap := map[string]*string{
		granterName: &granterAddr,
		granteeName: &granteeAddr,
	}
	nameToAddrTypeMap := map[string]string{
		granterName: granterAddrType,
		granteeName: granteeAddrType,
	}

	// Set the granter & grantee addresses based on the address type.
	for name, addr := range nameToAddrMap {
		switch nameToAddrTypeMap[name] {
		case "module":
			*addr = authtypes.NewModuleAddress(name).String()
		case "user":
			*addr = s.getKeyAddress(name)
		default:
			s.Fatal("unexpected address type")
		}
	}

	// Query the authz module for grants with the expected granter & grantee (authz.QueryGrantsRequest).
	argsAndFlags := []string{
		"query", "authz", "grants",
		granterAddr, granteeAddr, msgType,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// Unmarshal the JSON response into the grantsRes struct.
	// Using s.cdc.UnmarshalJSON() with *authz.QueryGrantsResponse complains about
	// the "type" field, expecting "@type" for the #Grants[n]#Authorization pb.Any.
	grantsRes := new(authzCLIGrantResponse)
	responseBz := []byte(strings.TrimSpace(res.Stdout))
	err = json.Unmarshal(responseBz, grantsRes)
	require.NoError(s, err)

	// Check if the expected grant is found in the response.
	grantFound := false
	for _, grant := range grantsRes.Grants {
		if grant.Authorization.Value.Msg == msgType {
			grantFound = true
			break
		}
	}
	require.True(s, grantFound)

	// Update the state of the suite with the granter & grantee names.
	s.granterName = granterName
	s.granteeName = granteeName
}

// AllModuleParamsShouldBeSetToTheirDefaultValues asserts that all module params are set to their default values.
func (s *suite) AllModuleParamsShouldBeSetToTheirDefaultValues(moduleName string) {
	s.AllModuleParamsAreSetToTheirDefaultValues(moduleName)
}

// TheAccountSendsAnAuthzExecMessageToUpdateAllModuleParams sends an authz exec
// message to update all module params for the given module.
func (s *suite) TheAccountSendsAnAuthzExecMessageToUpdateAllModuleParams(accountName, moduleName string, table gocuke.DataTable) {
	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParams = moduleParamsMap{
		moduleName: s.parseParamsTable(table),
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamsTxJSONFile(s.expectedModuleParams)

	// Send the authz exec tx to update all module params.
	s.sendAuthzExecTx(accountName, txJSONFile.Name())
}

// AllModuleParamsShouldBeUpdated asserts that all module params have been updated as expected.
func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string) {
	_, ok := s.expectedModuleParams[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	s.assertExpectedModuleParamsUpdated(moduleName)
}

// TheAccountSendsAnAuthzExecMessageToUpdateTheModuleParam sends an authz exec message to update a single module param.
func (s *suite) TheAccountSendsAnAuthzExecMessageToUpdateTheModuleParam(accountName, moduleName string, table gocuke.DataTable) {
	// NB: skip the header row & only expect a single row.
	param := s.parseParam(table, 1)

	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParams = moduleParamsMap{
		moduleName: {
			param.name: param,
		},
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamTxJSONFile(s.expectedModuleParams)

	// Send the authz exec tx to update the module param.
	s.sendAuthzExecTx(accountName, txJSONFile.Name())
}

// TheModuleParamShouldBeUpdated asserts that the module param has been updated as expected.
func (s *suite) TheModuleParamShouldBeUpdated(moduleName, paramName string) {
	moduleParamsMap, ok := s.expectedModuleParams[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	var foundExpectedParam bool
	for expectedParamName, _ := range moduleParamsMap {
		if paramName == expectedParamName {
			foundExpectedParam = true
			break
		}
	}
	require.True(s, foundExpectedParam, "param %q expectation not set on the test suite", paramName)

	s.assertExpectedModuleParamsUpdated(moduleName)
}

// AllModuleParamsShouldBeSetToTheirDefaultValues asserts that all module params are set to their default values.
func (s *suite) AllModuleParamsAreResetToTheirDefaultValues() {
	var anyMsgs []*types.Any
	authority := authtypes.NewModuleAddress(s.granterName).String()

	// List of all module MsgUpdateParams types and their respective default param functions
	modules := []struct {
		msgUpdateParamsType reflect.Type
		defaultParams       any
	}{
		{reflect.TypeOf(&apptypes.MsgUpdateParams{}), apptypes.DefaultParams()},
		{reflect.TypeOf(&gatewaytypes.MsgUpdateParams{}), gatewaytypes.DefaultParams()},
		{reflect.TypeOf(&prooftypes.MsgUpdateParams{}), prooftypes.DefaultParams()},
		{reflect.TypeOf(&servicetypes.MsgUpdateParams{}), servicetypes.DefaultParams()},
		{reflect.TypeOf(&sessiontypes.MsgUpdateParams{}), sessiontypes.DefaultParams()},
		{reflect.TypeOf(&sharedtypes.MsgUpdateParams{}), sharedtypes.DefaultParams()},
		{reflect.TypeOf(&suppliertypes.MsgUpdateParams{}), suppliertypes.DefaultParams()},
		{reflect.TypeOf(&tokenomicstypes.MsgUpdateParams{}), tokenomicstypes.DefaultParams()},
	}

	for _, module := range modules {
		msgUpdateParams := reflect.New(module.msgUpdateParamsType.Elem()).Interface().(proto.Message)
		msgUpdateParamsValue := reflect.ValueOf(msgUpdateParams).Elem()
		msgUpdateParamsValue.FieldByName("Authority").SetString(authority)
		msgUpdateParamsValue.FieldByName("Params").Set(reflect.ValueOf(module.defaultParams))

		anyMsg, err := types.NewAnyWithValue(msgUpdateParams)
		require.NoError(s, err)
		anyMsgs = append(anyMsgs, anyMsg)
	}

	file := s.newTempTxJSONFile(anyMsgs)
	s.sendAuthzExecTx(authority, file.Name())
}

// TheModuleParamShouldBeSetToItsDefaultValue asserts that the given param for the
// given module has been set to its default value.
func (s *suite) TheModuleParamShouldBeSetToItsDefaultValue(moduleName, paramName string) {
	// TODO_HACK: So long as no other modules are expected to have been changed by this scenario,
	// it is more than sufficient (and less code) to re-use the existing step which asserts that
	// all modules have their params set to their respective defaults.
	_ = paramName
	s.AllModuleParamsShouldBeSetToTheirDefaultValues(moduleName)
}

// ensureAccountForKeyName ensures that an account exists for the given key name in the keychain.
func (s *suite) ensureAccountForKeyName(keyName string) {
	s.Helper()

	// Get the address of the key.
	addr := s.getKeyAddress(keyName)

	// Fund the account with minimal tokens to ensure it can afford tx fees.
	coin, err := cosmostypes.ParseCoinNormalized(txFeesCoinStr)
	require.NoError(s, err)
	s.fundAddress(addr, coin)
}

// fundAddress sends the given amount & demon of tokens to the given address.
func (s *suite) fundAddress(addr string, coin cosmostypes.Coin) {
	s.Helper()

	// poktrolld tx bank send <from> <to> <amount> --keyring-backend test --chain-id <chain_id> --yes
	argsAndFlags := []string{
		"tx",
		"bank",
		"send",
		pnfKeyName,
		addr,
		coin.String(),
		"--yes",
	}

	_, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// TODO_IMPROVE: wait for the tx to be committed using an events query client
	// instead of sleeping for a specific amount of time.
	s.Logf("waiting %d seconds for the funding tx to be committed...", txDelaySeconds)
	time.Sleep(txDelaySeconds * time.Second)
}

// getKeyAddress uses the `keys show` CLI subcommand to get the address of a key.
func (s *suite) getKeyAddress(keyName string) string {
	s.Helper()

	argsAndFlags := []string{
		"keys",
		"show",
		keyName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("local", argsAndFlags...)
	require.NoError(s, err)

	keyListRes := make(map[string]any)
	err = json.Unmarshal([]byte(res.Stdout), &keyListRes)
	require.NoError(s, err)

	return keyListRes["address"].(string)
}

func (s *suite) assertExpectedModuleParamsUpdated(moduleName string) {
	s.Helper()

	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		computeUnitsToTokensMultiplier := uint64(s.expectedModuleParams[moduleName][tokenomicstypes.ParamComputeUnitsToTokensMultiplier].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&tokenomicstypes.QueryParamsResponse{
				Params: tokenomicstypes.Params{
					ComputeUnitsToTokensMultiplier: computeUnitsToTokensMultiplier,
				},
			},
		)
	case prooftypes.ModuleName:
		minRelayDifficultyBits := uint64(s.expectedModuleParams[moduleName][prooftypes.ParamMinRelayDifficultyBits].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&prooftypes.QueryParamsResponse{
				Params: prooftypes.Params{
					MinRelayDifficultyBits: minRelayDifficultyBits,
				},
			},
		)
	case sharedtypes.ModuleName:
		params := sharedtypes.DefaultParams()
		paramsMap := s.expectedModuleParams[moduleName]

		numBlocksPerSessionParam, ok := paramsMap[sharedtypes.ParamNumBlocksPerSession]
		if ok {
			params.NumBlocksPerSession = uint64(numBlocksPerSessionParam.value.(int64))
		}

		claimWindowOpenOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamClaimWindowOpenOffsetBlocks]
		if ok {
			params.ClaimWindowOpenOffsetBlocks = uint64(claimWindowOpenOffsetBlocksParam.value.(int64))
		}

		claimWindowCloseOffsetBlocksParam, ok := paramsMap[sharedtypes.ParamClaimWindowCloseOffsetBlocks]
		if ok {
			params.ClaimWindowCloseOffsetBlocks = uint64(claimWindowCloseOffsetBlocksParam.value.(int64))
		}

		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&sharedtypes.QueryParamsResponse{
				Params: params,
			},
		)
	case apptypes.ModuleName:
		maxDelegatedGateways := uint64(s.expectedModuleParams[moduleName][apptypes.ParamMaxDelegatedGateways].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&apptypes.QueryParamsResponse{
				Params: apptypes.Params{
					MaxDelegatedGateways: maxDelegatedGateways,
				},
			},
		)
	case servicetypes.ModuleName:
		addServiceFee := uint64(s.expectedModuleParams[moduleName][servicetypes.ParamAddServiceFee].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&servicetypes.QueryParamsResponse{
				Params: servicetypes.Params{
					AddServiceFee: addServiceFee,
				},
			},
		)
	default:
		s.Fatalf("ERROR: unexpected module name %q", moduleName)
	}
}

// assertUpdatedParams deserializes the param query response JSON into a
// MsgUpdateParams of type P & asserts that it matches the expected params.
func assertUpdatedParams[P cosmostypes.Msg](
	s *suite,
	queryParamsResJSON []byte,
	expectedParamsRes P,
) {
	s.Helper()

	queryParamsMsgValue := reflect.New(reflect.TypeOf(expectedParamsRes).Elem())
	queryParamsMsg := queryParamsMsgValue.Interface().(P)
	err := s.cdc.UnmarshalJSON(queryParamsResJSON, queryParamsMsg)
	require.NoError(s, err)
	require.EqualValues(s, expectedParamsRes, queryParamsMsg)
}
