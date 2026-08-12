package types_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestBulkParamsJSON_RoundTripsAndValidates guards the governance bulk-params
// artifacts in tools/scripts/params/bulk_params_*/tokenomics_params.json.
//
// MsgUpdateParams REPLACES the whole params struct, so a field missing from one of
// these files is silently written as its proto3 zero value on a live chain. That has
// already bitten twice:
//   - mint_ratio omitted => 0, which ValidateMintRatio rejects (fails loud, but the
//     file is unusable until fixed).
//   - overservicing_bonus_multiplier omitted => 0, which settlement coerces to 1 —
//     silently reverting settlement budget redistribution to OFF after governance
//     had enabled it (fails SILENT, which is why this test exists).
//
// This test fails whenever a new tokenomics param is added without being backfilled
// into every bulk file.
func TestBulkParamsJSON_RoundTripsAndValidates(t *testing.T) {
	paramsDir := filepath.Join("..", "..", "..", "tools", "scripts", "params")

	networkDirs, err := filepath.Glob(filepath.Join(paramsDir, "bulk_params_*"))
	require.NoError(t, err)
	require.NotEmpty(t, networkDirs, "no bulk_params_* directories found under %s", paramsDir)

	// Same proto-JSON codec the tx decoder uses, so uint64-as-string and double
	// encodings are exercised exactly as they will be when the tx is submitted.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Every field of Params that must be explicitly present in each bulk file.
	// Extend this list whenever a param is added to proto/pocket/tokenomics/params.proto.
	requiredFields := []string{
		"dao_reward_address",
		"mint_allocation_percentages",
		"global_inflation_per_claim",
		"mint_equals_burn_claim_distribution",
		"mint_ratio",
		"overservicing_bonus_multiplier",
	}

	for _, networkDir := range networkDirs {
		networkDir := networkDir
		t.Run(filepath.Base(networkDir), func(t *testing.T) {
			paramsFile := filepath.Join(networkDir, "tokenomics_params.json")
			paramsJSON, readErr := os.ReadFile(paramsFile)
			require.NoError(t, readErr)

			// Pull the params object out of the tx envelope.
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
			require.Equal(t, "/pocket.tokenomics.MsgUpdateParams", tx.Body.Messages[0].Type)

			rawParams := tx.Body.Messages[0].Params

			// Assert every param is explicitly present: an absent field would be
			// written to the chain as its zero value.
			var presentFields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rawParams, &presentFields))
			for _, field := range requiredFields {
				require.Contains(t, presentFields, field,
					"%s is missing tokenomics param %q; MsgUpdateParams replaces the whole "+
						"struct, so the chain would be written with its zero value",
					paramsFile, field,
				)
			}

			// Decode into the real type and run the same validation the chain runs.
			var params tokenomicstypes.Params
			require.NoError(t, cdc.UnmarshalJSON(rawParams, &params),
				"%s does not decode into tokenomics Params", paramsFile,
			)
			require.NoError(t, params.ValidateBasic(),
				"%s would be rejected on-chain", paramsFile,
			)

			// The zero value of the multiplier is benign (coerced to 1 at settlement),
			// but a bulk file must never ship it: it silently disables redistribution.
			require.NotZero(t, params.OverservicingBonusMultiplier,
				"%s must set overservicing_bonus_multiplier explicitly (>= 1)", paramsFile,
			)
		})
	}
}
