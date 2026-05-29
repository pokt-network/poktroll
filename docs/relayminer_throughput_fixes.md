# RelayMiner High-Throughput Fix Plan

## Context

A relayminer operator reported silent relay loss at high throughput (~8192 rps). Their
analysis blamed an uncached `GetServiceComputeUnitsPerRelay` gRPC call per relay as the
"primary cause." Code review disproved that: both `GetService` (CUPR) and
`GetServiceRelayDifficulty` are backed by an in-memory cache with `TTL = math.MaxInt64`
(`pkg/deps/config/suppliers.go:533`) — after the first relay per service there is no gRPC,
just a mutex-guarded map read. The published throughput math (100 rps, "98% loss") is built
on that false premise and should be discarded.

The real, verified problems:

1. **Silent work loss is invisible.** The drop site (`pkg/relayer/proxy/sync.go:524-533`)
   logs a warning but emits **no metric** — operators cannot quantify lost reward.
2. **Single-goroutine mining stage is the throughput ceiling.** `channel.Map`
   (`pkg/observable/channel/map.go:24`) spawns exactly one consumer goroutine. The miner
   stage (`mapMineDehydratedRelay`, `miner.go:102`) does `relay.Marshal()` + SHA256 of the
   full relay bytes per relay on that one goroutine — CPU-bound, one core. When it can't
   drain fast enough, the upstream `servedRewardableRelaysProducer` buffer (1000) fills and
   `sync.go` drops relays via `select/default`.
3. **Buffers are hardcoded, not tunable.** Publish buffer = 1000
   (`observable.go:16`), observer buffer = 50 (`observer.go:18`). No config knob.

Corrected pipeline:
```
HTTP handlers (N goroutines)
  └─► servedRewardableRelaysProducer   [chan buf 1000 — DROPS on full, sync.go:525]
       └─► goPublish → observer chan [buf 50]
            └─► miner Map  [1 goroutine: Marshal + SHA256 + cached difficulty]  ← ROOT BOTTLENECK
                 └─► observer chan [buf 50]
                      └─► session Map [1 goroutine: ensureSessionTree + cached CUPR + smst.Update]
```

Notes that reshape priority vs. the operator's analysis:
- **Stage 2 (session) is NOT the bottleneck.** `smst.Update` writes an in-memory
  `simplemap` + a *buffered* WAL append (10MB / 10s flush, `mined_relays_persistence.go`),
  not a synchronous disk write. It also runs only at the *mined* (difficulty-passing) rate,
  a small fraction of served relays.
- **`sessionsTreesMu` (session.go:317) is uncontended on the hot path** — the single
  session-Map goroutine is the only inserter. Each `SessionTree` has its own `sessionMu`
  (`sessiontree.go:34`), so per-session work is already lock-isolated.
- **`relayMeterMu` gRPC-under-write-lock claim is false** — the 3 gRPC calls run lock-free
  after the RLock is released; hot path is a cheap RLock cache hit.

## Fixes (priority order)

### Fix 1 — Drop observability  (trivial, zero behavior change, do first)

Make lost reward measurable before tuning anything.

- `pkg/relayer/metrics.go`: add counter `RelaysDroppedTotal` (mirror `RelaysTotal`,
  labels `service_id`, `supplier_operator_address`, `reason`) + helper
  `CaptureDroppedRelay(serviceId, supplier, reason)`.
- `pkg/relayer/proxy/sync.go:531` (`default:` case): call the helper with
  `reason="mining_channel_full"`. Downgrade the per-drop `Warn` to
  `ProbabilisticDebugInfo` to avoid log flooding when drops are sustained.
- Audit the websocket bridge send path (`pkg/relayer/proxy/websockets/`, fed via
  `async.go:92`) for a sibling non-blocking send; add the same metric if it drops.

### Fix 2 — Configurable pipeline buffers  (low risk; defaults unchanged)

A safety valve and the cheapest immediate relief — bigger buffers absorb bursts while
Fix 3 raises sustained capacity.

- `pkg/observable/channel/`: add `NewObservable` options `WithPublishBufferSize(n)` and
  `WithSubscribeBufferSize(n)` (store sub-buffer size on `channelObservable`, use it in
  `Subscribe` → `NewObserver`). Keep current constants as the defaults — **no change for
  existing callers.**
- Add a `Map` variant (or option) that threads buffer sizes into the intermediate
  observable it creates, used by the relay pipeline only.
- Config plumbing (mirror `DefaultRequestTimeoutSeconds`, traced
  `types.go:26 → relayminer_configs_reader.go:73 → component`):
  - `YAMLRelayMinerConfig` + `RelayMinerConfig`: add optional
    `served_relays_buffer_size` and `mining_pipeline_buffer_size` (defaults 1000 / 50).
  - Pre-create the `servedRewardableRelaysProducer` channel with the configured size at
    `pkg/relayer/proxy/proxy.go:116` via existing `WithPublisher`.
  - Pass mining-pipeline buffer into `miner.MinedRelays` Map construction.

### Fix 3 — Parallelize the miner (Marshal+hash) stage  (highest value, needs review+bench)

Removes the actual throughput ceiling.

- **Safety argument:** `mapMineDehydratedRelay` is pure and per-relay independent
  (over-servicing / relay-meter checks already happened upstream in `sync.go` before the
  producer send). The downstream SMT is a sparse merkle *sum* trie keyed by relay hash —
  insertion is commutative, so output order does not matter. This is offchain RelayMiner
  state, not consensus state. → safe to parallelize.
- Add `MapParallel(ctx, src, workers, transformFn)` in `pkg/observable/channel/`:
  N worker goroutines each range over `srcObserver.Ch()`, apply `transformFn`, publish to
  one shared dst producer (channel send/recv are concurrent-safe). `sync.WaitGroup`;
  close the dst producer only after all workers drain. Output order **not** preserved.
- Use it **only** for the miner stage in `miner.MinedRelays` (`miner.go:90`). Worker count
  from config (`mining_workers`, default `GOMAXPROCS` or a small fixed N like 4).
- Keep the **session stage single-threaded** — it is not the bottleneck, and parallelizing
  it would turn `sessionsTreesMu` into a real contention point.

## Verification

- **Unit:** new `MapParallel` test — feed K items, assert all K transformed (order-agnostic
  via set compare), producer closes exactly once on src close, no goroutine leak.
- **Unit:** observable buffer-option tests — default unchanged; custom sizes honored.
- **Bench:** `go test -bench` on miner stage, single vs parallel, with representative
  multi-KB relay payloads — confirm marshal+hash scales with workers.
- **Existing suites:** `make test_all` + relay integration/e2e — SMT roots must be identical
  (order-independence guarantee).
- **Load:** drive a relayminer at high rps; watch `RelaysDroppedTotal` go to ~0 and confirm
  claim/proof still settle.
- `make go_lint` before commit.

## Sequencing

Ship **Fix 1 + Fix 2** together (safe, immediate operator value + tunability). Land
**Fix 3** behind careful review + benchmark. None are consensus-breaking — RelayMiner is
offchain — so no coordinated upgrade required.
