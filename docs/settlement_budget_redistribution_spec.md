# Settlement Budget Redistribution

**Status:** Draft — for review
**Author:** Otto V (Head of Protocol) w/ Claude
**Date:** 2026-07-13
**Type:** Protocol improvement (consensus-breaking)
**Target:** v0.1.36 (next upgrade handler)

---

## 1. Problem

A supplier's payout for a session is capped at an **equal head-split** of the application's
session budget:

```go
// x/tokenomics/keeper/token_logic_modules.go:343-345
maxClaimableAmt := appStake.Amount.
    Quo(math.NewInt(numPendingSessions)).
    Quo(math.NewInt(actualNumSuppliers))
```

Where `actualNumSuppliers` = the number of suppliers that submitted a claim for that
`(application, session)` pair.

`actualNumSuppliers` was introduced to reclaim budget from no-shows ("if only 20 of 50
assigned suppliers claim, each gets appStake/20"). **It never fires.** PATH dispatches
traffic *and health probes* to every session member, so every member serves something,
every member claims, and the divisor is always the full `NumSuppliersPerSession`.

Measured on mainnet, 12 settlement blocks (heights 833213 … 834133, N=20):

| | |
|---|---|
| claims settled | 35,954 |
| claimers per `(app, service, session)` | median **50**, exactly 50 in **80%** of groups |
| claims that hit the cap | **34** (0.09%) |
| total claimed | 73,161 POKT |
| total settled (paid) | 55,373 POKT |
| **delivered but unpaid** | **17,788 POKT = 24.3% of all value claimed** |

Extrapolated: **~37M POKT/yr of relay work delivered and never paid for.** (12-block
sample; bursty — one block had zero capped claims, another 40%. Order of magnitude only.)

### 1.0 Scope — this is one service and one supplier, and Friday's N=20 flip detonated it

The 24.3% figure is real but it is **not** network-wide, and the spec must not imply that.
Comparing five settlement blocks either side of the **N=60 → N=20 flip at h831001** (2026-07-10):

| | PRE-FLIP (N=60) | POST-FLIP (N=20) | change |
|---|---|---|---|
| unpaid as % of claimed | **2.9%** | **31.1%** | **10.7×** |
| unpaid per chain-block | 6.6 POKT | **97.6 POKT** | **14.9×** |
| capped claims | 5 | 14 | |
| **share of loss borne by kalorius.tech** | **100%** | **100%** | |

**Only `poly` (all of it) and `bsc` (63 POKT pre-flip, zero now) have ever shown a loss.**
The other 79 services: exactly zero. In both eras **100% of the loss lands on a single
entity, kalorius.tech.**

Why the flip amplified it ~15×, in order of magnitude:

1. **`numPendingSessions = ceil(sessionEndToProofWindowCloseBlocks / N)` stepped 1 → 2.**
   At N=60 the 33-block settlement lag fits inside one session (`B = appStake`); at N=20 it
   spans two (`B = appStake / 2`). **Every application's per-session budget halved
   overnight, and with it the floor.** This is *correct* behaviour — the stake genuinely must
   cover two concurrent sessions — but it is a discontinuous step that was not modelled as
   part of the N change. Observed floors: 660–1,129 POKT → 235–269 POKT.
2. **Claimers per session rose** (26–46 → 43–50), dividing the smaller `B` into more pieces.
3. **kalorius's share of poly relays grew** 52–72% → 86–93%.
4. **Shorter sessions have higher relative variance.** A burst that averaged out over 60
   blocks now lands entirely inside one 20-block window and blows through the cap.
   (Observed post-flip poly range: 127k–589k estimated relays/block.)

Net effect on kalorius's largest single claim: **1.5–1.7× floor (N=60) → 2.7–7.2× floor (N=20).**

> **Not a claim-inflation bug.** An earlier draft hypothesised that the RelayMiner was
> failing to reset its SMST at the new, 3×-more-frequent session boundaries, inflating
> claims. A longer time-series refutes it: poly traffic was already climbing steeply
> pre-flip (83k → 201k estimated relays/block over ~4h) and there is **no step change at
> h831001**. Claims scale correctly. The cap, not the traffic, is what changed.

**`poly` is the canary, not the exception.** The cap only bites where volume exceeds `B/50`.
The other 79 services are quiet because they have not grown into the wall yet. They will.

### 1.0.1 Live operational risk — a QoS cliff, one config flag away

kalorius currently serves **86–93% of all poly relays** and is paid roughly **16% of what it
claims**. It is losing money every session, and `enable_over_servicing` is **opt-in**.

It is one config change away from switching overservicing off — entirely rationally — at
which point it stops serving past its floor and **poly relays begin getting dropped**. The
network's highest-volume service depends on a supplier the protocol is actively punishing for
being the best, and nothing prevents that supplier from simply stopping.

This raises the priority of this change from "correct the economics" to "close a live QoS
cliff on the busiest service."

### 1.0.2 `actualNumSuppliers` fires where it is not needed, and never where it is

**`actualNumSuppliers` is NOT being removed by this change.** The floor formula is unchanged
(§2). But it is worth being precise about how little it currently achieves, because it is the
sharpest available statement of the problem.

It was introduced to reclaim budget from no-shows: *"if only 20 of 50 assigned suppliers
claim, each gets appStake/20."* Measured over 388 `(app, service, session)` groups across 6
settlements:

| N (claiming suppliers) | groups | |
|---|---|---|
| **50 (full)** | 315 | **81%** |
| 40–49 | 45 | 12% |
| 20–39 | 3 | 1% |
| **< 20** | 25 | 6% |

The groups where it delivers a large benefit — `N = 1` or `2`, a **25–50× larger floor** —
are `base`, `solana`, `arb-one`, `eth`, all claiming **~0 POKT**. Dormant apps with no
traffic.

**Where the cap actually bites — poly, high volume — `N` is always 50**, because PATH probes
every session member and every member therefore claims.

> **The no-show correction was the right idea, but health probes mean there are no no-shows
> on any service that matters.** You cannot fix a head-split by shrinking the head count when
> every head shows up. You have to stop splitting by heads.

### 1.1 What actually happens to that value

Nothing is burned. `token_logic_modules.go:177` passes the **post-cap**
`actualSettlementCoin` into `tlmCtx.SettlementCoin`; `tlm_relay_burn_equals_mint.go:86-94`
burns and debits the application's stake by **that** amount only. The overage never touches
the app's stake and is never minted.

The code already names it: *"deducts the overserviced amount from the claimable amount
(**free work**)"* (`token_logic_modules.go:98`).

So the mechanism is an **unpriced subsidy from suppliers to applications**:

- The application received the relays.
- The application **kept the stake** it would have owed.
- The supplier paid the infra cost and got nothing.

Because the app's stake is never debited for the overage, it doesn't drain — the floor
stays high and the free ride **renews every session indefinitely**. No self-correction.

### 1.2 The head-split is a coordination artifact, not an economic rule

`pkg/relayer/proxy/relay_meter.go:418` meters a supplier against
`stake / numSuppliersPerSession / pendingSessions` — its 1/50 slice, **not** against the
application's budget.

That slice exists for exactly one reason: a RelayMiner cannot observe what the other 49
suppliers have served mid-session, so it cannot know how much of the app's budget remains.
The head-split is a *coordination-free approximation* of "your fair share."

But PATH routes by **QoS**, not by equal split. The approximation is wrong by a large
factor and the entire error is charged to whoever does the work.

**The application's session budget `B` is the only real ceiling. `B/N` is being enforced as
if it were `B`. It isn't.**

### 1.3 Worked example — poly, session 834100

App `pokt1hufj6cdgu83dluput6klhmh54vtrgtl3drttva`. Floor = 235 POKT, N = 50, **B = 11,738 POKT**.

| entity | slots | slot % | relay % | paid | POKT / Mrelay |
|---|---|---|---|---|---|
| **kalorius.tech** | 2 | 4% | **92.8%** | 470 | **454** |
| nodefleet.net | 26 | 52% | 2.6% | 85 | 2,885 |
| easy2stake.com | 14 | 28% | **0.1%** | 3 | 2,885 |
| rpcgate.xyz | 5 | 10% | 2.6% | 83 | 2,885 |
| spacebelt.xyz | 2 | 4% | 1.3% | 41 | 2,885 |

- kalorius claimed **2,982** POKT and was paid **470**. Lost **2,512** in one session.
- Every other entity was paid **6.4× more per relay** for doing essentially nothing.
- Idle slots froze the budget: nodefleet **6,019 POKT**, easy2stake **3,284 POKT**.
- **The app spent 3,212 of its 11,738 budget — 27%.** The money was there the whole time.

### 1.4 This is the sybil engine

Per-address payout is hard-capped at `B/N`. Staking bigger buys nothing (the session
lottery is uniform per operator address, `session_hydrator.go:255`). **The only way to grow
revenue is to own more addresses.**

Mainnet confirms: **183 owners running 4,228 operator addresses at a median of 1.01×
min_stake.** Nobody stakes big, because staking big is strictly dominated. kalorius holds
210 addresses and still landed only 2 slots in that session — it cannot get paid for work
it already did; its only lever is buying more lottery tickets.

**Any session-balancing or services-per-stake proposal fails here.** Rebalance so nodefleet
holds 5 slots instead of 26 — the 21 slots handed to other suppliers still get health
probes, still claim, still count in `N`. The floor stays `B/50`. kalorius is still capped at
235 and still eats 2,512. Session balancing changes *which entity holds the idle budget*,
not *that the budget is idle*.

---

## 2. Mechanism

Demote `B/N` from **hard ceiling** to **guaranteed floor**. The application's session budget
`B` becomes the only true ceiling.

Per `(application, session)` group, with `N` = claiming suppliers:

```
B        = min(appStake / numPendingSessions, perSessionSpendLimit)   // unchanged
floor    = B / N                                                      // unchanged
unused   = Σ max(0, floor − claim_i)          // budget the idle slots did not consume
excess_i = max(0, claim_i − floor)
Σexcess  = Σ excess_i

bonus_i  = (Σexcess > 0) ? unused × excess_i / Σexcess : 0

settled_i = min( claim_i,  floor + bonus_i,  m × floor )
```

`m` = `overservicing_bonus_multiplier` — a new tokenomics param. See §5.

### 2.1 Properties

1. **Guarantee is unchanged.** Serve ≤ `B/N` → paid in full, guaranteed, coordination-free.
   Identical to today.
2. **Above the floor is a gamble, not a promise.** You may claim against budget the
   application committed and nobody else used. Positive expected value; not guaranteed.
   Today it is a *certain loss*. The change is **certain loss → positive-EV bet**.
3. **Bounded by B.** `Σ settled_i ≤ B` always. The app never pays more than it committed.
   *"100k means 100k"* — enforced exactly.
4. **Monotone.** `min(claim, floor + bonus) ≥ min(claim, floor)`. **No supplier is ever
   worse off than today.** Zero losers.
5. **Order-independent.** `bonus_i` is a pure function of `(claim_i, floor, unused, Σexcess)`.
   No sorting required; no map-iteration-order dependence.
6. **`N` cancels out in the regime we are actually in.** `Σclaims ≤ B` ⟺ `Σexcess ≤ unused`
   ⟹ `bonus_i ≥ excess_i` ⟹ **everyone is paid in full — for any `N`.** So in all 12 of 12
   measured sessions, `actualNumSuppliers` has **zero effect on the payout**. It retains two
   real jobs: setting the *guaranteed* floor in the oversubscribed regime (`Σclaims > B`,
   never observed), and serving as the ex-ante signal the RelayMiner meters against
   (`relay_meter.go:418`). Both are free. **Removing it would be a strict regression — it
   stays exactly as-is.** See §1.0.2.

### 2.2 Why not pure pro-rata

`settled_i = claim_i × min(1, B / Σclaims)` is simpler and was considered.

In the regime mainnet actually occupies (`Σclaims ≤ B`, **12 of 12 sessions**) the two are
**bit-for-bit identical** — provable: `Σclaims ≤ B` ⟺ `Σexcess ≤ unused`, so the bonus always
covers every supplier's full excess and everyone is paid in full under both.

They diverge only when `Σclaims > B` (never observed on mainnet). There:

| supplier | today | floor+redistribute | pure pro-rata |
|---|---|---|---|
| heavy | 234 | 5,714 | 5,771 |
| heavy | 234 | 5,714 | 5,771 |
| small | 21 | **21** | **15** |
| probe-only | 20 | **20** | **14** |
| **monotone vs today?** | — | **yes** | **no** |

Pure pro-rata haircuts a supplier that served 20 POKT of probes and had nothing to do with
the oversubscription — **paying it less than today**. It has losers, and they are the small
operators. It also destroys the risk-free line the RelayMiner meter relies on.

**Rejected. Use floor + redistribute.**

### 2.3 Recovery against real data

Replaying the mechanism over the 12 sampled settlement blocks:

| | |
|---|---|
| unpaid today | 17,788 POKT |
| **recovered by redistribution** | **17,788 POKT (100.0%)** |
| still unpaid (demand > B) | **0 POKT (0.0%)** |
| sessions where `Σclaims` fits inside `B` | **12 / 12** |

Mainnet's problem is **100% misallocation, 0% underfunding.** The fix is not partial.

---

## 3. Implementation

### 3.0 Consensus safety of `settlementContext` — read this before touching it

`settlementContext` **is not the dangerous kind of cache**, and this change does not make it
one. Verified:

1. **The `Keeper` struct holds zero caches** (`keeper.go:17-42`): `cdc`, `storeService`,
   `logger`, `authority`, sibling keepers, `sharedQuerier`, `tokenLogicModules`. Nothing
   stateful. This is what CLAUDE.md forbids, and it is absent.
2. **`settlementContext` is a local variable** — `settle_pending_claims.go:44`:
   `settlementContext := NewSettlementContext(ctx, &k, logger)`. Created, used, discarded
   inside a single call.
3. **One call site: the EndBlocker** (`x/tokenomics/module/abci.go:27`). **Unreachable from
   the query path.**

This is precisely the *block-scoped local variable* pattern CLAUDE.md prescribes as the safe
alternative to keeper caches.

**Contrast with the session-hydrator AppHash failure** (`session_hydrator.go:67-75`), which
died of two properties `settlementContext` does not have:

| | session hydrator (broke consensus) | settlementContext |
|---|---|---|
| lifetime | **persisted across blocks** | **one EndBlocker call** |
| populated by | **external RPC queries** | only the EndBlocker |

The hydrator's killer was the *query path* poisoning a cache the *consensus path* then read,
producing divergent gas consumption. That cannot happen here.

**It already carries the value we are changing.** `supplierCountPerAppSession` — the
`actualNumSuppliers` divisor at `token_logic_modules.go:345` — is served from this cache
today. If it were unsafe, mainnet would already be halting.

**This change adds FIELDS to an existing per-block struct. No new cache, no new lifetime, no
new reachability.**

#### Hard determinism rules (enforce in review)

- ✅ Claim iteration order is a KV prefix iterator — identical on every node.
- ✅ All accumulations (`unused`, `totalExcess`, claim sums) are **commutative integer sums**
  — order-independent regardless of iteration order.
- ✅ `bonus_i` is a **pure function** of `(claim_i, floor, unused, totalExcess)`. No
  cross-claim ordering dependence. No sorting required.
- ✅ `math.Int` (big.Int) throughout. **No float64.** No `time.Now()`. No randomness.
- 🚫 **NEVER range over `budgetPerAppSession` to produce output or state.** Lookup only.
  Map iteration order is the one remaining way this could break consensus. If a future change
  needs to iterate it, **sort the keys first** — no exceptions.

### 3.1 `x/tokenomics/keeper/settlement_context.go`

Phase 1 of settlement already collects every claim and calls `IncrementSupplierCount`
(`settle_pending_claims.go:78-82`). Extend that pass to also accumulate claim amounts.

Add to `settlementContext`:

```go
// budgetPerAppSession holds the precomputed budget allocation for each
// (application, session) pair. Populated in Phase 1.5, consumed in Phase 2.
budgetPerAppSession map[appSessionKey]*sessionBudget
```

```go
type sessionBudget struct {
    numSuppliers       int64     // == actualNumSuppliers
    floor              math.Int  // B / N (stake terms); B = min(appStake/numPendingSessions, spendLimit)
    unused             math.Int  // Σ max(0, floor − claim_i)
    totalExcess        math.Int  // Σ max(0, claim_i − floor)
    spendLimitExceeded bool      // whether per_session_spend_limit bound B (for the event)
}
```

Methods (as built):

- `AccumulateClaimBudget(ctx, claim *Claim) error` — Phase 1.5: prices the claim (stake terms),
  lazily inits its group's `floor` via `getOrInitSessionBudget`, folds into `unused`/`totalExcess`.
- `getOrInitSessionBudget(ctx, appAddr, sessionId, sessionEndBlockHeight) (*sessionBudget, error)`
  — memoized per group; computes `floor` incl. the spend-limit clamp (so `B` is folded here, not
  in `ensureClaimAmountLimits`). Also serves as the Phase-2 fail-safe (see §3.3).
- `GetSessionBudget(appAddr, sessionId) (*sessionBudget, bool)` — pure lookup.

`floor` is fixed at group init; `unused`/`totalExcess` are commutative integer sums → order-safe,
no separate finalize step, no key sorting. **`B` is not stored** — `N × floor` is the effective
budget bound, and only `floor` is needed downstream.

### 3.2 `x/tokenomics/keeper/settle_pending_claims.go`

Insert **Phase 1.5** between the collection loop and the settle loop. As built:

```go
// Phase 1.5: precompute each (app, session) group's floor + unused/excess totals.
for i := range collectedClaims {
    claim := &collectedClaims[i]
    if warmErr := settlementContext.ClaimCacheWarmUp(ctx, claim); warmErr != nil {
        continue // faulty claim; Phase 2 discards it deterministically
    }
    if budgetErr := settlementContext.AccumulateClaimBudget(ctx, claim); budgetErr != nil {
        continue // unpriceable; Phase 2 discards it for the same reason
    }
}
```

`AccumulateClaimBudget` prices the claim (`GetClaimeduPOKT` + global inflation → stake terms)
and folds it into its group via `getOrInitSessionBudget`, which lazily computes the floor once
per group and memoizes it. No separate `FinalizeSessionBudgets` step is needed — `unused` /
`totalExcess` complete as claims accumulate, and the floor is fixed at group init.

`ClaimCacheWarmUp` is called inside `settleClaim` (line 868); calling it here first makes Phase
2's call a cache hit. **Verified idempotent** (all three sub-caches are map-guarded and
short-circuit on the second call — §9 Q5, resolved).

> **Claim-set consistency — as built.** `IncrementSupplierCount` stays in **Phase 1** (not
> moved). A claim that fails warmup or pricing in Phase 1.5 is skipped in the budget math *and*
> fails identically in Phase 2 (`settleClaim` bails before `ProcessTokenLogicModules`), so it
> never settles — exactly the pre-change treatment, where such claims were already counted in
> `N` but discarded. A warmup-failed claim inflating `N` slightly shrinks the floor, which is
> conservative (never overpays) and matches legacy behavior. No `collectedClaims` filtering
> required.

> **PERF decision — `claimSettlementCoin` is NOT cached; `GetClaimeduPOKT` is recomputed.**
> An earlier draft mandated caching it on the claim's cache entry so Phase 2 reads it back.
> Dropped: `GetClaimeduPOKT` is pure `big.Rat` arithmetic over already-loaded params/difficulty
> (**zero KV reads**), measured at **1.1 µs/op**. The extra per-claim computation (it now runs
> 3× per claim: Phase 1.5 + EventClaimSettled + TLM) costs **~3 ms per settlement** — below the
> threshold that justifies a cache field + its invalidation surface. See §3.2.1 for the
> measured numbers. Reinstate the cache only if per-settlement claim volume grows ~10×.

### 3.2.1 Performance budget

Measured on mainnet (heights 834020–834140, N=20):

| | |
|---|---|
| settlement block (`h%20==13`, ~2,100 txs, ~3,000 claims) | **60.7s** |
| all other blocks (median) | **60.7s** |
| block *after* settlement (`h%20==14`) | **62.5–63.0s** ← the ~2s state-commit diff |

Block time is bound by CometBFT's ~60s commit timeout, not execution. Settlement's true cost
is ~**2–3s against a 60s budget (~5% utilisation)**. Enormous headroom.

**Store reads — a net REDUCTION, not an increase.** `ClaimCacheWarmUp` is *moved* into Phase
1.5, not added: it is idempotent (map-guarded), so Phase 2's warmup becomes a cache hit —
same reads, same claim set, just resequenced. The one store-touching call in the budget path,
`GetParamsAtHeight`, moved *out of* the per-claim `ensureClaimAmountLimits` and *into*
`getOrInitSessionBudget`, which is memoized per group: **~2,700 per-claim reads → ~63 per-group
reads**, i.e. ~2,600 fewer param-history reads per settlement. Everything else Phase 1.5 does is
in-memory map lookups + pure `big.Rat`/`math.Int` arithmetic (`GetClaimeduPOKT` does zero KV
reads).

**The one thing this change recomputes.** Contrary to an earlier draft, `GetClaimeduPOKT` is
*not* cached — it now runs 3× per claim (was 2×: EventClaimSettled + TLM; Phase 1.5 adds the
3rd). This was a deliberate simplification: the spec's §3.2 per-claim cache was dropped because
the recompute is arithmetic-only and cheap. Cost of that decision, measured below: ~3 ms per
settlement. Reclaimable by caching `claimSettlementCoin` on the claim's cache entry if ever
needed — not worth the invalidation surface at this magnitude.

**Measured added cost** (Go benchmarks, Apple M1; extrapolated to ~2,700 claims / ~63 groups):

| benchmark | ns/op | allocs/op | what it is |
|---|---|---|---|
| `GetClaimeduPOKT` | 1,109 | 37 | the extra per-claim pricing op (pure `big.Rat`, zero KV) |
| `Phase15_AccumulateClaimBudget` | 2,134 | 54 | true marginal per-claim Phase 1.5 cost (steady state) |
| `Phase15_Group50` | 634,885 | 7,501 | full Phase 1.5 for one 50-claim group (incl. *moved* warmup) |

- Added compute ≈ **2.1 µs × 2,700 claims ≈ 5.7 ms**, offset by ~2–3 ms of removed
  `GetParamsAtHeight` reads → **~3 ms net** on a ~2–3 s settlement = **~0.1% overhead**.
- Transient memory: ~2.6 KB/claim ≈ **~7 MB/settlement** of block-scoped `big.Rat` garbage,
  freed at commit (not a leak). The `budgetPerAppSession` map itself is ~63 × ~200 B ≈ **~13 KB**.
- Complexity: single **O(claims)** pass; `getOrInitSessionBudget` memoized (O(groups) real work);
  no sorting, no map ranges, no O(N²).

**Folded-in hot-loop fixes (same PR) leave settlement net FASTER than main.** The two known
settlement hot-loop issues were fixed alongside this change and measured before/after:

| fix | benchmark | before → after |
|---|---|---|
| aggregation struct keys (kill per-op `fmt.Sprintf`) | `Aggregation_MainnetScale` (2,500 claims) | **−25.6% time, −27.6% allocs** (112,113 → 81,194) |
| single `calculateAddressRewards` (kill LRM recompute) | `DistributeValidatorRewards_WithRemainder` | **−10.9% allocs** on the remainder path, scaling ~8 allocs/stakeholder |

These savings exceed the ~3 ms the redistribution pass adds, so the combined change is a **net
performance improvement** on settlement. The aggregation fix preserves the exact prior bank-op /
event ordering (legacy string sort key, built only at sort time); the distribution fix is
deterministic-identical (`calculateAddressRewards` is a pure function computed once instead of
twice).

> **Context:** the N=60→20 flip already tripled settlement *frequency* (claims per settlement
> are unchanged at ~2,700; settlements now run every 20 blocks instead of 60). The chain
> processes **~3× more claims per hour** than before 2026-07-10. Per-block it still fits
> comfortably; this change makes each of those (more frequent) settlements slightly cheaper,
> not more expensive.

### 3.3 `x/tokenomics/keeper/token_logic_modules.go`

Rewrite `ensureClaimAmountLimits`. Replace the two divisions and the duplicated spend-limit
path with a lookup of the precomputed `sessionBudget` (as built):

```go
// sb obtained by ProcessTokenLogicModules via getOrInitSessionBudget (fail-safe, see below)
// and passed in; `sb.floor` and `sb.spendLimitExceeded` are already computed.
sb := sessionBudget

maxClaimableAmt := sb.floor
claimExcess := minRequiredAppStakeAmt.Sub(sb.floor)
if sb.totalExcess.IsPositive() && claimExcess.IsPositive() {
    bonus := sb.unused.Mul(claimExcess).Quo(sb.totalExcess)   // big.Int; no overflow at
                                                              // realistic magnitudes (~2^100)
    maxClaimableAmt = sb.floor.Add(bonus)
}

// overservicing_bonus_multiplier: cap at m * floor. Zero is treated as 1 (legacy cap) so an
// unset/clobbered value is benign and never silently enables redistribution (§5).
effectiveM := tokenomicsParams.OverservicingBonusMultiplier
if effectiveM < 1 {
    effectiveM = 1
}
multiplierCap := sb.floor.Mul(math.NewIntFromUint64(effectiveM))
if maxClaimableAmt.GT(multiplierCap) {
    maxClaimableAmt = multiplierCap
}
```

Everything downstream (`supplierAppStakeToMaxSettlementAmount`, the
`EventApplicationOverserviced` emission using `sb.spendLimitExceeded`, the
`minRequiredAppStakeAmt` clamp) stays.

**Spend-limit is folded into `floor` in `getOrInitSessionBudget`** (Phase 1.5), not in a
parallel branch in `ensureClaimAmountLimits` — `B = min(appStake/numPendingSessions,
perSessionSpendLimit)`, then `floor = B / N`. `spendLimitExceeded` records which term won.

Keep the `divide sessions first, then suppliers` ordering so integer truncation matches.

> **Fail-safe, not fail-closed (as built).** `ProcessTokenLogicModules` obtains `sb` via
> `getOrInitSessionBudget`, not a bare `GetSessionBudget`. On a budget miss (only possible when
> `ProcessTokenLogicModules` runs outside the two-phase flow — e.g. an isolated unit test) it
> lazily computes a **floor-only** budget (`unused = totalExcess = 0` ⇒ no bonus ⇒ the legacy
> cap). This is provably conservative (a supplier is paid at most its floor, so the app can
> never be overpaid) and deterministic. In the real EndBlocker flow Phase 1.5 always populates
> the group first, so the full unused/excess aggregates are used. This decoupling — the floor is
> always recoverable from state; only the *bonus* needs the precomputed aggregates — is why the
> change required **zero edits** to the existing `ProcessTokenLogicModules` unit tests.

### 3.4 Dust

`bonus_i` uses floor division, so `Σ bonus_i ≤ unused`. The remainder is simply never
allocated — not minted, not burned, stays in the app's stake. Deterministic, no sorting,
no dust-assignment rule needed. **Do not "fix" this by distributing the remainder** — that
would require an ordering and buy nothing.

---

## 4. Risks

### 4.1 Application stake exhaustion — NOT a risk (proven safe)

An earlier draft flagged this as a blocker on the assumption that an application could stake
for multiple services, letting several concurrent sessions each draw
`B = appStake / numPendingSessions` from the same `applicationInitialStake`.

**That assumption is false.** `x/shared/types/service_configs.go:15`:

```go
if len(services) != 1 {
    return fmt.Errorf("application must have exactly one service: %v", services)
}
```

Hard-coded, not a governance param. Mainnet confirms: **122 applications, all with exactly
one service, zero exceptions.**

With one service per application the bound closes exactly:

- `numPendingSessions = ceil(sessionEndToProofWindowCloseBlocks / numBlocksPerSession)` = **2** at N=20.
- One service ⇒ an app has exactly one session per session-window ⇒ **at most
  `numPendingSessions` of its sessions are ever pending settlement at once.** That is the
  definition of the divisor.
- Within a settlement block, every claim for app `A` reads the same
  `applicationInitialStake = S` (cached once in `applicationInitialStakeMap`).
- This change guarantees `Σ settled ≤ B` **per session group** (§2.1, property 3).

Therefore, across all of `A`'s claims in a block:

```
Σ settled  ≤  numPendingSessions × B
           =  numPendingSessions × (S / numPendingSessions)
           =  S
```

**The application's stake cannot go negative. No running-stake clamp is needed.** The
`numPendingSessions` divisor *is* the guard, and it is exactly tight. Redistribution operates
strictly inside a bound that already holds.

Note also that sessions settling in *different* blocks are strictly more conservative: the
second session's `applicationInitialStake` has already been debited by the first, so its `B`
shrinks geometrically.

> **One test required, not a blocker:** assert
> `len(candidateSessionEndHeights) ≤ numPendingSessions`. The O2 orphan fix
> (`candidateSessionEndHeightsForLiveParams`, `settle_pending_claims.go:573`) can emit a
> second candidate across a shared-params change. It is deduped and capped at ~2 today, which
> is within the bound — but if a future window-offset change could produce more candidates
> than `numPendingSessions`, the inequality above breaks. Pin it with a test.

### 4.2 App–supplier self-dealing — already guarded, with a large margin

Stake your own app, route relays to your own supplier, farm the pool. Today capped at
`B/N` per session; redistribution raises the ceiling to `B`.

**This is a deeply losing trade, and not by the small margin an earlier draft claimed.**
Live mainnet params:

```
mint_ratio                                  = 0.975
mint_equals_burn_claim_distribution.supplier = 0.79
global_inflation_per_claim                   = 1e-06   (negligible)
```

A colluding app+supplier burns `X` from its own application stake and receives
`0.975 × 0.79 = 0.770 × X` back as the supplier. **A 23% loss per round-trip**, not 2.5%.
(Even capturing the source-owner share too, it's `0.975 × 0.815 = 0.795` — still a 20% loss.)

> **Invariant to surface on param updates:**
> `mint_ratio × mint_equals_burn_claim_distribution.supplier < 1`
> Currently **0.770** — an enormous margin. This, not the head-split cap, is the real
> anti-collusion mechanism. Shipped in v0.1.35 as a **logged warning** on `MsgUpdateParam(s)`
> and in the upgrade handler, so a future governance action cannot *silently* erode the
> margin.
>
> Deliberately NOT a hard validation error: `mint_ratio ≤ 1` and the distribution shares sum
> to 1, so the product is `≤ 1` for every legal param set — self-dealing can be driven to
> break-even (`mint_ratio = 1`, `supplier = 1.0`) but never to a profit. Rejecting that
> single corner would put a DAO-governed policy value on a path (`Params.ValidateBasic`)
> that genesis validation and upgrade handlers also run, where an error halts the chain.
> There is also no economic cliff at the bound: `supplier = 0.9999` is indistinguishable
> from `1.0` and would pass either way.

### 4.2b Fabricated claims — already guarded by the proof threshold

`proof_requirement_threshold = 10,000,000 upokt` = **10 POKT**. Any claim above 10 POKT
**requires a merkle proof** against the SMST with app-signed relays. Fabrication cannot
reach the bonus pool.

Un-proofed fabrication is bounded at 10 POKT/claim → at most ~500 POKT across a 50-supplier
session, against a `B` of ~11,738 POKT. **~4%. Irrelevant.**

> Supersedes an earlier draft's §4.4 concern. The `proof_requirement_int: 0` observed in
> sampled events was a **2.64 POKT `base` claim** — below the threshold. kalorius's poly
> claims are **1,697 POKT**, 170× the threshold, and require proofs.
>
> **Keep `proof_requirement_threshold` low.** It is now load-bearing for this mechanism.
> Raising it materially would open a fabrication path into the bonus pool.

### 4.3 Relay-mining difficulty shock

Suppliers currently stop serving (or eat the loss) at the floor. Once overservicing pays,
served volume on affected services rises. The per-service EMA difficulty (α=0.1, ~22
sessions to 90%) will adjust upward, reducing estimated relays per mined relay.

**Expect a one-time difficulty adjustment on `poly` (and to a lesser extent `bsc`).**
Not a correctness issue. Monitor `EventRelayMiningDifficultyUpdated` post-upgrade.

### 4.4 Fabricated claims — see §4.2b (resolved, not a risk)

*Superseded. An earlier draft flagged `proof_requirement_int: 0` as an open fabrication
vector. It is closed: `proof_requirement_threshold = 10 POKT`, and every claim above it
requires a merkle proof. See §4.2b.*

### 4.5 The cost to the Foundation — THE decision that gates this change

**PNF funds the applications. This change is a direct increase in PNF's relay spend.** This
is the real objection and it must be answered, not buried.

Measured post-flip, per chain-block (12 consecutive settlements):

| | POKT/blk | POKT/yr |
|---|---|---|
| **network settled — what PNF pays today** | 219.75 | **109.1M** |
| network claimed — what the work is actually worth | 323.50 | **160.7M** |
| **unpaid (the supplier subsidy)** | 103.75 | **51.5M — +47%** |
| …of which kalorius | 103.76 | **51.5M — 100% of it** |

Also note: today the cap **silently bounds PNF's exposure**. Removing it means relay spend
grows **linearly with traffic, uncapped**. A capped liability becomes an uncapped one. That
is a budgeting change, not just a cost increase.

#### The honest reframe

The head-split cap is functioning as an **accidental, invisible price control that falls
entirely on whoever works hardest**:

- PNF pays for **109M** of relays.
- PNF **receives 161M** of relays.
- **kalorius funds the 51M difference.**

Only three positions exist:

| | | |
|---|---|---|
| **A** | Pay the 161M. Fix the cap, absorb +47%. | honest, expensive |
| **B** | Pay 109M — but say so with a **price**, applied to everyone. | **budget-neutral** |
| **C** | Keep charging kalorius the difference. | **status quo — NOT STABLE** (§1.0.1: they leave, poly QoS breaks) |

#### Option B — budget-neutral, and it solves the concentration problem for free

Relay price = `CUPR × CUTTM`. If 161M exceeds what PNF will pay, **the price is too high** —
and the honest lever is CUTTM, not a cap. Scale claims by `109.1 / 160.7 = 0.679`:

**`compute_units_to_tokens_multiplier: 128473 → 87270`**

| | today | Option B | change |
|---|---|---|---|
| **kalorius** (93% of poly work) | 62.9 POKT/blk | **113.2** | **+80%** |
| everyone else | 156.8 | 106.5 | **−32%** |
| **PNF TOTAL** | **219.8** | **219.8** | **0%** |

**PNF pays exactly what it pays today.** And the money moves from **holding slots** to
**doing work** — which is the original supplier-concentration problem, solved at zero cost:

> nodefleet: **52% of poly slots, 2.6% of poly relays.**
> kalorius: **4% of slots, 93% of relays.**

The sybil flywheel loses its motor, because owning slots stops paying. No session-balancing
rule, no identity layer, no services-per-stake cap required.

> **Be straight about the cross-subsidy.** The −32% is a real cut for honest *non-poly*
> suppliers too, not only idle slot-holders. It is a genuine repricing and they will notice.
> The defensible answer: 161M is apparently above what PNF is willing to pay, so the price
> *was* too high — and a price cut everyone shares is fairer than a 100% levy on one supplier.

#### The budget control PNF already has and is not using

**`per_session_spend_limit`** — implemented (`token_logic_modules.go:350-375`), per-application,
explicit, hard. **It is `null` on the poly app today.**

This is the correct instrument for bounding PNF's exposure: visible, tunable, per-app. Set it
*above* real usage as a safety ceiling. **Set it too low and you have merely rebuilt the
head-split cap under a new name.**

#### Second-order

`mint = settled × 0.975`, so a larger settled base means **more absolute deflation**. Net
supply effect improves slightly. A point in favour under Option A; neutral under Option B.

---

## 5. Parameter & rollout

> **`m` is a rollout valve, NOT an economic safeguard.** Both risks it was originally
> introduced to guard are already covered with large margins — self-dealing by the reward
> split (§4.2, 23% loss) and fabrication by the proof threshold (§4.2b, 10 POKT). `m` is not
> load-bearing for safety.
>
> **Any `m` below `numSuppliersPerSession` is the same bug at a higher threshold.** A supplier
> whose honest work exceeds `m × floor` returns to doing free work. At `m = 5`, kalorius
> (observed at **7.2× floor**) would *still* be bleeding — the spec would have failed to fix
> the only case that motivated it. Only `m ≥ numSuppliers` (where `B` binds) fully fixes it.
>
> **The real ceiling is `B`, the application's own committed budget.** That is the complete
> answer to "how much can I be paid": *up to what the application committed and nobody else
> used.* `m` exists only to let us ship the code before we ship the economics.

New tokenomics param:

```
overservicing_bonus_multiplier (uint64), default 1
```

`settled_i ≤ m × floor_i`, with **`0` treated as `1`** at settlement.

**Benign zero-value (safety).** The zero value — from a fresh proto3 decode, an un-run
upgrade handler, or a clobbered write — is coerced to `1` (the legacy floor cap) *in the read
path*, so it can **never silently enable redistribution**. This is deliberate: the earlier
"0 = unlimited" design made the dangerous setting the default, the opposite of a safe
zero-value (contrast the anchored-grid genesis fallback that made *its* zero benign).
Consequently the upgrade is economically inert even if the param migration never runs.

**`m = 0` and `m = 1` both reproduce today exactly** (`min(claim, floor + bonus, 1 × floor)`
= `min(claim, floor)`). Clean rollout, no flag-day risk:

1. **Ship the consensus change with `m = 1`** (set explicitly by the upgrade handler so the
   stored value is self-documenting). Zero behaviour change; validates the new settlement path
   and events under real load with no economic effect.
2. **Raise `m` by governance** once the path is proven. Rollback = lower `m`. **No second
   upgrade needed.**

**Target end state: `m ≥ numSuppliersPerSession` (effectively "only `B` binds").** Because
`settled_i ≤ B = numSuppliers × floor`, any `m` at least the per-session supplier count makes
the multiplier cap non-binding and `B` the sole ceiling — the complete answer to "how much can
I be paid." A large value (e.g. `128`) is the "unlimited" setting; do not park `m` at a small
finite value, which is a diluted version of the bug it fixes.

Suggested ramp: **`1` (no-op, prove the code) → large (e.g. `128`, ship the economics).**
Observed max is 7.2× floor and it will move with traffic, so an intermediate step should sit
far above reality — `20×` floor, never 3–5×.

> Given §1.0.1 (live QoS cliff on poly), consider compressing the ramp. The `m = 1` no-op
> stage buys confidence in the *code*; it buys nothing while kalorius is still bleeding.
> Decide against how much upgrade-handler risk you want to carry. This is §9 question 1.

---

## 6. Companion work (not in this change)

- **PATH budget enforcement.** PATH is the only component that sees an app's total session
  consumption. It should (a) stop routing when `B` is spent — making §2's `Σclaims > B`
  branch stay hypothetical — and (b) expose remaining budget to suppliers, turning the
  above-floor gamble into a **priced decision** instead of a blind bet. One piece of
  accounting, two problems. **Treat as a dependency, not a nice-to-have.**
- **RelayMiner meter semantics.** `relay_meter.go:418` must stop treating `B/N` as a hard
  stop. Below floor = risk-free; above floor = speculative. It should meter against the
  app's remaining budget, not a fictional 1/50 slice.
- **Linear stake weighting** in `session_hydrator.go`. Once payout tracks work, splitting
  stops paying. Would collapse 4,228 supplier records toward ~183. Separate change.
- **Session diversity cap.** Demoted to a **liveness** measure only (`solana`: nodefleet is
  89% of candidates, ~45 of 50 slots). Not an economic lever. Low priority.
- **Services-per-stake cap.** Collateral hygiene only; **concentration-neutral by
  construction** (scales every actor's ticket count by the same factor). **Drop from this
  workstream.**

---

## 7. Tests

**Unit — `x/tokenomics/keeper/token_logic_modules_test.go`**

| case | expect |
|---|---|
| `Σclaims < B`, all below floor | all paid in full; no bonus; `unused > 0` |
| `Σclaims < B`, one heavy overservicer (the poly shape) | heavy paid in **full**; others unchanged; `Σsettled = Σclaims` |
| `Σclaims > B` | `Σsettled == B` exactly; below-floor claims paid in full; heavy absorb 100% of shortfall |
| `m = 1` | **byte-identical to pre-change settlement** — golden test vs current impl |
| `totalExcess == 0` | no division by zero; bonus = 0 |
| `N == 1` | floor = B; single supplier can take the whole budget |
| dust | `Σ bonus_i ≤ unused`; remainder unallocated; app stake reflects it |
| spend limit binds | `B = perSessionSpendLimit`; `spendLimitExceeded` event flag correct |
| app stake bound (§4.1) | `len(candidateSessionEndHeights) ≤ numPendingSessions`; app stake never negative even when every pending session settles in one block against the same initial stake |

**Integration — `tests/integration/tokenomics/`**

- Replay the poly 834100 shape: 2 heavy + 48 probe-only, 50 claimers. Assert kalorius-equivalent
  goes 470 → 2,982 and app spend goes 700 → 3,212 with 8,526 left in budget.

**Determinism**

- Same claim set, shuffled iteration order → identical `settled_i` for every claim and
  identical AppHash. This is the one that matters.

**Performance (§3.2.1)**

- Benchmark `SettlePendingClaims` at ~3,000 claims / ~63 (app,session) groups, before vs
  after. Assert no new store reads (count them) and that `GetClaimeduPOKT` is called
  **exactly once per claim**. The regression to watch for is accidental double-computation
  in Phase 2, not the arithmetic itself.

---

## 8. Upgrade wiring

- Consensus-breaking. **Must** land in `app/upgrades/v0.1.36.go`.
- Register `overservicing_bonus_multiplier` default = **1** in the upgrade handler's param
  migration.
- Co-schedule with the other pending consensus-breaking change:
  supplier owner-only unstake-cancel (**PR #1980**).
- Avoid settle-heavy and boundary heights when scheduling: at N=20 settlement lands at
  `h % 20 == 13`; the session boundary is `h % 20 == 1`. Target 09:00–11:00 EDT.

---

## 9. Open questions for review

**1. §4.5 — Option A or Option B? THIS GATES EVERYTHING ELSE.**
   Absorb +47% (109M → 161M POKT/yr), or pair the fix with a CUTTM cut
   (128,473 → 87,270) and stay budget-neutral at 109M while reallocating from slot-holders
   to workers? This is a Foundation treasury decision, not a protocol one. **Everything
   below is downstream of it.**

2. **§4.5** — set `per_session_spend_limit` on the funded apps (currently `null`) as an
   explicit exposure ceiling? Recommended regardless of A/B. Must be set *above* real usage.

3. **§5** — ship at `m = 1` (no-op, prove the code) then open to a large `m` (e.g. `128`,
   where `B` binds), or go straight there? §1.0.1 argues for compressing the ramp.

4. **§4.2** — encode `mint_ratio × supplier_share < 1` as a hard param-validation error?
   **IMPLEMENTED** in `Params.ValidateBasic` (currently 0.770). Flag if you'd rather keep it
   advisory; it gates only the governance `MsgUpdateParams` path, not settlement.

5. ~~Is `ClaimCacheWarmUp` genuinely idempotent?~~ **RESOLVED — yes.** All three sub-caches
   (`cacheApplication`, `cacheSupplier`, `cacheServiceAndDifficulty`) are map-guarded and
   short-circuit on the second call; `IncrementSupplierCount` is separate from warmup, so
   moving warmup earlier does not double-count `N`.

6. Sample size — 12 settlement blocks. Widen to ~100 before quoting the 51.5M POKT/yr figure
   anywhere external.

### Resolved during investigation

- **§4.1 app-stake exhaustion — DEAD.** Applications are limited to exactly one service
  (`service_configs.go:15`, hard-coded; 122/122 on mainnet). The `numPendingSessions` divisor
  is an exactly-tight guard. No clamp needed.
- **§4.2 self-dealing — DEAD.** `mint_ratio × supplier_share = 0.975 × 0.79 = 0.770` → a 23%
  loss per round-trip.
- **§4.2b fabrication — DEAD.** `proof_requirement_threshold = 10 POKT`; kalorius's claims are
  170× that and require proofs.
- **Grace-period double-counting — DEAD (my error).** `mined/blk` runs 198,011 → 197,218
  straight through the flip. No step. Leave `grace_period_end_offset_blocks = 10` alone.
- **Settlement performance — MEASURED, net faster.** §3.2.1: the redistribution pass adds
  ~3 ms/settlement (~0.1%) and *reduces* store reads; the two folded-in hot-loop fixes
  (aggregation struct keys −26% time/−28% allocs; single `calculateAddressRewards` −11% allocs
  on the remainder path) more than pay for it. Net: settlement is faster than main.
- **`m` as a safeguard — UNNECESSARY.** It is a rollout valve only (§5).
