//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// txDelaySeconds is the number of seconds to wait for a tx to be committed before making assertions.
const txDelaySeconds = 3

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

	default:
		s.Fatalf("unexpected module name: (%v)", moduleName)
	}
}

// AnAuthzGrantFromTheAccountToTheAccountForTheMessage queries the authz module for grants
// with the expected granter & grantee (authz.QueryGrantsRequest) & asserts that the expected
// grant is found in the response.
func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForTheMessage(
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

// TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams sends an authz exec
// message to update all module params for the given module.
func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams(moduleName string, table gocuke.DataTable) {
	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParams = moduleParamsMap{
		moduleName: s.parseParamsTable(table),
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamsTxJSONFile(s.expectedModuleParams)

	// Send the authz exec tx to update all module params.
	s.sendAuthzExecTx(txJSONFile.Name())
}

// AllModuleParamsShouldBeUpdated asserts that all module params have been updated as expected.
func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string) {
	_, ok := s.expectedModuleParams[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	s.assertExpectedModuleParamsUpdated(moduleName)
}

// TheUserSendAnAuthzExecMessageToUpdateTheModuleParam sends an authz exec message to update a single module param.
func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateTheModuleParam(moduleName string, table gocuke.DataTable) {
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
	s.sendAuthzExecTx(txJSONFile.Name())
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

// getKeyAddress uses the `keys show` CLI subcommand to get the address of a key.
func (s *suite) getKeyAddress(keyName string) string {
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
		computeUnitsToTokensMultiplier := uint64(s.expectedModuleParams[moduleName][computeUnitsToTokensMultipler].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&tokenomicstypes.QueryParamsResponse{
				Params: tokenomicstypes.Params{
					ComputeUnitsToTokensMultiplier: computeUnitsToTokensMultiplier,
				},
			},
		)
	case prooftypes.ModuleName:
		minRelayDifficultyBits := uint64(s.expectedModuleParams[moduleName][minRelayDifficultyBits].value.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&prooftypes.QueryParamsResponse{
				Params: prooftypes.Params{
					MinRelayDifficultyBits: minRelayDifficultyBits,
				},
			},
		)
	default:
		s.Fatalf("unexpected module name %q", moduleName)
	}
}

// assertUpdatedParams deserializes the param query response JSON into a
// MsgUpdateParams of type P & asserts that it matches the expected params.
func assertUpdatedParams[P cosmostypes.Msg](
	s *suite,
	queryParamsResJSON []byte,
	expectedParamsRes P,
) {
	queryParamsMsgValue := reflect.New(reflect.TypeOf(expectedParamsRes).Elem())
	queryParamsMsg := queryParamsMsgValue.Interface().(P)
	err := s.cdc.UnmarshalJSON(queryParamsResJSON, queryParamsMsg)
	require.NoError(s, err)
	require.EqualValues(s, expectedParamsRes, queryParamsMsg)
}
