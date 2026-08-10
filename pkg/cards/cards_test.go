package cards_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/cards"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// validServiceCard mirrors the worked `eth` example in docs/pocket_service_card.md.
const validServiceCard = `{
  "schema": "pocket-service-card/v1",
  "description": "Ethereum mainnet execution layer JSON-RPC.",
  "rpc_types": [
    {"type": "JSON_RPC",  "intent": "expected", "backend_hint": "geth/erigon HTTP, default :8545"},
    {"type": "WEBSOCKET", "intent": "expected", "notes": "eth_subscribe"}
  ],
  "apis": ["ethereum-json-rpc"],
  "specs": [
    {"kind": "openrpc", "url": "https://specs.example.org/eth/v1/openrpc.json",
     "sha256": "3b1f0000000000000000000000000000000000000000000000000000000000ff"}
  ],
  "access": "public",
  "results": "deterministic",
  "serving": {
    "backend": "Ethereum execution client with archive state from genesis.",
    "implementations": ["geth >= 1.14"],
    "sync": "archive",
    "min_disk_gb": 3000,
    "healthcheck": [
      {
        "rpc_type": "JSON_RPC",
        "request": {"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []},
        "expect": {"json_path": "$.result", "matches": "^0x[0-9a-f]+$"}
      }
    ]
  },
  "updated": "2026-08-10"
}`

// validGatewayCard mirrors the worked gateway example in docs/pocket_service_card.md.
const validGatewayCard = `{
  "schema": "pocket-gateway-card/v1",
  "description": "Managed Pocket gateway with x402 metered access.",
  "services": ["eth", "poly"],
  "rpc_types": [{"type": "JSON_RPC"}, {"type": "WEBSOCKET"}],
  "endpoints": [{"url": "https://rpc.example.org", "rpc_type": "JSON_RPC"}],
  "access": "gated",
  "payment": [{"protocol": "x402", "network": "base", "payee": "0xabc"}],
  "updated": "2026-08-10"
}`

func TestValidate_EmbeddedSchemasCompile(t *testing.T) {
	// A broken embedded schema must fail loudly here rather than at a user's first publish.
	for _, kind := range []cards.Kind{cards.KindService, cards.KindGateway} {
		raw, err := cards.Schema(kind)
		require.NoError(t, err)
		require.NotEmpty(t, raw)

		// Validating a minimal card exercises unmarshal + compile of the schema itself.
		minimal := fmt.Sprintf(`{"schema":%q}`, cards.SchemaKey[kind])
		require.NoError(t, cards.Validate(kind, []byte(minimal)), "kind %s", kind)
	}
}

func TestValidate_WorkedExamples(t *testing.T) {
	require.NoError(t, cards.Validate(cards.KindService, []byte(validServiceCard)))
	require.NoError(t, cards.Validate(cards.KindGateway, []byte(validGatewayCard)))
}

func TestValidate_Rejections(t *testing.T) {
	tests := []struct {
		desc        string
		kind        cards.Kind
		card        string
		errContains string
	}{
		{
			desc:        "not JSON",
			kind:        cards.KindService,
			card:        `not json at all`,
			errContains: "not valid JSON",
		},
		{
			desc:        "empty",
			kind:        cards.KindService,
			card:        ``,
			errContains: "card is empty",
		},
		{
			desc:        "missing schema key",
			kind:        cards.KindService,
			card:        `{"description":"no schema key"}`,
			errContains: "does not match",
		},
		{
			desc:        "wrong schema value",
			kind:        cards.KindService,
			card:        `{"schema":"pocket-gateway-card/v1"}`,
			errContains: "does not match",
		},
		{
			desc:        "unknown rpc type",
			kind:        cards.KindService,
			card:        `{"schema":"pocket-service-card/v1","rpc_types":[{"type":"HTTP"}]}`,
			errContains: "does not match",
		},
		{
			desc: "`required` used instead of `intent`",
			kind: cards.KindService,
			// Deliberately rejected: `required` implies an enforcement layer that does
			// not exist. See docs/pocket_service_card.md.
			card:        `{"schema":"pocket-service-card/v1","rpc_types":[{"type":"JSON_RPC","required":true}]}`,
			errContains: "does not match",
		},
		{
			desc:        "malformed sha256",
			kind:        cards.KindService,
			card:        `{"schema":"pocket-service-card/v1","specs":[{"url":"https://x/y","sha256":"NOTAHASH"}]}`,
			errContains: "does not match",
		},
		{
			desc:        "gateway payment without protocol",
			kind:        cards.KindGateway,
			card:        `{"schema":"pocket-gateway-card/v1","payment":[{"network":"base"}]}`,
			errContains: "does not match",
		},
		{
			desc:        "unknown card kind",
			kind:        cards.Kind("supplier"),
			card:        `{"schema":"pocket-service-card/v1"}`,
			errContains: "unknown card kind",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := cards.Validate(test.kind, []byte(test.card))
			require.Error(t, err)
			require.ErrorContains(t, err, test.errContains)
		})
	}
}

// TestValidate_ForwardCompatibility asserts the ignore-unknown-keys rule: a v1 reader must
// accept a card carrying keys it has never heard of, which is how v1.x cards stay readable.
func TestValidate_ForwardCompatibility(t *testing.T) {
	card := `{"schema":"pocket-service-card/v1","future_field":{"nested":[1,2,3]},
	          "rpc_types":[{"type":"JSON_RPC","some_future_hint":"x"}]}`
	require.NoError(t, cards.Validate(cards.KindService, []byte(card)))
}

// TestValidate_OpenVocabularies asserts the fields documented as open vocabularies accept
// values outside the documented set, since freezing them is exactly what the card format
// exists to avoid.
func TestValidate_OpenVocabularies(t *testing.T) {
	cardsToCheck := []string{
		`{"schema":"pocket-service-card/v1","rpc_types":[{"type":"JSON_RPC","intent":"nice-to-have"}]}`,
		`{"schema":"pocket-service-card/v1","access":"invite-only"}`,
		`{"schema":"pocket-service-card/v1","results":"probabilistic"}`,
		`{"schema":"pocket-service-card/v1","specs":[{"kind":"grpc-reflection","url":"https://x/y"}]}`,
		`{"schema":"pocket-gateway-card/v1","payment":[{"protocol":"lightning"}]}`,
	}
	for i, card := range cardsToCheck {
		kind := cards.KindService
		if i == len(cardsToCheck)-1 {
			kind = cards.KindGateway
		}
		require.NoError(t, cards.Validate(kind, []byte(card)), "card %d", i)
	}
}

func TestValidate_OversizedRejectedBeforeSchema(t *testing.T) {
	// Oversized AND malformed: the size error must win, since that is the one the chain
	// would also reject on.
	oversized := make([]byte, sharedtypes.MaxServiceMetadataSizeBytes+1)
	err := cards.Validate(cards.KindService, oversized)
	require.ErrorContains(t, err, "exceeds the onchain maximum")
}

func TestSummary(t *testing.T) {
	require.Contains(t, cards.Summary([]byte(validServiceCard)), "pocket-service-card/v1")
	require.Contains(t, cards.Summary([]byte(`{"description":"x"}`)), "(no schema key)")
}

// TestValidate_ReportsEveryProblem asserts the formatter surfaces ALL violations with their
// JSON paths, not just the first, so a publisher fixes a card in one pass.
func TestValidate_ReportsEveryProblem(t *testing.T) {
	card := `{
	  "schema": "pocket-service-card/v1",
	  "rpc_types": [{"type": "HTTP", "required": true}],
	  "specs": [{"url": "https://x/y", "sha256": "NOTAHASH"}]
	}`

	err := cards.Validate(cards.KindService, []byte(card))
	require.Error(t, err)

	for _, want := range []string{
		"/rpc_types/0/type",
		"/rpc_types/0/required",
		"/specs/0/sha256",
	} {
		require.Contains(t, err.Error(), want, "every offending path must be reported")
	}

	// Leaf messages must be actionable, not the library's generic wrappers.
	require.Contains(t, err.Error(), "value must be one of", "enum error must list allowed values")
	require.NotContains(t, err.Error(), "validation failed", "generic wrapper messages must not leak")
}

// TestValidate_RequiredKeyHint asserts the deliberately-forbidden `required` key produces an
// explanation rather than the library's opaque message, since the whole point of rejecting
// it is that the name implies enforcement that does not exist.
func TestValidate_RequiredKeyHint(t *testing.T) {
	card := `{"schema":"pocket-service-card/v1","rpc_types":[{"type":"JSON_RPC","required":true}]}`

	err := cards.Validate(cards.KindService, []byte(card))
	require.Error(t, err)
	require.Contains(t, err.Error(), "Use `intent` instead")
	require.Contains(t, err.Error(), "nothing enforces it")
}
