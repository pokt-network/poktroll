//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"
)

// txCodeOK is the tx result code which indicates that a tx was committed successfully.
const txCodeOK = uint32(0)

// cliTxResponse is the subset of the fields of the CLI's tx response JSON which the
// E2E suite asserts on. The broadcast response (`tx authz exec`, i.e. CheckTx) and the
// committed response (`query tx`, i.e. the tx result) share this shape.
type cliTxResponse struct {
	TxHash string `json:"txhash"`
	Code   uint32 `json:"code"`
	RawLog string `json:"raw_log"`
}

// execTxJSONTemplate is a text template for a tx JSON file which is
// intended to be used with the `authz exec` CLI subcommand: `pocketd tx authz exec <tx_json_file>`.
var execTxJSONTemplate = template.Must(
	template.New("txJSON").Parse(`{ "body": {{.}} }`),
)

// sendAuthzExecTx sends an authz exec tx using the `authz exec` CLI subcommand:
// `pocketd tx authz exec <tx_json_file>`, waits for it to be committed, and asserts
// that it succeeded.
// It ensures that all module params are reset to their default values after the
// test completes.
func (s *suite) sendAuthzExecTx(signingKeyName, txJSONFilePath string) {
	s.Helper()

	txRes := s.broadcastAndWaitForAuthzExecTx(signingKeyName, txJSONFilePath)
	require.Equalf(s, txCodeOK, txRes.Code,
		"authz exec tx %s was rejected with code %d: %s", txRes.TxHash, txRes.Code, txRes.RawLog,
	)
}

// sendAuthzExecTxExpectingError sends an authz exec tx, waits for it to be committed,
// and asserts that the chain REJECTED it with a raw log containing expectedRawLogSubstr.
//
// DEV_NOTE: A message which fails validation (e.g. params which fail `ValidateBasic`)
// is rejected at delivery, NOT at broadcast, so the CLI still exits 0. Without an
// explicit assertion on the committed tx result, such a rejection is silent and only
// surfaces later as a confusing "params were not updated" diff.
func (s *suite) sendAuthzExecTxExpectingError(signingKeyName, txJSONFilePath, expectedRawLogSubstr string) {
	s.Helper()

	txRes := s.broadcastAndWaitForAuthzExecTx(signingKeyName, txJSONFilePath)
	require.NotEqualf(s, txCodeOK, txRes.Code,
		"expected authz exec tx %s to be rejected but it succeeded", txRes.TxHash,
	)
	require.Containsf(s, txRes.RawLog, expectedRawLogSubstr,
		"authz exec tx %s was rejected with an unexpected error", txRes.TxHash,
	)
}

// broadcastAndWaitForAuthzExecTx broadcasts an authz exec tx, waits for it to be
// committed, and returns the committed tx result. It ensures that all module params
// are reset to their default values after the test completes.
func (s *suite) broadcastAndWaitForAuthzExecTx(signingKeyName, txJSONFilePath string) cliTxResponse {
	s.Helper()

	argsAndFlags := []string{
		"tx", "authz", "exec",
		txJSONFilePath,
		"--from", signingKeyName,
		keyRingFlag,
		fmt.Sprintf("--%s=json", cli.OutputFlag),
		"--yes",
	}
	res, err := s.pocketd.RunCommandOnHost("", argsAndFlags...)
	require.NoError(s, err)

	broadcastTxRes := s.parseCLITxResponse(res.Stdout)

	// Reset all module params to their default values after the test completes.
	// DEV_NOTE: Registered before the assertions below so that a rejected tx (or a
	// failed assertion on one) still leaves the chain in a clean state for the
	// scenarios which follow.
	s.once.Do(func() {
		s.Cleanup(func() { s.resetAllModuleParamsToDefaults() })
	})

	// A non-zero code here means the tx failed CheckTx and will never be committed,
	// so there is nothing to wait for; the broadcast response is the final result.
	if broadcastTxRes.Code != txCodeOK {
		return broadcastTxRes
	}

	// TODO_IMPROVE: wait for the tx to be committed using an events query client
	// instead of sleeping for a specific amount of time.
	//
	// First attempt:
	// eventAttrMatchFn := newEventAttributeMatchFn("action", "/cosmos.authz.v1beta1.MsgExec")
	// s.waitForTxResultEvent(eventAttrMatchFn)
	//
	// This resulted in observing many more events than expected, even accounting
	// for those corresponding to the param reset step, which is automatically
	// registered in a s.Cleanup() above. There are no useful attributes on these
	// events such that we can filter out the noise.

	s.Logf("waiting %d seconds for the authz exec tx to be committed...", txDelaySeconds)
	time.Sleep(txDelaySeconds * time.Second)

	return s.queryCommittedTx(broadcastTxRes.TxHash)
}

// broadcastTxAndRequireCommitted runs a `pocketd tx ...` command against the node under
// test, blocks until the resulting tx is committed & asserts that it succeeded.
//
// DEV_NOTE: Prefer this over waiting on a tx-result EVENT when a step may run more than
// once in a scenario. The events replay client also serves previously observed events, so
// a second identical tx matches the first one's event & the assertion which follows reads
// stale state. Confirming the returned tx hash cannot alias like that.
func (s *suite) broadcastTxAndRequireCommitted(args ...string) cliTxResponse {
	s.Helper()

	// The response must be parseable, so the output flag is appended here rather than left
	// to each call site: without it the CLI emits YAML & every caller would have to
	// remember to opt in.
	if !slices.ContainsFunc(args, func(arg string) bool {
		return strings.HasPrefix(arg, fmt.Sprintf("--%s", cli.OutputFlag))
	}) {
		args = append(args, fmt.Sprintf("--%s=json", cli.OutputFlag))
	}

	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	broadcastTxRes := s.parseCLITxResponse(res.Stdout)
	require.Equalf(s, txCodeOK, broadcastTxRes.Code,
		"tx failed to broadcast with code %d: %s", broadcastTxRes.Code, broadcastTxRes.RawLog,
	)

	// Give the tx a block to land. queryCommittedTx retries on its own, but its first
	// attempt would otherwise always miss & print a "tx not found" retry banner.
	time.Sleep(txDelaySeconds * time.Second)

	committedTxRes := s.queryCommittedTx(broadcastTxRes.TxHash)
	require.Equalf(s, txCodeOK, committedTxRes.Code,
		"tx %s was rejected with code %d: %s", committedTxRes.TxHash, committedTxRes.Code, committedTxRes.RawLog,
	)

	return committedTxRes
}

// queryCommittedTx queries the tx with the given hash & returns its committed result.
func (s *suite) queryCommittedTx(txHash string) cliTxResponse {
	s.Helper()

	argsAndFlags := []string{
		"query", "tx", txHash,
		fmt.Sprintf("--%s=json", cli.OutputFlag),
	}
	res, err := s.pocketd.RunCommandOnHostWithRetry("", numQueryRetries, argsAndFlags...)
	require.NoErrorf(s, err, "failed to query committed tx %s", txHash)

	return s.parseCLITxResponse(res.Stdout)
}

// parseCLITxResponse extracts the tx response JSON from the given CLI stdout.
// The JSON object is located by its opening brace because the CLI may emit
// non-JSON preamble (e.g. gas estimates) ahead of it.
func (s *suite) parseCLITxResponse(cliStdout string) cliTxResponse {
	s.Helper()

	jsonStartIdx := strings.Index(cliStdout, "{")
	require.GreaterOrEqualf(s, jsonStartIdx, 0, "no tx response JSON found in CLI output: %s", cliStdout)

	var txRes cliTxResponse
	err := json.Unmarshal([]byte(cliStdout[jsonStartIdx:]), &txRes)
	require.NoErrorf(s, err, "failed to unmarshal tx response JSON: %s", cliStdout[jsonStartIdx:])

	return txRes
}

// newTempUpdateParamsTxJSONFile creates & returns a new temp file with the JSON representation of a tx
// which contains a MsgUpdateParams to update **all module params** for each module & paramsAnyMap
// in the given moduleParamsMap. The returned file is intended for use with the `authz exec` CLI
// subcommand: `pocketd tx authz exec <tx_json_file>`.
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
// `pocketd tx authz exec <tx_json_file>`.
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
