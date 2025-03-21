//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"
)

// execTxJSONTemplate is a text template for a tx JSON file which is
// intended to be used with the `authz exec` CLI subcommand: `poktrolld tx authz exec <tx_json_file>`.
var execTxJSONTemplate = template.Must(
	template.New("txJSON").Parse(`{ "body": {{.}} }`),
)

// sendAuthzExecTx sends an authz exec tx using the `authz exec` CLI subcommand:
// `poktrolld tx authz exec <tx_json_file>`.
// It returns before the tx has been committed but after it has been broadcast.
// It ensures that all module params are reset to their default values after the
// test completes.
func (s *suite) sendAuthzExecTx(signingKeyName, txJSONFilePath string) {
	s.Helper()

	argsAndFlags := []string{
		"tx", "authz", "exec",
		txJSONFilePath,
		"--from", signingKeyName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cli.OutputFlag),
		"--yes",
	}
	_, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	// TODO_IMPROVE: wait for the tx to be committed using an events query client
	// instead of sleeping for a specific amount of time.
	//
	// First attempt:
	// eventAttrMatchFn := newEventAttributeMatchFn("action", "/cosmos.authz.v1beta1.MsgExec")
	// s.waitForTxResultEvent(eventAttrMatchFn)
	//
	// This resulted in observing many more events than expected, even accounting
	// for those corresponding to the param reset step, which is automatically
	// registered in a s.Cleanup() below. There are no useful attributes on these
	// events such that we can filter out the noise.

	s.Logf("waiting %d seconds for the authz exec tx to be committed...", txDelaySeconds)
	time.Sleep(txDelaySeconds * time.Second)

	// Reset all module params to their default values after the test completes.
	s.once.Do(func() {
		s.Cleanup(func() { s.resetAllModuleParamsToDefaults() })
	})
}

// newTempUpdateParamsTxJSONFile creates & returns a new temp file with the JSON representation of a tx
// which contains a MsgUpdateParams to update **all module params** for each module & paramsAnyMap
// in the given moduleParamsMap. The returned file is intended for use with the `authz exec` CLI
// subcommand: `poktrolld tx authz exec <tx_json_file>`.
func (s *suite) newTempUpdateParamsTxJSONFile(moduleParams moduleParamsMap) *os.File {
	s.Helper()

	var anyMsgs []*types.Any

	// Collect msgs to update all params (per msg) for each module.
	// E.g., 3 modules with 2 params each will result in 3 MsgUpdateParams messages in one tx.
	for moduleName, paramsMap := range moduleParams {
		// Convert the params map to a MsgUpdateParams message.
		msgUpdateParams := s.paramsMapToMsgUpdateParams(moduleName, paramsMap)

		// Convert the MsgUpdateParams message to a pb.Any message.
		anyMsg, err := types.NewAnyWithValue(msgUpdateParams)
		require.NoError(s, err)

		anyMsgs = append(anyMsgs, anyMsg)
	}

	return s.newTempTxJSONFile(anyMsgs)
}

// newTempUpdateParamTxJSONFile creates & returns a new temp file with the JSON representation of a tx
// which contains a MsgUpdateParam to update params **individually** for each module & paramsAnyMap in the
// given moduleParamsMap. The returned file is intended for use with the `authz exec` CLI subcommand:
// `poktrolld tx authz exec <tx_json_file>`.
func (s *suite) newTempUpdateParamTxJSONFile(moduleParams moduleParamsMap) *os.File {
	s.Helper()

	var anyMsgs []*types.Any

	// Collect msgs to update given params, one param per msg, for each module.
	// E.g., 3 modules with 2 given params each will result in 6 MsgUpdateParam messages in one tx.
	for moduleName, paramsMap := range moduleParams {
		for _, param := range paramsMap {
			// Convert the params map to a MsgUpdateParam message.
			msgUpdateParam := s.newMsgUpdateParam(moduleName, param)

			// Convert the MsgUpdateParams message to a pb.Any message.
			anyMsg, err := types.NewAnyWithValue(msgUpdateParam)
			require.NoError(s, err)

			anyMsgs = append(anyMsgs, anyMsg)
		}
	}

	return s.newTempTxJSONFile(anyMsgs)
}

// newTempTxJSONFile creates & returns a new temp file with the JSON representation
// of a tx which contains the given pb.Any messages. The temp file is removed when
// the test completes.
func (s *suite) newTempTxJSONFile(anyMsgs []*types.Any) *os.File {
	s.Helper()

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
	err = execTxJSONTemplate.Execute(tempFile, string(txBodyJSON))
	require.NoError(s, err)

	return tempFile
}
