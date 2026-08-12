package types_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// derivedSharedParamFields are stamped by recordParamsHistory from the current session
// grid, NOT supplied by governance. msg_server_update_param.go overwrites whatever a tx
// carries for them, so a bulk artifact is correct to omit them.
var derivedSharedParamFields = map[string]bool{
	"session_grid_anchor_height": true,
	"session_number_at_anchor":   true,
}

// requiredSharedParamFields derives the set of shared params a bulk artifact MUST carry
// straight from the generated proto struct.
//
// Deliberately reflective rather than a hand-maintained slice: the sibling tokenomics
// guard uses a literal list with a "extend this whenever a param is added" comment, and
// that is precisely how compute_unit_cost_granularity (proto field 11) went missing from
// shared_all.json unnoticed. A list that must be updated by hand cannot guard against
// forgetting to update things by hand.
func requiredSharedParamFields(t *testing.T) []string {
	t.Helper()

	var fields []string
	paramsType := reflect.TypeOf(sharedtypes.Params{})
	for i := 0; i < paramsType.NumField(); i++ {
		protoTag := paramsType.Field(i).Tag.Get("protobuf")
		for _, part := range strings.Split(protoTag, ",") {
			name, found := strings.CutPrefix(part, "name=")
			if !found || derivedSharedParamFields[name] {
				continue
			}
			fields = append(fields, name)
		}
	}

	require.NotEmpty(t, fields, "reflection derived no shared param field names")

	// Canary: if the protobuf tag layout ever changes shape, the loop above would
	// silently derive an empty-ish set and every assertion built on it would pass
	// vacuously. Pin the two fields whose absence actually caused incidents --
	// compute_unit_cost_granularity (the pricing divisor that was missing from
	// shared_all.json) and num_blocks_per_session.
	require.Contains(t, fields, "compute_unit_cost_granularity",
		"reflection failed to derive a known shared param; the guard would pass vacuously")
	require.Contains(t, fields, "num_blocks_per_session",
		"reflection failed to derive a known shared param; the guard would pass vacuously")
	require.NotContains(t, fields, "session_grid_anchor_height",
		"derived anchor fields must be excluded; governance does not supply them")

	return fields
}

// TestSharedBulkParamsJSON_RoundTripsAndValidates guards every governance artifact that
// submits a shared MsgUpdateParams.
//
// MsgUpdateParams REPLACES the whole params struct, so a field missing from one of these
// files is silently written as its proto3 zero value on a live chain. For shared params
// that is not merely cosmetic: compute_unit_cost_granularity is a DIVISOR in claim
// pricing, and the window offsets feed GetNumPendingSessions, which settlement divides
// by in the EndBlocker.
func TestSharedBulkParamsJSON_RoundTripsAndValidates(t *testing.T) {
	paramsDir := filepath.Join("..", "..", "..", "tools", "scripts", "params")

	paramsFiles, err := filepath.Glob(filepath.Join(paramsDir, "bulk_params_*", "shared_params.json"))
	require.NoError(t, err)

	// The single-module bulk template is the artifact an operator reaches for when
	// changing shared params outside a network-specific rollout, so it is held to the
	// same bar as the per-network files.
	templateFile := filepath.Join(paramsDir, "params_templates", "shared_all.json")
	if _, statErr := os.Stat(templateFile); statErr == nil {
		paramsFiles = append(paramsFiles, templateFile)
	}
	require.NotEmpty(t, paramsFiles, "no shared bulk params artifacts found under %s", paramsDir)

	// Same proto-JSON codec the tx decoder uses, so uint64-as-string encodings are
	// exercised exactly as they will be when the tx is submitted.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	requiredFields := requiredSharedParamFields(t)

	for _, paramsFile := range paramsFiles {
		paramsFile := paramsFile
		t.Run(filepath.Base(filepath.Dir(paramsFile))+"/"+filepath.Base(paramsFile), func(t *testing.T) {
			paramsJSON, readErr := os.ReadFile(paramsFile)
			require.NoError(t, readErr)

			var tx struct {
				Body struct {
					Messages []struct {
						Type   string          `json:"@type"`
						Params json.RawMessage `json:"params"`
					} `json:"messages"`
				} `json:"body"`
			}
			require.NoError(t, json.Unmarshal(paramsJSON, &tx), "%s is not valid JSON", paramsFile)
			require.Len(t, tx.Body.Messages, 1, "%s should carry exactly one message", paramsFile)
			require.Equal(t, "/pocket.shared.MsgUpdateParams", tx.Body.Messages[0].Type)

			rawParams := tx.Body.Messages[0].Params

			var presentFields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rawParams, &presentFields))
			for _, field := range requiredFields {
				require.Contains(t, presentFields, field,
					"%s is missing shared param %q; MsgUpdateParams replaces the whole "+
						"struct, so the chain would be written with its zero value",
					paramsFile, field,
				)
			}

			// Decode into the real type and run the same validation the chain runs.
			var params sharedtypes.Params
			require.NoError(t, cdc.UnmarshalJSON(rawParams, &params),
				"%s does not decode into shared Params", paramsFile,
			)
			require.NoError(t, params.ValidateBasic(),
				"%s would be rejected on-chain", paramsFile,
			)
		})
	}
}
