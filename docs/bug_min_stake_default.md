# Bug: Application Auto-Unstake Uses Hardcoded Default Instead of On-Chain Param

**Discovered:** 2026-03-23
**Resolved:** 2026-05-21 (v0.1.34, branch `fix/duplicate-revshare-validation`)
**Severity:** HIGH — zombie applications remain staked below min_stake indefinitely
**Related Issue:** https://github.com/pokt-network/poktroll/issues/1846

## Status: FIXED

- **Fix 1 (settlement check)** — `token_logic_modules.go` now reads
  `k.applicationKeeper.GetParams(ctx).MinStake` instead of `apptypes.DefaultMinStake`.
  Stake only decreases via settlement burn, so this fires in the same block an app
  first drops below the on-chain min_stake — every future crossing is caught.
- **Fix 2 (one-time backfill)** — `MarkBelowMinStakeApplicationsUnbonding` in
  `x/application/keeper/unbond_applications.go`, invoked from the `v0.1.34` upgrade
  handler. Clears the pre-upgrade backlog of idle below-min_stake apps that may never
  be settled again. A *recurring* scan was deliberately rejected: it could only ever
  find pre-upgrade backlog (future crossings are caught by Fix 1) yet would pay the
  per-session scan cost forever.
- **Tests** — `TestProcessTokenLogicModules_AppStakeDropsBelowMinStakeAfterSession`
  (discriminating: min_stake=1,000 POKT, proves the param is read) and
  `TestMarkBelowMinStakeApplicationsUnbonding` (sweep semantics).

### No orphaned-payment risk (verified)

When an app is force-unbonded, its in-flight claims still settle before removal:
- The session hydrator stops assigning the app to new sessions once
  `queryHeight > UnstakeSessionEndHeight` (`Application.IsActive`).
- The unbonding period (`ApplicationUnbondingPeriodSessions × NumBlocksPerSession`
  = 1 session = 60 blocks on mainnet) strictly exceeds the settlement lag (~33 blocks).
- So the last session the app is assigned to settles ~27 blocks before the app is
  removed and its remaining stake returned. Suppliers are paid for proven work.

Invariant to preserve: `unbonding_period_blocks ≥ settlement_lag`. Holds comfortably
at 1 session; only at risk if governance pushes claim+proof windows past 60 blocks.

## Summary

During settlement, the auto-unstake check for applications uses `DefaultMinStake` (1 POKT)
instead of the on-chain governance parameter `min_stake` (currently 1,000 POKT on mainnet).
This means apps that drop below 1,000 POKT but above 1 POKT are never force-unstaked.

## Root Cause

**File:** `x/tokenomics/keeper/token_logic_modules.go:224`

```go
// BUG: Uses hardcoded default (1 POKT) instead of on-chain param (1,000 POKT)
if tlmCtx.Application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {
```

- `apptypes.DefaultMinStake` = 1,000,000 upokt (1 POKT) — hardcoded in `x/application/types/params.go:20`
- On-chain `min_stake` = 1,000,000,000 upokt (1,000 POKT) — set via governance

## Impact

- Applications with stake between 1 POKT and 999 POKT remain staked in a zombie state
- They continue to be assigned to sessions and burn more stake
- They are never force-unstaked because `UnstakeSessionEndHeight` is never set
- The EndBlocker in `x/application/keeper/unbond_applications.go:67-70` only checks
  min_stake for apps that are ALREADY in unbonding state — it doesn't scan active apps

## Evidence

Mainnet app `pokt1640lmzpetrwmgfvnjklsl7pysw9uk2dcgdkhp4`:
- Current stake: 893,659,070 upokt (893 POKT)
- On-chain min_stake: 1,000,000,000 upokt (1,000 POKT)
- `unstake_session_end_height: 0` — NOT unbonding
- Still active, still assigned to sessions

## Fix

### Fix 1: Use on-chain param instead of default

```go
// x/tokenomics/keeper/token_logic_modules.go:224
// BEFORE (buggy):
if tlmCtx.Application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {

// AFTER (correct):
minStake := k.applicationKeeper.GetParams(ctx).MinStake
if tlmCtx.Application.Stake.Amount.LT(minStake.Amount) {
```

### Fix 2 (optional): EndBlocker scan for zombie apps

Add a scan in `x/application/keeper/unbond_applications.go` EndBlocker to catch
any active apps (UnstakeSessionEndHeight == 0) that are below min_stake and
initiate forced unbonding. This would be a safety net for any apps that slipped
through before Fix 1.

## Consensus Safety

Both fixes are consensus-breaking — they change state machine behavior.
Must be deployed via a coordinated upgrade.

## Connection to Issue #1846

Issue #1846 reports that apps get undelegated from gateways when stake hits 0.
The auto-undelegation happens during `UnbondApplication()` when the unbonding
period ends. With this bug, apps between 1-999 POKT never enter unbonding at all,
so they don't get undelegated — but they also don't get cleaned up. The fix here
would cause apps below 1,000 POKT to properly enter unbonding and eventually
get cleaned up (including delegation removal).
