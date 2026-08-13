# Pocket Cards — Service Cards and Gateway Cards (v1)

A **card** is a small, self-describing JSON document stored on-chain in a `pocket.shared.Metadata`
container. There are two kinds, and this document covers both:

| | **Service card** | **Gateway card** |
|---|---|---|
| Describes | An API class (`eth`, `ai-inference`) | A reachable endpoint that fronts services |
| Stored in | `Service.metadata.card` | `Gateway.metadata.card` |
| `schema` value | `"pocket-service-card/v1"` | `"pocket-gateway-card/v1"` |
| Set with | `MsgAddService` / `MsgEditService` | `MsgUpdateGatewayMetadata` |
| CLI | `tx service add-service`, `tx service edit-service` | `tx gateway update-gateway-metadata` |
| Canonical schema | `pkg/cards/service_card.schema.json` | `pkg/cards/gateway_card.schema.json` |

Cards required no protocol change to introduce. The `metadata` field already existed, already
accepted arbitrary bytes, and is already returned by the existing queries.

The canonical schemas live in `pkg/cards/` so they can be compiled into `pocketd` and enforced
client-side. This document is the prose companion, not a second copy.

> Jump to [Commands](#commands) for the full CLI reference for both card kinds.

## Why a document instead of proto fields

- **The schema is not settled.** Proto field numbers are permanent, `reserved` forever, bind
  every indexer and JSON consumer, and each revision costs a governance upgrade. A card
  versions itself in-band and ships v2 whenever the ecosystem is ready.
- **The chain cannot verify most of it.** Provisioning, access and trust claims are owner
  assertions. Typing them adds structure without adding truth.
- **It costs the same.** A typed descriptor for `eth` encodes to ~333 bytes of proto; the same
  content as JSON is ~445. Size is not the deciding factor.

## Rules (both card kinds)

1. The payload MUST be a single UTF-8 JSON object.
2. `schema` is the only REQUIRED key. Everything else is optional.
3. Readers MUST ignore unknown keys — this is how v1 readers survive v1.x cards.
4. Readers MUST NOT treat any field as chain-verified, or as enforced by any layer. The chain
   checks size only; nothing else validates, requires or penalises anything stated here. Every
   value is the owner's intention.
5. Target ≤ 4 KiB; the hard limit is `MaxServiceMetadataSizeBytes` (256 KiB), which applies to
   gateway cards too despite the name.
6. A full method-level API spec does **not** go inline — the Ethereum OpenRPC spec is ~2.4 MiB
   raw, ~920 KiB minified and gzipped. Point at it with `specs[]`.

---

# Service cards

A service card describes a Pocket service. It has two audiences, both first-class:

- **Consumers** — gateways, agents, and clients deciding *what this service is and how to call it*.
- **Suppliers** — node runners deciding *what backend to run and which transports to expose*.

## Fields

| Key | Audience | Notes |
|---|---|---|
| `schema` | both | **Required.** `"pocket-service-card/v1"`. |
| `description` | both | Prose. Free of the `Service.name` character-class limits. |
| `rpc_types` | **both** | The transports, and the owner's (non-enforceable) intent for each. See below. |
| `apis` | consume | Higher-level contract names, e.g. `"ethereum-json-rpc"`. Deliberately unregistered. |
| `specs` | consume | Pointers to full specifications. |
| `access` | consume | `"public"` \| `"gated"`. Open vocabulary. |
| `results` | consume | `"deterministic"` \| `"variable"`. Are two suppliers interchangeable? Open vocabulary. |
| `trust` | consume | Objects, not enum constants — they can name a resolvable registry. |
| `docs` | both | Documentation URL. |
| `serving` | **serve** | Everything a node runner needs. See below. |
| `updated` | both | ISO 8601 date of last card revision. |

### `rpc_types` — the field both audiences read

```json
"rpc_types": [
  {"type": "JSON_RPC",  "intent": "expected", "backend_hint": "geth/erigon HTTP, default :8545"},
  {"type": "WEBSOCKET", "intent": "expected", "backend_hint": "geth/erigon WS, default :8546", "notes": "eth_subscribe"},
  {"type": "GRPC",      "intent": "optional"}
]
```

A consumer reads which transports exist. A node runner reads which ones the service owner expects
them to stand up.

**`intent` is an intention, not a requirement, and it is not enforceable by anything.** The word
is deliberate:

- Nothing checks it at stake time. A supplier serving only `JSON_RPC` against a card that says
  `WEBSOCKET: expected` is a completely valid supplier and will be paid normally.
- Nothing checks it at relay time either. Even a stake-time check would only confirm a supplier
  *declared* an endpoint of that type, which is satisfiable by pointing any URL at any type.
- Consumers may legitimately ignore it. A gateway that doesn't want to carry websocket traffic
  simply doesn't route it, and is not doing anything wrong.

It is the service owner saying "this is what I think serving this service well looks like." That
is genuinely useful to a node runner deciding what to configure, and it is worth nothing as a
guarantee. Naming it `required` would have lent it an authority no layer actually provides.

Open vocabulary: `expected`, `optional`, `deprecated` (a transport being sunset). Unknown values
mean unspecified, never invalid.

`type` uses the on-chain `RPCType` enum names verbatim (`GRPC`, `WEBSOCKET`, `JSON_RPC`, `REST`,
`COMET_BFT`), which makes it directly actionable in two ways:

**It maps 1:1 onto the RelayMiner config.** `GetRPCTypeFromConfig` upper-cases the YAML key and
looks it up in the enum, so a card entry becomes a `rpc_type_service_configs` key by lowercasing:

```yaml
suppliers:
  - service_id: eth
    listen_url: http://0.0.0.0:8545
    rpc_type_service_configs:
      json_rpc:                              # card: {"type": "JSON_RPC",  "intent": "expected"}
        backend_url: http://eth-node:8545
      websocket:                             # card: {"type": "WEBSOCKET", "intent": "expected"}
        backend_url: ws://eth-node:8546
```

**It is observable against chain state.** Every staked supplier publishes
`SupplierServiceConfig.endpoints[].rpc_type` on-chain, so an indexer can diff *intended* (card)
against *served* (supplier configs) and surface coverage in both directions — transports a service
advertises that nobody serves, and gaps between what an owner expects and what the supplier set
provides. That is useful information for consumers picking a service and for owners deciding
whether to recruit suppliers. It is **not** a compliance report, and no supplier is out of line
for appearing in it.

### `serving` — the node runner's section

```json
"serving": {
  "backend": "Ethereum execution client with archive state from genesis.",
  "implementations": ["geth >= 1.14", "erigon >= 2.60", "reth >= 1.0"],
  "sync": "archive",
  "min_disk_gb": 3000,
  "healthcheck": [
    {
      "rpc_type": "JSON_RPC",
      "request": {"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []},
      "expect": {"json_path": "$.result", "matches": "^0x[0-9a-f]+$"}
    }
  ],
  "notes": "Serving from a pruned node will fail archive-depth requests."
}
```

`healthcheck` is the piece worth arguing for: a node runner can verify their backend answers
correctly **before** staking, instead of discovering it from failed relays. It is also the most
likely first runtime consumer of the card — the same probe a QoS layer needs to score suppliers.
Today that per-service knowledge lives in gateway implementations; in the card, the service owner
maintains it instead.

### `results`

Are two suppliers interchangeable for the same request?

- `deterministic` — same query, same bytes, from any supplier. A consumer can retry across
  suppliers, cache responses, and cross-check one against another. `eth` is this.
- `variable` — suppliers return different output for the same input, legitimately. No retry
  equivalence, no cross-checking, responses are not comparable. `ai-inference` is this: each
  supplier brings its own model.

This is a routing and QoS property, not a marketing one. What backend a supplier runs, and
whether the service specifies it or leaves the choice open, is described concretely in
`serving.backend` and `serving.implementations` instead.

### `specs[]`

```json
{"kind": "openrpc", "url": "https://…/eth/v2/openrpc.json", "sha256": "d4c3…"}
```

- `kind` — `"openrpc"` | `"openapi"` | `"asyncapi"` | anything else.
- `url` — any scheme the publisher can serve, including `ipfs://`. Unrelated to the chain's
  `validUrlSchemes`, which governs supplier endpoints.
- `sha256` — **optional, and its presence is a statement.**
  - **Present** = the owner pins this exact document. Consumers MUST verify and MUST reject
    non-matching content. The card stays usable; only the linked spec is discarded.
  - **Absent** = a living document with no integrity claim. Consumers fetch unverified, knowingly.

Publishers SHOULD use immutable, version-addressed URLs (`/v2/openrpc.json`, or a CID) so a new
revision never invalidates the pinned one. Drift then degrades to *lag* rather than breakage.
Overwriting a pinned URL in place is the anti-pattern.

### `trust[]`

```json
{"standard": "ERC-8004", "registry": "eip155:1:0x8004…"}
```

An object rather than an enum constant: a pointer to a live registry is worth more than a label,
and real verification is session- and execution-bound — it belongs in a validation layer, not in
rarely-updated service metadata.

## Example — `eth`

```json
{
  "schema": "pocket-service-card/v1",
  "description": "Ethereum mainnet execution layer JSON-RPC. Archive and full-node methods; eth_subscribe over websocket.",
  "rpc_types": [
    {"type": "JSON_RPC",  "intent": "expected", "backend_hint": "geth/erigon HTTP, default :8545"},
    {"type": "WEBSOCKET", "intent": "expected", "backend_hint": "geth/erigon WS, default :8546", "notes": "eth_subscribe"}
  ],
  "apis": ["ethereum-json-rpc", "eth-subscribe"],
  "specs": [
    {"kind": "openrpc", "url": "https://specs.example.org/eth/v1/openrpc.json", "sha256": "3b1f…"}
  ],
  "access": "public",
  "results": "deterministic",
  "serving": {
    "backend": "Ethereum execution client with archive state from genesis.",
    "implementations": ["geth >= 1.14", "erigon >= 2.60", "reth >= 1.0"],
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
  "docs": "https://docs.example.org/eth",
  "updated": "2026-08-10"
}
```

## Example — `ai-inference`

The only mainnet service with metadata before cards shipped carries a 387-byte descriptor
including `"BYOM: bring your own model/backend"`. As a v1 card:

```json
{
  "schema": "pocket-service-card/v1",
  "description": "Decentralized AI inference gateway. OpenAI-compatible chat completions routed through POKT.",
  "rpc_types": [{"type": "REST", "intent": "expected"}],
  "apis": ["openai-chat-completions"],
  "specs": [{"kind": "openapi", "url": "https://specs.example.org/ai-inference/v1/openapi.json"}],
  "access": "public",
  "results": "variable",
  "serving": {
    "backend": "Any OpenAI-compatible inference server (vLLM, Ollama, TGI).",
    "healthcheck": [
      {
        "rpc_type": "REST",
        "request": {"path": "/v1/models", "method": "GET"},
        "expect": {"json_path": "$.data", "matches": "non-empty array"}
      }
    ]
  },
  "updated": "2026-08-10"
}
```

---

# Gateway cards

A gateway carries the same container (`pocket.shared.Metadata`), the same size cap, and the
same rules — only the field set differs, because a gateway is a reachable endpoint rather
than an API class.

Set it with `MsgUpdateGatewayMetadata`, never with `MsgStakeGateway`. That separation is
deliberate: `MsgStakeGateway` enforces a strictly-positive stake delta on every call, so
folding the card into it would mean escrowing real POKT to fix a typo. Updating a card
costs gas only. The gateway must already be staked; updating is allowed even while unbonding,
since a gateway on its way out may still want to correct or withdraw what it advertises.

## Fields

| Key | Notes |
|---|---|
| `schema` | **Required.** `"pocket-gateway-card/v1"`. |
| `description` | Prose. |
| `services` | Service IDs this gateway advertises. Advertising only — see below. |
| `rpc_types` | Transports this gateway accepts. Same `RPCType` enum names as the service card, but **no `intent`** — a gateway either accepts a transport or it doesn't. |
| `endpoints` | Stable entry points, `{"url": …, "rpc_type": …}`. See below. |
| `access` | `"public"` \| `"gated"`. Open vocabulary. |
| `payment` | How to pay for access, e.g. `{"protocol": "x402", "network": "base", "payee": "0x…"}`. Open object, not a closed enum. |
| `trust` | Same shape as the service card. |
| `docs` | Documentation URL. |
| `updated` | ISO 8601 date of last card revision. |

There is deliberately **no `serving`** block — that section exists for node runners standing up a
backend, and nobody "runs" someone else's gateway. There is also **no `results`**: determinism is
a property of the service, not of the gateway fronting it.

## Example

```json
{
  "schema": "pocket-gateway-card/v1",
  "description": "Managed Pocket gateway with x402 metered access.",
  "services": ["eth", "poly", "ai-inference"],
  "rpc_types": [{"type": "JSON_RPC"}, {"type": "WEBSOCKET"}],
  "endpoints": [{"url": "https://rpc.example.org", "rpc_type": "JSON_RPC"}],
  "access": "gated",
  "payment": [{"protocol": "x402", "network": "base", "payee": "0x…"}],
  "trust": [{"standard": "ERC-8004", "registry": "eip155:1:0x8004…"}],
  "docs": "https://docs.example.org",
  "updated": "2026-08-10"
}
```

## Notes specific to gateways

- **`services` is advertising, not a claim the chain checks.** Entries are not verified to
  exist, and a gateway is not obliged to route everything it lists. The gateway↔service
  relationship is separately derivable from application delegations, so an indexer can
  compare the two.
- **Keep `endpoints` to stable entry points.** Everything about a gateway churns faster than
  a service does — tiers, capacity, regional hosts. On-chain staleness is worse than absence
  because it carries the chain's implied authority, so put durable entry points here and let
  a URL carry the volatile detail. Same principle as `specs[]` on the service side.

---

# Commands

Everything below is `pocketd`. Add `--network=<main|beta|local>`, `--home`, and
`--keyring-backend` as your setup requires.

## Validate a card offline

The chain never parses the card, so this is the only place a malformed one is caught before it
costs gas. Both subcommands take a file path and validate against the compiled-in schema:

```bash
pocketd tx service validate-card ./card.json
pocketd tx gateway validate-card ./gateway-card.json
```

`add-service`, `edit-service` and `update-gateway-metadata` run the same check automatically
before broadcasting. Pass `--skip-card-validation` to publish a payload that is deliberately not
a card.

## Publish a service card

`add-service` both creates and updates — for an existing service it is an update, and only
`add_service_fee` on creation costs more than gas.

```bash
# From a file
pocketd tx service add-service \
    <service-id> "<service-description>" <compute-units-per-relay> \
    --card-file ./card.json \
    --from <owner> --fees 300upokt --network=beta

# Or from base64 (mutually exclusive with --card-file)
pocketd tx service add-service \
    <service-id> "<service-description>" <compute-units-per-relay> \
    --card-base64 "$(base64 -w0 ./card.json)" \
    --from <owner> --fees 300upokt --network=beta
```

Only the service owner can update a service's card.

## Publish service cards in bulk

`edit-service` updates existing services from a YAML config, batched into a single transaction by
default:

```bash
pocketd tx service edit-service --config ./services.yaml --from <owner> --fees 300upokt
```

```yaml
# services.yaml
services:
  - service_id: svc1
    compute_units_per_relay: 15
  - service_id: svc2
    compute_units_per_relay: 25
    card_file: ./cards/svc2.json
```

- An entry that names no `card_file` leaves that service's stored card untouched.
- A service is skipped only when **both** its `compute_units_per_relay` and its card already
  match what is on-chain.
- Card comparison is **byte-exact**, so reformatting a card file counts as a change even when the
  JSON is semantically identical.
- `--disable-batch-msgs` sends one transaction per service instead of one batched transaction.
- The command verifies on-chain that each service exists and that the signer is its owner.

## Publish a gateway card

Signed by the gateway itself — the `--from` address *is* the gateway being updated, so there is no
address argument:

```bash
pocketd tx gateway update-gateway-metadata \
    --card-file ./gateway-card.json \
    --from <gateway> --fees 300upokt --network=beta
```

`--card-base64` works the same way and is mutually exclusive with `--card-file`. Unlike
`add-service`, **one of the two is required**: there is no no-op form of this command.

## Read a card back

Gateways have a dedicated query that decodes the card for you:

```bash
pocketd query gateway card <gateway-address>

# Exact stored bytes, no re-indenting — use this for hashing, diffing,
# or feeding the card straight back into --card-file
pocketd query gateway card <gateway-address> --raw > gateway-card.json
```

Services have **no equivalent decoding command** today. Read the card out of the service query and
decode it yourself:

```bash
pocketd query service show-service <service-id> -o json \
  | jq -r '.service.metadata.card' | base64 -d
```

## Clearing a card

There is currently **no way to clear a card back to nil.**

- Omitting `--card-file` / `--card-base64` **preserves** the stored card. It does not delete it.
  Both `MsgAddService` and `MsgUpdateGatewayMetadata` treat a nil `metadata` as "leave the stored
  card alone", so a client that does not re-send the card cannot silently destroy it.
- The closest available action is publishing a minimal payload such as `{}`, which erases the
  content but leaves `metadata` non-nil — `Metadata.ValidateBasic` rejects a zero-length card.

Consumers must therefore treat a non-nil `Metadata` as **"may be empty"**, not "has content".
A dedicated clear mechanism is tracked as tech debt in the `x/service` and `x/gateway` message
servers.

# What the chain does and does not do

| | |
|---|---|
| Enforces | Payload size only (`MaxServiceMetadataSizeBytes`, 256 KiB decoded). |
| Preserves | Stored metadata when an update omits it (since v0.1.35; previously any unrelated update wiped it). |
| Does not | Parse, schema-check, fetch `specs[].url`, verify `sha256`, or attest to any claim. |
| Cannot | Filter or query on card contents — that belongs to an off-chain index. |

Card storage is raw bytes, so every JSON representation returned by the query APIs carries it
base64-encoded.

# Promotion path

If the schema stabilizes **and** something consumes it at runtime — QoS scoring suppliers with
`serving.healthcheck`, or a router selecting on `rpc_types` — the settled fields become
load-bearing rather than descriptive, and are worth promoting to typed proto fields. That is the
right moment to freeze a taxonomy: after a consumer exists, not before.
