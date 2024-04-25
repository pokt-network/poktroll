//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

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

	args := strings.Split(fmt.Sprintf("query %s params", moduleName), " ")
	args = append(args, "--output", "json")
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	s.Log(res.Stdout)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		var tokenomicsParamsRes tokenomicstypes.QueryParamsResponse
		err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &tokenomicsParamsRes)
		require.NoError(s, err)
		require.Equal(s, tokenomicstypes.DefaultParams(), tokenomicsParamsRes.GetParams())

	case prooftypes.ModuleName:
		var proofParamsRes prooftypes.QueryParamsResponse
		err = s.cdc.UnmarshalJSON([]byte(res.Stdout), &proofParamsRes)
		require.NoError(s, err)
		require.Equal(s, prooftypes.DefaultParams(), proofParamsRes.GetParams())

	default:
		s.Fatal("unexpected module name")
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
	args := strings.Split(
		fmt.Sprintf(
			"query authz grants %s %s %s --output json",
			granterAddr, granteeAddr, msgType,
		), " ",
	)
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	// TODO_IN_THIS_COMMIT: is there a better way to unmarshal this?
	// Using s.cdc.UnmarshalJSON() with *authz.QueryGrantsResponse complains about
	// the "type" field, expecting "@type" for the #Grants[n]#Authorization pb.Any.
	var grantsRes struct {
		Grants []struct {
			Authorization struct {
				Type  string `json:"type"`
				Value struct {
					Msg string `json:"msg"`
				} `json:"value"`
			} `json:"authorization"`
		} `json:"grants"`
	}

	err = json.Unmarshal([]byte(res.Stdout), &grantsRes)
	require.NoError(s, err)

	grantFound := false
	for _, grant := range grantsRes.Grants {
		if grant.Authorization.Value.Msg == msgType {
			grantFound = true
			break
		}
	}
	require.True(s, grantFound)

	s.granterName = granterName
	s.granteeName = granteeName
}

func (s *suite) TheUserSendsAnAuthzExecMessageToUpdateAllModuleParams(moduleName string, table gocuke.DataTable) {
	paramsMap := make(map[string]struct {
		paramType  string
		paramValue any
	})

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

	tempFile, err := os.CreateTemp("", "exec.json")
	require.NoError(s, err)
	s.Cleanup(func() {
		os.Remove(tempFile.Name())
	})

	var (
		msg        proto.Message
		paramsJSON []byte
	)

	switch moduleName {
	case tokenomicstypes.ModuleName:
		msg = &tokenomicstypes.MsgUpdateParams{Params: tokenomicstypes.Params{}}
		for paramName, param := range paramsMap {
			switch paramName {
			case "compute_units_to_tokens_multipler":
				msg.(*tokenomicstypes.MsgUpdateParams).Params.
					ComputeUnitsToTokensMultiplier = uint64(param.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", param.paramType, paramName)
			}
		}
		paramsJSON, err = s.cdc.MarshalJSON(&msg.(*tokenomicstypes.MsgUpdateParams).Params)
		require.NoError(s, err)
	case prooftypes.ModuleName:
		msg = &prooftypes.MsgUpdateParams{Params: prooftypes.Params{}}
		for paramName, param := range paramsMap {
			switch paramName {
			case "min_relay_difficulty_bits":
				msg.(*prooftypes.MsgUpdateParams).Params.
					MinRelayDifficultyBits = uint64(param.paramValue.(int64))
			default:
				s.Fatalf("unexpected %q type param name %q", param.paramType, paramName)
			}
		}
		paramsJSON, err = s.cdc.MarshalJSON(&msg.(*prooftypes.MsgUpdateParams).Params)
		require.NoError(s, err)
	default:
		s.Fatalf("unexpected module name %q", moduleName)
	}

	locals := struct{ Type, Authority, Params string }{
		Type:      proto.MessageName(msg),
		Authority: authtypes.NewModuleAddress(s.granterName).String(),
		Params:    string(paramsJSON),
	}
	buf := new(bytes.Buffer)
	err = updateParamsTxJSONTemplate.Execute(buf, locals)
	require.NoError(s, err)

	// TODO_IMPROVE: reset the module params to their default values using t.Cleanup()
	// here so that the test can be run multiple times per localnet boot
	// (i.e. doesn't require localnet restart).

	// TODO_IN_THIS_COMMIT: find a better and/or more conventional way to programmatically generate tx JSON containing pb.Any messages.
	// - 👍 s.cdc.MarshalJSON() converts the pb.Any message to a JSON object with a "@type" field.
	// - 👎 s.cdc.MarshalJSON() the value of the "@type" field is missing a preceeding "/" that authz exec CLI is expecting.
	// - 👎 EOF error currently when calling authz exec CLI with the generated JSON.
	// - 👎 s.cdc.MarshalJSON() can only be applied to proto.Message implementations (i.e. protobuf types).
	// - ? is there a tx-level object that can be used to serialize & which can be constructed in this scope?
	// - 👎 otherwise, must use a template for the tx JSON structure (s.cdc.MarshalJSON() can't be used on a tx.Tx object).

	replacedJSON := bytes.Replace(buf.Bytes(), []byte(`"@type": "`), []byte(`"@type": "/`), 1)
	_, err = tempFile.Write(replacedJSON)
	require.NoError(s, err)

	s.Log("tempfile:")
	fileOut, err := os.ReadFile(tempFile.Name())
	require.NoError(s, err)

	s.Log(string(fileOut))

	cmd := strings.Split(
		fmt.Sprintf(
			"tx authz exec %s --from %s --keyring-backend test --output json",
			tempFile.Name(), s.granteeName,
		), " ",
	)

	res, err := s.pocketd.RunCommandOnHost("", cmd...)

	require.NoError(s, err)

	s.Log(res.Stdout)
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
	cmd := "keys show %s --keyring-backend test --output json"
	args := strings.Split(fmt.Sprintf(cmd, keyName), " ")
	res, err := s.pocketd.RunCommand(args...)
	require.NoError(s, err)

	keyListRes := make(map[string]any)
	err = json.Unmarshal([]byte(res.Stdout), &keyListRes)
	require.NoError(s, err)

	return keyListRes["address"].(string)
}
