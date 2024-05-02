//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testclient"
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

// txTemplateLocals is the template locals struct used to populate the
// updateParamsTxJSONTemplate template.
type txTemplateLocals struct {
	Type      string
	Authority string
	Params    string
}

// updateParamsTxJSONTemplate is a text template for a tx JSON file which is
// intended to be used with the `authz exec` CLI subcommand.
var updateParamsTxJSONTemplate = template.Must(
	template.
		New("txJSON").
		Parse(`{
	"body": {
	  "messages": [
		  {
			"@type": "{{.Type}}",
			"authority": "{{.Authority}}",
			"params": {{.Params}}
		  }
	  ]
	}
}
`))

func (s *suite) AllModuleParamsAreSetToTheirDefaultValues(moduleName string) {
	s.Skip("TODO_TEST: complete step definitions for update_params.feature scenarios")

	argsAndFlags := []string{
		"query",
		moduleName,
		"params",
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	s.Log(res.Stdout)

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

func (s *suite) AnAuthzGrantFromTheAccountToTheAccountForTheMessage(granterName, granterAddrType, granteeName, granteeAddrType string, msgType string) {
	var granterAddr, granteeAddr string
	nameToAddrMap := map[string]*string{
		granterName: &granterAddr,
		granteeName: &granteeAddr,
	}
	nameToAddrTypeMap := map[string]string{
		granterName: granterAddrType,
		granteeName: granteeAddrType,
	}

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
	ctx := context.Background()

	// Convert the text table to a map of param names to their types & values.
	paramsMap := s.parseParamsTable(table)

	// Use the map of params to populate a tx JSON template & write it to a file.
	txJSONPath := s.newTempTxJSONFile(moduleName, paramsMap)

	argsAndFlags := []string{
		"tx", "authz", "exec",
		txJSONPath,
		"--from", s.granteeName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	_, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// Reset all module params to their default values after the test completes.
	s.Cleanup(func() { s.resetAllModuleParamsToDefaults(ctx) })
}

func (s *suite) AllModuleParamsShouldBeUpdated(moduleName string, table gocuke.DataTable) {
	panic("PENDING")
}

func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateModuleParam(moduleName, paramName string, table gocuke.DataTable) {
	panic("PENDING")
}

func (s *suite) TheModuleParamShouldBeUpdated(moduleName, paramName string, table gocuke.DataTable) {
	panic("PENDING")
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
	for rowIdx := 1; rowIdx < table.NumRows()-1; rowIdx++ {
		var paramValue any
		paramName := table.Cell(rowIdx, 0).String()
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

		paramsMap[paramName] = struct {
			paramType  string
			paramValue any
		}{
			paramType:  paramType,
			paramValue: paramValue,
		}
	}

	return paramsMap
}

func (s *suite) paramsMapToMsgUpdateParams(moduleName string, paramsMap map[string]anyMap) (msg proto.Message, paramsJSON []byte) {
	switch moduleName {
	case tokenomicstypes.ModuleName:
		msgUpdateParams := &tokenomicstypes.MsgUpdateParams{Params: tokenomicstypes.Params{}}
		for paramName, param := range paramsMap {
			switch paramName {
			case "compute_units_to_tokens_multiplier":
				msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(param.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", param.paramType, paramName)
			}
		}
		paramsJSON, err := s.cdc.MarshalJSON(&msgUpdateParams.Params)
		require.NoError(s, err)
		return proto.Message(msgUpdateParams), paramsJSON

	case prooftypes.ModuleName:
		msgUpdateParams := &prooftypes.MsgUpdateParams{Params: prooftypes.Params{}}
		for paramName, param := range paramsMap {
			switch paramName {
			case "min_relay_difficulty_bits":
				msgUpdateParams.Params.MinRelayDifficultyBits = uint64(param.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", param.paramType, paramName)
			}
		}
		paramsJSON, err := s.cdc.MarshalJSON(&msgUpdateParams.Params)
		require.NoError(s, err)
		return proto.Message(msgUpdateParams), paramsJSON

	default:
		s.Fatalf("unexpected module name %q", moduleName)
		return nil, nil
	}
}

// newTempTxJSONFile creates a new temp file with the JSON representation of a tx,
// intended for use with the `authz exec` CLI subcommand. It returns the file path.
func (s *suite) newTempTxJSONFile(moduleName string, paramsMap map[string]anyMap) string {
	buf := new(bytes.Buffer)
	locals := s.newTxTemplateLocals(moduleName, paramsMap)
	require.NoError(s, updateParamsTxJSONTemplate.Execute(buf, locals))

	tempFile, err := os.CreateTemp("", "exec.json")
	require.NoError(s, err)

	// Remove tempFile when the test completes.
	s.Cleanup(func() {
		os.Remove(tempFile.Name())
	})

	// TODO_IMPROVE: find a better and/or more conventional way to programmatically generate tx JSON containing pb.Any messages.
	// - 👍 s.cdc.MarshalJSON() converts the pb.Any message to a JSON object with a "@type" field.
	// - 👎 s.cdc.MarshalJSON() the value of the "@type" field is missing a preceding "/" that authz exec CLI is expecting.
	// - 👎 EOF error currently when calling authz exec CLI with the generated JSON.
	// - 👎 s.cdc.MarshalJSON() can only be applied to proto.Message implementations (i.e. protobuf types).
	// - ? is there a tx-level object that can be used to serialize & which can be constructed in this scope?
	// - 👎 otherwise, must use a template for the tx JSON structure (s.cdc.MarshalJSON() can't be used on a tx.Tx object).

	replacedJSON := bytes.Replace(buf.Bytes(), []byte(`"@type": "`), []byte(`"@type": "/`), 1)
	_, err = tempFile.Write(replacedJSON)
	require.NoError(s, err)

	return tempFile.Name()
}

func (s *suite) newTxTemplateLocals(moduleName string, paramsMap map[string]anyMap) *txTemplateLocals {
	// Convert the params map to a MsgUpdateParams message & JSON.
	msgUpdateParams, paramsJSON := s.paramsMapToMsgUpdateParams(moduleName, paramsMap)

	return &txTemplateLocals{
		Type:      proto.MessageName(msgUpdateParams),
		Authority: authtypes.NewModuleAddress(s.granterName).String(),
		Params:    string(paramsJSON),
	}
}

// resetAllModuleParamsToDefaults resets all module params to their default values using
// a single authz exec message. It blocks until the resulting tx has been committed.
func (s *suite) resetAllModuleParamsToDefaults(ctx context.Context) {
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	authzClient := authz.NewMsgClient(clientCtx)

	proofDefaultParams := prooftypes.DefaultParams()
	proofDefaultParamsAny, err := codectypes.NewAnyWithValue(&proofDefaultParams)
	require.NoError(s, err)

	tokenomicsDefaultParams := tokenomicstypes.DefaultParams()
	tokenomicsDefaultParamsAny, err := codectypes.NewAnyWithValue(&tokenomicsDefaultParams)
	require.NoError(s, err)

	msgExec := &authz.MsgExec{
		Grantee: authtypes.NewModuleAddress(s.granteeName).String(),
		Msgs: []*codectypes.Any{
			proofDefaultParamsAny,
			tokenomicsDefaultParamsAny,
		},
	}

	_, err = authzClient.Exec(ctx, msgExec)
	require.NoError(s, err)
}
