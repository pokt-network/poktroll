//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
	"time"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// authzCLIGrantResponse is the JSON response struct for the authz grants query CLI subcommand.
type authzCLIGrantResponse struct {
	Grants []struct {
		Authorization struct {
			Type  string `json:"type"`
			Value struct {
				Msg string `json:"msg"`
			} `json:"value"`
		} `json:"authorization"`
	} `json:"grants"`
}

// txDelaySeconds is the number of seconds to wait for a tx to be committed before making assertions.
const txDelaySeconds = 3

// updateParamsTxJSONTemplate is a text template for a tx JSON file which is
// intended to be used with the `authz exec` CLI subcommand.
var updateParamsTxJSONTemplate = template.Must(
	template.New("txJSON").Parse(`{ "body": {{.Body}} }`),
)

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

func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams(moduleName string, table gocuke.DataTable) {
	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParamsMap = map[string]map[string]anyMap{
		moduleName: s.parseParamsTable(table),
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamsTxJSONFile(s.expectedModuleParamsMap)

	// Send the authz exec tx to update all module params.
	s.sendAuthzExecTx(txJSONFile.Name())
}

// TODO_IN_THIS_COMMIT: move
func (s *suite) sendAuthzExecTx(txJSONFilePath string) {
	argsAndFlags := []string{
		"tx", "authz", "exec",
		txJSONFilePath,
		"--from", s.granteeName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
		"--yes",
	}
	_, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// TODO_IMPROVE: wait for the tx to be committed using an events query client
	// instead of sleeping for a specific amount of time.
	s.Logf("waiting %d seconds for the tx to be committed...", txDelaySeconds)
	time.Sleep(txDelaySeconds * time.Second)

	// Reset all module params to their default values after the test completes.
	s.once.Do(func() {
		s.Cleanup(func() { s.resetAllModuleParamsToDefaults(s.ctx) })
	})
}

func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string) {
	_, ok := s.expectedModuleParamsMap[moduleName]
	require.True(s, ok, "module %q params expectation not set on the test suite", moduleName)

	s.assertExpectedModuleParamsUpdated(moduleName)
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
		// TODO_IN_THIS_COMMIT: move string key to a const.
		computeUnitsToTokensMultiplier := uint64(s.expectedModuleParamsMap[moduleName]["compute_units_to_tokens_multiplier"].paramValue.(int64))
		assertUpdatedParams(s,
			[]byte(res.Stdout),
			&tokenomicstypes.QueryParamsResponse{
				Params: tokenomicstypes.Params{
					ComputeUnitsToTokensMultiplier: computeUnitsToTokensMultiplier,
				},
			},
		)
	case prooftypes.ModuleName:
		// TODO_IN_THIS_COMMIT: move string key to a const.
		minRelayDifficultyBits := uint64(s.expectedModuleParamsMap[moduleName]["min_relay_difficulty_bits"].paramValue.(int64))
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

// TODO_IN_THIS_COMMIT: refactor with `#TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams`.
func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateTheModuleParam(moduleName string, table gocuke.DataTable) {
	// NB: skip the header row & only expect a single row.
	paramName, paramValueType := s.parseParam(table, 1)

	// NB: set s#moduleParamsMap for later assertion.
	s.expectedModuleParamsMap = map[string]map[string]anyMap{
		moduleName: {
			paramName: paramValueType,
		},
	}

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONFile := s.newTempUpdateParamTxJSONFile(s.expectedModuleParamsMap)

	// Send the authz exec tx to update the module param.
	s.sendAuthzExecTx(txJSONFile.Name())
}

func (s *suite) TheModuleParamShouldBeUpdated(moduleName, paramName string) {
	moduleParamsMap, ok := s.expectedModuleParamsMap[moduleName]
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

type anyMap struct {
	paramType  string
	paramValue any
}

func (s *suite) parseParamsTable(table gocuke.DataTable) map[string]anyMap {
	paramsMap := make(map[string]anyMap)

	// NB: skip the header row.
	for rowIdx := 1; rowIdx < table.NumRows(); rowIdx++ {
		paramName, paramValueType := s.parseParam(table, rowIdx)
		paramsMap[paramName] = paramValueType
	}

	return paramsMap
}

func (s *suite) parseParam(table gocuke.DataTable, rowIdx int) (paramName string, paramValueType anyMap) {
	var paramValue any
	paramName = table.Cell(rowIdx, 0).String()
	paramType := table.Cell(rowIdx, 2).String()

	switch paramType {
	case "string":
		paramValue = table.Cell(rowIdx, 1).String()
	case "int64":
		paramValue = table.Cell(rowIdx, 1).Int64()
	case "bytes":
		paramValue = []byte(table.Cell(rowIdx, 1).String())
	default:
		s.Fatalf("unexpected param type %q", paramType)
	}

	return paramName, anyMap{
		paramType:  paramType,
		paramValue: paramValue,
	}
}

func (s *suite) paramsMapToMsgUpdateParams(moduleName string, paramsMap map[string]anyMap) (msg cosmostypes.Msg) {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	switch moduleName {
	case tokenomicstypes.ModuleName:
		msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
			Authority: authority,
			Params:    tokenomicstypes.Params{},
		}

		for paramName, paramValue := range paramsMap {
			//s.Logf("paramName: %s, paramValue: %v", paramName, paramValue.paramValue)
			switch paramName {
			case "compute_units_to_tokens_multiplier":
				msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(paramValue.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", paramValue.paramType, paramName)
			}
		}
		msg = proto.Message(msgUpdateParams)

	case prooftypes.ModuleName:
		msgUpdateParams := &prooftypes.MsgUpdateParams{
			Authority: authority,
			Params:    prooftypes.Params{},
		}

		for paramName, paramValue := range paramsMap {
			s.Logf("paramName: %s, paramValue: %v", paramName, paramValue.paramValue)
			switch paramName {
			case "min_relay_difficulty_bits":
				msgUpdateParams.Params.MinRelayDifficultyBits = uint64(paramValue.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", paramValue.paramType, paramName)
			}
		}
		msg = proto.Message(msgUpdateParams)

	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}

// TODO_IN_THIS_COMMIT: refactor with `#paramsMapToMsgUpdateParams`.
func (s *suite) paramsMapToMsgUpdateParam(moduleName, paramName string, paramTypeValue anyMap) (msg cosmostypes.Msg) {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	//switch paramTypeValue.paramType {
	//case "string":
	//	msg = paramMsgUpdateFromParam[tokenomicstypes.MsgUpdateParam_AsString](
	//		moduleName,
	//		authority,
	//		paramName,
	//		paramTypeValue,
	//	)
	//default:
	//	s.Fatalf("unexpected param type %q", paramTypeValue.paramType)
	//}

	// TODO_IMPROVE: refactor/simplify?
	switch moduleName {
	case tokenomicstypes.ModuleName:
		//msg := &tokenomicstypes.MsgUpdateParam{
		//	Authority: authority,
		//	Name:      paramName,
		//	AsType: paramAsType(paramTypeValue),
		//}

		switch paramTypeValue.paramType {
		case "string":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &tokenomicstypes.MsgUpdateParam_AsString{
					AsString: paramTypeValue.paramValue.(string),
				},
			})
		case "int64":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &tokenomicstypes.MsgUpdateParam_AsInt64{
					AsInt64: paramTypeValue.paramValue.(int64),
				},
			})
		case "bytes":
			msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &tokenomicstypes.MsgUpdateParam_AsBytes{
					AsBytes: paramTypeValue.paramValue.([]byte),
				},
			})
		}
	case prooftypes.ModuleName:
		switch paramTypeValue.paramType {
		case "string":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &prooftypes.MsgUpdateParam_AsString{
					AsString: paramTypeValue.paramValue.(string),
				},
			})
		case "int64":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &prooftypes.MsgUpdateParam_AsInt64{
					AsInt64: paramTypeValue.paramValue.(int64),
				},
			})
		case "bytes":
			msg = proto.Message(&prooftypes.MsgUpdateParam{
				Authority: authority,
				Name:      paramName,
				AsType: &prooftypes.MsgUpdateParam_AsBytes{
					AsBytes: paramTypeValue.paramValue.([]byte),
				},
			})
		}
	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}

//// TODO_IN_THIS_COMMIT: move (to bottom)
//// func paramAsType(paramTypeValue anyMap)
//func paramMsgUpdateFromParam[T tokenomicstypes.IsMsgUpdateParams_AsType](
//	moduleName,
//	authority,
//	paramName string,
//	asType T,
//) (msg cosmostypes.Msg) {
//	switch moduleName {
//	case tokenomicstypes.ModuleName:
//		msg = &tokenomicstypes.MsgUpdateParam{
//			Authority: authority,
//			Name:      paramName,
//			AsType:    asType,
//		}
//	}
//}

// newTempTxJSONFile creates a new temp file with the JSON representation of a tx,
// intended for use with the `authz exec` CLI subcommand. It returns the file path.
func (s *suite) newTempUpdateParamsTxJSONFile(moduleParamsMap map[string]map[string]anyMap) *os.File {
	var anyMsgs []*codectypes.Any

	for moduleName, paramsMap := range moduleParamsMap {
		// Convert the params map to a MsgUpdateParams message.
		msg := s.paramsMapToMsgUpdateParams(moduleName, paramsMap)

		// Convert the MsgUpdateParams message to a pb.Any message.
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(s, err)

		anyMsgs = append(anyMsgs, anyMsg)
	}

	return s.newTempTxJSONFile(anyMsgs)
}

// TODO_IN_THIS_COMMIT: godoc comment...
// TODO_IN_THIS_COMMIT: refactor with `#newTempTxJSONFile`
func (s *suite) newTempUpdateParamTxJSONFile(moduleParamsMap map[string]map[string]anyMap) *os.File {
	var anyMsgs []*codectypes.Any

	for moduleName, paramsMap := range moduleParamsMap {
		for paramName, paramTypeValueMap := range paramsMap {
			// Convert the params map to a MsgUpdateParams message.
			//msg := s.paramsMapToMsgUpdateParams(moduleName, paramsMap)
			msg := s.paramsMapToMsgUpdateParam(moduleName, paramName, paramTypeValueMap)

			// Convert the MsgUpdateParams message to a pb.Any message.
			anyMsg, err := codectypes.NewAnyWithValue(msg)
			require.NoError(s, err)

			anyMsgs = append(anyMsgs, anyMsg)
		}
	}

	return s.newTempTxJSONFile(anyMsgs)
}

func (s *suite) newTempTxJSONFile(anyMsgs []*codectypes.Any) *os.File {
	// Construct a TxBody with the pb.Any message for serialization.
	txBody := &tx.TxBody{
		Messages: anyMsgs,
	}

	// Serialize txBody to JSON for interpolation into the tx JSON template.
	txBodyJSON, err := s.cdc.MarshalJSON(txBody)
	require.NoError(s, err)

	// Create a temporary file to write the interpolated tx JSON.
	tempFile, err := os.CreateTemp("", "exec.json")
	require.NoError(s, err)

	defer func(f *os.File) {
		_ = f.Close()
	}(tempFile)

	// Remove tempFile when the test completes.
	s.Cleanup(func() {
		_ = os.Remove(tempFile.Name())
	})

	// Interpolate txBodyJSON into the tx JSON template.
	err = updateParamsTxJSONTemplate.Execute(
		tempFile,
		struct{ Body string }{
			Body: string(txBodyJSON),
		},
	)
	require.NoError(s, err)

	return tempFile
}

// resetAllModuleParamsToDefaults resets all module params to their default values using
// a single authz exec message. It blocks until the resulting tx has been committed.
func (s *suite) resetAllModuleParamsToDefaults(ctx context.Context) {
	s.Log("resetting all module params to their default values")

	// Tokenomics module default params.
	tokenomicsMsgUpdateParamsToDefaultsAny, err := codectypes.NewAnyWithValue(
		&tokenomicstypes.MsgUpdateParams{
			Authority: authtypes.NewModuleAddress(s.granterName).String(),
			Params:    tokenomicstypes.DefaultParams(),
		},
	)
	require.NoError(s, err)

	// Proof module default params.
	proofMsgUpdateParamsToDefaultsAny, err := codectypes.NewAnyWithValue(
		&prooftypes.MsgUpdateParams{
			Authority: authtypes.NewModuleAddress(s.granterName).String(),
			Params:    prooftypes.DefaultParams(),
		},
	)
	require.NoError(s, err)

	anyMsgs := []*codectypes.Any{
		tokenomicsMsgUpdateParamsToDefaultsAny,
		proofMsgUpdateParamsToDefaultsAny,
	}

	resetTxJSONFile := s.newTempTxJSONFile(anyMsgs)

	s.sendAuthzExecTx(resetTxJSONFile.Name())
}

// TODO_IN_THIS_COMMIT: godoc comment...
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
