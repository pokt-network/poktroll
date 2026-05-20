# Proposal: Apply Validator Commission to Settlement Reward Distribution

## Problem Statement

Validators on the Pocket Network Shannon chain currently receive **no meaningful financial incentive** for running validator infrastructure. The 14% "proposer" allocation from settlement rewards is distributed purely by stake weight, ignoring validator commission rates entirely. This means:

1. **Cosmos inflation is 0%** — no standard block rewards exist
2. **Commission rates are decorative** — validators set commission (5%-50%) but it applies to nothing
3. **Validators earn the same per-token as delegators** — no premium for running infrastructure
4. **Delegators have no reason to prefer low-commission validators** — commission has no effect on returns

On mainnet (block 705573), validators collectively received only **1,607,219 upokt (0.27%)** of the 605,656,913 upokt reward pool — purely from their self-bonded stake, not from any commission mechanism.

## Current Architecture

### Settlement Reward Flow

The settlement reward distribution follows this configured allocation (`MintEqualsBurnClaimDistribution`):

```
{
  "dao":         0.045,   // 4.5%
  "proposer":    0.14,    // 14% → all validators + delegators
  "supplier":    0.79,    // 79%
  "sourceOwner": 0.025,   // 2.5%
  "application": 0.0      // 0% (currently unused)
}
```

**Verified on mainnet block 705573 (2,931 claims, 4,437 POKT settled):**

| Category     | Config | Actual   | Amount (upokt)  |
|-------------|--------|----------|-----------------|
| DAO          | 4.5%   | 4.5001%  | 194,680,344     |
| Proposer     | 14.0%  | 14.0000% | 605,656,913     |
| Supplier     | 79.0%  | 79.0000% | 3,417,642,261   |
| Source Owner | 2.5%   | 2.5000%  | 108,151,787     |

### Code Path: How the 14% "Proposer" Share Is Distributed Today

#### Step 1: Accumulation (per-claim)

In each TLM, the proposer amount is accumulated rather than distributed per-claim:

**`x/tokenomics/token_logic_module/tlm_relay_burn_equals_mint.go:253-263`**
```go
// Accumulate proposer amount for batched validator reward distribution (#1758).
// Instead of calling distributeValidatorRewards per-claim, we add the proposerAmount
// to the shared accumulator. The accumulated total is flushed once after all claims
// are processed, giving the Largest Remainder Method a larger input and eliminating
// per-claim precision loss from floor division on small amounts.
if !proposerAmount.IsZero() {
    opReason := tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION
    accumulateValidatorReward(tlmbem.tlmCtx, opReason, proposerAmount)
}
```

The same pattern exists in `tlm_global_mint.go:220-230`.

#### Step 2: Flush (once per settlement block)

**`x/tokenomics/keeper/settle_pending_claims.go:180`**
```go
batchedResult, flushErr := k.FlushBatchedValidatorRewards(ctx, settlementContext)
```

This calls `FlushBatchedValidatorRewards` (line 650) which invokes `DistributeValidatorRewards`.

#### Step 3: Distribution — pure stake weight, NO commission

**`x/tokenomics/token_logic_module/distribution_validator.go:20-31`**
```go
// DistributeValidatorRewards distributes session settlement rewards to
// all bonded validators and their delegators.
//
// Specifically:
//   - Validator stake weight is used to distribute rewards regardless of who the block proposer is.
//   - Commission is not taken into account since this is independent of consensus rewards.
//   - The validator's self-bonded and delegated stake is taken into account.
//   - Delegators receive rewards proportional to their delegated stake.
//
// For a stakeholder (self-bonded validator or delegator), the distribution formula is:
//
//	stakeholderReward = totalValidatorRewardAmount × (stakeholderStake / totalBondedStake)
```

The function at line 32 gets all bonded validators via `GetBondedValidatorsByPower` (line 85), discovers all stakeholder stakes via `discoverStakeholderStakes` (line 147), then distributes proportionally using the Largest Remainder Method (`calculateProportionalRewards`, line 157).

**Commission is never read from the staking module anywhere in the tokenomics distribution code.**

### Proto Definition

**`proto/pocket/tokenomics/params.proto:67-85`**
```protobuf
message MintEqualsBurnClaimDistribution {
  double dao = 1;
  // TODO_TECHDEBT: Rename "proposer" to "validators" to reflect the work done in #1753.
  double proposer = 2;
  double supplier = 3;
  double source_owner = 4;
  double application = 5;
}
```

Note the existing `TODO_TECHDEBT` at line 71 acknowledging the naming issue.

## Mainnet Validator Set (as of block 705573)

| Validator      | Commission | Total Bonded       | Self-Bonded        | Self-Bond % |
|---------------|-----------|-------------------|-------------------|------------|
| Stakenodes     | 10%       | 12,094,519,829,988 | ~0                 | ~0%        |
| HighStakes     | 5%        | 6,664,443,850,998  | 1,000,000          | 0.00%      |
| Kleomedes α    | 9%        | 6,224,980,238,840  | 1,000,000          | 0.00%      |
| polkachu       | 5%        | 6,110,848,656,169  | 90,000,000         | 0.00%      |
| PNF-12         | 10%       | 5,402,042,824,012  | 2,142,828,332      | 0.04%      |
| Blockval       | 0%        | 5,120,174,128,457  | 996,000,000        | 0.02%      |
| Stake&Relax    | 9%        | 4,999,998,000,000  | 1,000,000          | 0.00%      |
| Validatus      | 8%        | 4,352,003,180,654  | 1,000,000          | 0.00%      |
| eddyzags       | 10%       | 3,699,997,995,662  | 200,000,000,000    | 5.41%      |
| PNF-01         | 50%       | 3,201,000,000,000  | 1,000,000,000      | 0.03%      |
| PNF-04         | 50%       | 3,154,000,000,000  | 4,000,000,000      | 0.13%      |
| CosmosSpaces   | 9%        | 3,103,001,000,000  | 1,000,000          | 0.00%      |
| StakeUp        | 9%        | 3,000,028,000,000  | 30,000,000         | 0.00%      |
| Dungeon        | 5%        | 3,000,007,000,000  | 9,000,000          | 0.00%      |
| PNF-06         | 50%       | 2,600,042,000,000  | 42,000,000         | 0.00%      |
| PNF-07         | 50%       | 2,600,006,000,000  | 5,000,000          | 0.00%      |
| PNF-05         | 50%       | 1,700,049,994,786  | 50,000,000         | 0.00%      |
| PNF-11         | 50%       | 1,252,999,995,407  | 3,000,000,000      | 0.24%      |
| PNF-08         | 50%       | 1,101,938,639,123  | 1,938,639,123      | 0.18%      |
| Kleomedes β    | 50%       | 1,000,001,000,000  | 1,000,000          | 0.00%      |
| **Total**      |           | **80,382,082,334,096** |                |            |

20 validators, 8 PNF-operated (50% commission), 12 community (0-10% commission).

## Model Comparison: Block 705573 Settlement

Total proposer reward pool: **605,656,913 upokt (605.66 POKT)**

### Model A: Current (Pure Stake Weight, No Commission)

```
stakeholderReward = totalReward × (stakeholderStake / totalBondedStake)
```

| Validator      | Commission | Val Gets    | Delegators Get | Total       |
|---------------|-----------|------------|---------------|------------|
| Stakenodes     | 10%       | 0          | 91,128,885     | 91,128,885  |
| HighStakes     | 5%        | 7          | 50,214,746     | 50,214,753  |
| Kleomedes α    | 9%        | 7          | 46,903,509     | 46,903,516  |
| polkachu       | 5%        | 678        | 46,042,888     | 46,043,566  |
| PNF-12         | 10%       | 16,145     | 40,686,763     | 40,702,908  |
| Blockval       | 0%        | 7,504      | 38,571,601     | 38,579,105  |
| Stake&Relax    | 9%        | 7          | 37,673,604     | 37,673,611  |
| Validatus      | 8%        | 7          | 32,791,141     | 32,791,148  |
| eddyzags       | 10%       | 1,506,945  | 26,371,523     | 27,878,468  |
| PNF-01         | 50%       | 7,534      | 24,111,121     | 24,118,655  |
| PNF-04         | 50%       | 30,138     | 23,734,385     | 23,764,523  |
| CosmosSpaces   | 9%        | 7          | 23,380,253     | 23,380,260  |
| StakeUp        | 9%        | 226        | 22,604,161     | 22,604,387  |
| Dungeon        | 5%        | 67         | 22,604,161     | 22,604,228  |
| PNF-06         | 50%       | 316        | 19,590,286     | 19,590,602  |
| PNF-07         | 50%       | 37         | 19,590,294     | 19,590,331  |
| PNF-05         | 50%       | 376        | 12,809,033     | 12,809,409  |
| PNF-11         | 50%       | 22,604     | 9,418,406      | 9,441,010   |
| PNF-08         | 50%       | 14,607     | 8,288,197      | 8,302,804   |
| Kleomedes β    | 50%       | 7          | 7,534,725      | 7,534,732   |
| **Total**      |           | **1,607,219** | **604,049,682** | **605,656,901** |

**Validators receive 0.27% of the pool.** Commission rates have zero effect.

### Model B: With Commission Applied

```
validatorCommission = poolShare × commissionRate
remaining = poolShare - validatorCommission
validatorSelfDel = remaining × (selfBondedStake / validatorTotalStake)
validatorTotal = validatorCommission + validatorSelfDel
delegatorRewards = remaining - validatorSelfDel
```

| Validator      | Commission | Commission$ | Self-Del$ | Val Total   | Del Total   | Val Change    |
|---------------|-----------|------------|----------|------------|------------|--------------|
| Stakenodes     | 10%       | 9,112,888  | 0        | 9,112,888   | 82,015,997  | +9,112,888   |
| HighStakes     | 5%        | 2,510,737  | 7        | 2,510,744   | 47,704,009  | +2,510,737   |
| Kleomedes α    | 9%        | 4,221,316  | 6        | 4,221,322   | 42,682,194  | +4,221,315   |
| polkachu       | 5%        | 2,302,178  | 644      | 2,302,822   | 43,740,744  | +2,302,144   |
| PNF-12         | 10%       | 4,070,290  | 14,531   | 4,084,821   | 36,618,087  | +4,068,676   |
| Blockval       | 0%        | 0          | 7,504    | 7,504       | 38,571,601  | 0            |
| Stake&Relax    | 9%        | 3,390,624  | 6        | 3,390,630   | 34,282,981  | +3,390,623   |
| Validatus      | 8%        | 2,623,291  | 6        | 2,623,297   | 30,167,851  | +2,623,290   |
| eddyzags       | 10%       | 2,787,846  | 1,356,250| 4,144,096   | 23,734,372  | +2,637,151   |
| PNF-01         | 50%       | 12,059,327 | 3,767    | 12,063,094  | 12,055,561  | +12,055,560  |
| PNF-04         | 50%       | 11,882,261 | 15,069   | 11,897,330  | 11,867,193  | +11,867,192  |
| CosmosSpaces   | 9%        | 2,104,223  | 6        | 2,104,229   | 21,276,031  | +2,104,222   |
| StakeUp        | 9%        | 2,034,394  | 205      | 2,034,599   | 20,569,788  | +2,034,373   |
| Dungeon        | 5%        | 1,130,211  | 64       | 1,130,275   | 21,473,953  | +1,130,208   |
| PNF-06         | 50%       | 9,795,301  | 158      | 9,795,459   | 9,795,143   | +9,795,143   |
| PNF-07         | 50%       | 9,795,165  | 18       | 9,795,183   | 9,795,148   | +9,795,146   |
| PNF-05         | 50%       | 6,404,704  | 188      | 6,404,892   | 6,404,517   | +6,404,516   |
| PNF-11         | 50%       | 4,720,505  | 11,302   | 4,731,807   | 4,709,203   | +4,709,203   |
| PNF-08         | 50%       | 4,151,402  | 7,303    | 4,158,705   | 4,144,099   | +4,144,098   |
| Kleomedes β    | 50%       | 3,767,366  | 3        | 3,767,369   | 3,767,363   | +3,767,362   |
| **Total**      |           |            |          | **100,281,066** | **505,375,835** |           |

**Validators receive 16.56% of the pool.** Commission now matters.

### Summary

|                          | Current (Model A) | With Commission (Model B) | Delta          |
|--------------------------|-------------------|---------------------------|----------------|
| Validator income         | 1,607,219 (0.27%) | 100,281,066 (16.56%)      | **+98,673,847** |
| Delegator income         | 604,049,682 (99.73%) | 505,375,835 (83.44%)   | **-98,673,847** |
| Validators earn (multiplier) | 1x           | **62x**                   |                |

## Proposed Change

### What to Modify

The change is isolated to a single function: **`DistributeValidatorRewards`** in `x/tokenomics/token_logic_module/distribution_validator.go`.

Currently the function:
1. Gets all bonded validators and their total bonded tokens
2. Discovers all stakeholders (validators + delegators) and their stakes
3. Distributes rewards proportionally by stake weight using the Largest Remainder Method

The proposed change adds a commission step between steps 2 and 3:

```
For each validator:
  1. Calculate validator's pool share = totalReward × (validatorBondedTokens / totalBondedTokens)
  2. Calculate commission = poolShare × validator.GetCommission()
  3. Send commission directly to validator operator account
  4. Distribute remainder (poolShare - commission) to all of this validator's
     stakeholders (including self-delegation) proportionally by stake
```

### Files to Change

| File | Change |
|------|--------|
| `x/tokenomics/token_logic_module/distribution_validator.go` | Core logic: apply commission before distributing remainder to delegators |
| `proto/pocket/tokenomics/params.proto:71` | Rename `proposer` → `validators` (existing TODO_TECHDEBT) |
| `x/tokenomics/keeper/token_logic_modules_test.go` | Update tests to verify commission-based distribution |
| `x/tokenomics/token_logic_module/validator_distribution_precision_test.go` | Update precision tests |

### Consensus Impact

This is a **consensus-breaking change** — it changes the settlement output (different bank operations for the same input). Requires a coordinated **software upgrade** with an upgrade handler.

### Considerations

1. **Delegator impact**: Delegators to high-commission validators (PNF at 50%) would see reduced returns. Delegators should be given advance notice so they can redelegate if desired.

2. **PNF validators**: 8 of 20 validators are PNF-operated at 50% commission. Under the new model, PNF validators would capture significant commission income. PNF may want to adjust commission rates before this change goes live.

3. **Blockval (0% commission)**: Unaffected — continues to pass all rewards to delegators.

4. **Market dynamics**: Once commission matters, delegators will have a real incentive to compare validators, creating healthy competition. Validators will have a real incentive to attract delegations through reliability and competitive commission rates.

5. **Naming cleanup**: The proto field `proposer` should be renamed to `validators` as part of this change (existing TODO at `params.proto:71`).

## Event Impact

Validator/delegator rewards are emitted as **batched** events (since #1758), not per-claim. The commission change shifts event *values* but introduces **no schema change, no new event types, and no new fields**.

### Emission path

1. TLMs accumulate the proposer bucket per-claim, then `FlushBatchedValidatorRewards` (`x/tokenomics/keeper/settle_pending_claims.go:180`) produces a synthetic result with `ModToAcctTransfer` entries tagged `*_VALIDATOR_REWARD_DISTRIBUTION` / `*_DELEGATOR_REWARD_DISTRIBUTION`.
2. `aggregateModToAcctTransfers` (`:152`) aggregates by `(SenderModule=tokenomics, RecipientAddress, OpReason)`.
3. Each aggregated key → one `SendCoinsFromModuleToAccount` + one `EventSettlementBatch{OpType:"mod_to_acct", Recipient, OpReason, TotalAmount, NumClaims}` (`:459`).

### What changes

| Event | Before | After |
|-------|--------|-------|
| `EventSettlementBatch` VALIDATOR-reason `TotalAmount` | self-bond share only (~0.27% of pool) | + commission carve-out (~16.56% of pool at current mainnet rates) |
| `EventSettlementBatch` DELEGATOR-reason `TotalAmount` | full stake share | minus commission (lower for high-commission validators) |
| Per-claim `EventClaimSettled` | — | **unchanged** (proposer bucket already batched out of `reward_distribution`) |
| Bank `coin_received` / `coin_spent` / `transfer` | — | amounts/recipients shift identically to the batch events |

Notes:

- **Proposer pool total is unchanged** (still 14% on mainnet). Only the intra-pool split between validators and delegators moves. The sum of all reward events is identical.
- **Slightly more `mod_to_acct` events possible**: a validator with ~0 self-bond previously emitted no transfer (zero amounts are skipped); now its non-zero commission emits one VALIDATOR-reason event it did not before. Bounded by validator count (≤ +20/settlement block at mainnet) — negligible.
- A validator's commission + its self-delegation slice **merge into a single** VALIDATOR-reason event (same `SenderModule|Recipient|OpReason` aggregation key). Cross-validator delegators still merge to one event. Keying behavior is unchanged.

### New: `EventValidatorRewardDistribution` (per-validator summary)

To make the commission/delegator split observable **without** per-delegator events (which would multiply with delegator count and undo #1758), the distribution emits one `EventValidatorRewardDistribution` per bonded validator per op_reason per settlement block. It is bounded by the validator-set size (~20 validators × 2 TLM op reasons ≈ 40 events/settlement block) — constant in delegator and claim count.

Fields: `session_end_block_height`, `op_reason`, `validator_operator_address`, `validator_account_address`, `commission_rate`, `pool_share_upokt`, `commission_upokt`, `self_delegation_reward_upokt`, `delegators_reward_upokt`, `total_delegated_stake_upokt`, `num_delegators`.

A delegator's per-validator earnings are reconstructable from this event joined with their own delegation amount (which indexers already track) — no per-delegator events needed:

```
delegatorReward = (self_delegation_reward_upokt + delegators_reward_upokt)
                  × (delegatorStake / total_delegated_stake_upokt)
```

where `(self_delegation_reward_upokt + delegators_reward_upokt)` is the post-commission remainder. Sole-stakeholder validators (no external delegations) report the entire remainder in `self_delegation_reward_upokt` with `delegators_reward_upokt = 0`, `total_delegated_stake_upokt = 0`, `num_delegators = 0`.

Emitted from `DistributeValidatorRewards` (`x/tokenomics/token_logic_module/distribution_validator.go`); `session_end_block_height` is threaded from `FlushBatchedValidatorRewards` (same value used for `EventSettlementBatch`).

### Indexer impact (Pocketdex et al.)

Existing events (`EventSettlementBatch`, bank events) need no schema change — same fields, shifted values: validator-income rows jump ~62×, delegator-income rows for 50%-commission (PNF) validators drop ~17%. Any validator-vs-delegator APR computation reflects the new split immediately.

The new `EventValidatorRewardDistribution` is additive — indexers may optionally consume it to attribute per-validator delegator earnings (the merged `EventSettlementBatch` delegator row alone cannot, since a delegator's rewards across multiple validators collapse into a single transfer). This is a values change plus one additive event, not a migration.

## Data Source

All figures verified from mainnet RPC at `https://sauron-rpc.infra.pocket.network/` on block 705573 (settlement block, height % 60 == 33), queried on 2026-04-08.
