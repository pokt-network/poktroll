---
title: Inspecting Tokenomics Data
sidebar_position: 7
---

:::info
This guide provides comprehensive workflows and practical examples for analyzing tokenomics data in the Pocket Network. It includes CLI commands, query patterns, and data structures for different stakeholder types.
:::

## Overview <!-- omit in toc -->

This guide helps network participants understand:

1. **What workflow** they need to evaluate their tokenomics data
2. **What events and parameters** they need to collect
3. **Where to find** the relevant information in the protocol
4. **How to interpret** the results for their specific use case

## Table of Contents <!-- omit in toc -->

- [Stakeholder Workflows](#stakeholder-workflows)
  - [🏗️ Supplier/Operator Workflows](#️-supplieroperator-workflows)
    - [1. Revenue Analysis \& Optimization](#1-revenue-analysis--optimization)
    - [2. Earnings Reconciliation \& Forecasting](#2-earnings-reconciliation--forecasting)
    - [3. Claim \& Proof Management](#3-claim--proof-management)
  - [📱 Application/Consumer Workflows](#-applicationconsumer-workflows)
    - [4. Service Cost Analysis \& Budgeting](#4-service-cost-analysis--budgeting)
    - [5. Service Quality \& Economic Efficiency](#5-service-quality--economic-efficiency)
  - [🌐 Network Analytics Workflows](#-network-analytics-workflows)
    - [6. Protocol Economics Health Check](#6-protocol-economics-health-check)
    - [7. Parameter Impact Analysis](#7-parameter-impact-analysis)
- [Common Data Requirements](#common-data-requirements)
  - [Core Events to Track](#core-events-to-track)
  - [Key Parameters](#key-parameters)
  - [Data Collection Points](#data-collection-points)

## Stakeholder Workflows

### 🏗️ Supplier/Operator Workflows

#### 1. Revenue Analysis & Optimization

**Question**: "How can I maximize my earnings?"

**Required Data:**

- Supplier operator address
- Claims submitted, settled, and expired
- Proof submission rates and success rates
- Settlement amounts by session and service
- Current stake amount and revenue share configuration

**Events to Monitor:**

- `EventClaimCreated` - Track claims submitted
- `EventClaimSettled` - Track successful settlements
- `EventClaimExpired` - Track expired claims
- `EventProofSubmitted` - Track proof submissions
- `EventProofUpdated` - Track proof updates
- `EventSupplierSlashed` - Track slashing events

**Parameters:**

- `mint_equals_burn_claim_distribution` - Supplier allocation percentage
- `global_inflation_per_claim` - Additional inflation rewards
- `compute_units_to_tokens_multiplier` - Conversion rate
- Service-specific `compute_units_per_relay` - Service pricing

**Analysis Workflow:**

1. Calculate expected earnings: `(Relays × Service CUPR × Multiplier × Supplier %)`
2. Compare against actual settlement amounts
3. Identify gaps from expired claims or failed proofs
4. Analyze revenue per staked token efficiency
5. Compare performance against network averages

**Query Examples:**

```bash
# List all claims for a supplier
pocketd q proof list-claims --supplier-operator-address <supplier_address>

# Get specific claim details
pocketd q proof show-claim <session_id> <supplier_operator_address>

# Check relay mining difficulty for service
pocketd q service relay-mining-difficulty <service_id>

# Query tokenomics parameters
pocketd q tokenomics params
```

**Settlement Amount Calculation:**

```go
// Formula: numEstimatedComputeUnits * computeUnitsToTokensMultiplier / computeUnitCostGranularity
claimedAmount = (relays * serviceComputeUnitsPerRelay * difficultyMultiplier) * 
                (computeUnitsToTokensMultiplier / computeUnitCostGranularity)

// Example with defaults:
// - 1000 relays, 1 CUPR, 1.0 difficulty
// - CUTTM: 42,000,000, Granularity: 1,000,000
claimedAmount = (1000 * 1 * 1.0) * (42,000,000 / 1,000,000) = 42,000 uPOKT
```

**Revenue Distribution (MintEqualsBurnClaimDistribution):**

```go
// Default percentages:
supplierShare = claimedAmount * 0.7   // 70%
daoShare = claimedAmount * 0.1        // 10%
proposerShare = claimedAmount * 0.05  // 5%
sourceOwnerShare = claimedAmount * 0.15 // 15%
applicationShare = claimedAmount * 0.0  // 0%
```

---

#### 2. Earnings Reconciliation & Forecasting

**Question**: "Why did I earn X instead of Y this month?"

**Required Data:**

- Historical settlement amounts by time period
- Relay mining difficulty changes
- Parameter changes during the period
- Application overservicing incidents
- Proof submission timing and success rates

**Events to Monitor:**

- `EventRelayMiningDifficultyUpdated` - Difficulty adjustments
- `EventApplicationOverserviced` - Overservicing incidents
- `EventTokenomicsParamsUpdated` - Parameter changes

**Analysis Workflow:**

1. Expected vs Actual calculation breakdown
2. Factor impact analysis (difficulty, inflation, overservicing)
3. Identify optimization opportunities
4. Project future earnings based on current trends

**Historical Data Queries:**

```bash
# Get claims by session end height range
pocketd q proof list-claims --session-end-height <height>

# Get relay mining difficulty history
pocketd q service relay-mining-difficulty-all

# Track parameter changes via events
pocketd q tx --query="message.action='/pocket.tokenomics.MsgUpdateParams'"
```

**EMA Calculation (RelayMiningDifficulty):**

```go
// EMA formula: newEMA = alpha * currentValue + (1 - alpha) * previousEMA
// Alpha = 0.1 (smoothing factor)
newRelaysEMA = 0.1 * numRelaysThisSession + 0.9 * previousRelaysEMA

// Difficulty scaling ratio
difficultyScalingRatio = targetNumRelays / newRelaysEMA
// If ratio > 1.0: difficulty decreases (easier mining)
// If ratio < 1.0: difficulty increases (harder mining)
```

**Revenue Forecasting:**

```go
// Expected monthly revenue
expectedMonthlyRevenue = avgDailyRelays * 30 * serviceComputeUnitsPerRelay * 
                        currentDifficultyMultiplier * computeUnitsToTokensMultiplier / 
                        computeUnitCostGranularity * supplierAllocationPercentage
```

---

#### 3. Claim & Proof Management

**Question**: "What's my claim/proof status and what should I do?"

**Required Data:**

- Active claims by session and status
- Proof submission windows and deadlines
- Historical proof success rates
- Slashing risk exposure

**Parameters:**

- `proof_window_open_offset_blocks` - When proof window opens
- `proof_window_close_offset_blocks` - When proof window closes
- `proof_request_probability` - Probability of proof requirement

**Analysis Workflow:**

1. List active claims awaiting proof windows
2. Calculate upcoming proof deadlines
3. Assess risk of claim expiration
4. Prioritize proof submissions by value and deadline

**Claim Status Tracking:**

```bash
# Get all active claims
pocketd q proof list-claims

# Check proof submission status
pocketd q proof show-proof <session_id> <supplier_operator_address>

# Query proof parameters for window timing
pocketd q proof params
```

**Deadline Calculations:**

```go
// From shared module parameters
proofWindowOpenHeight = sessionEndHeight + gracePeriodsEndOffsetBlocks + 
                       claimWindowCloseOffsetBlocks + proofWindowOpenOffsetBlocks

proofWindowCloseHeight = proofWindowOpenHeight + proofWindowCloseOffsetBlocks

// Current block height vs deadlines
blocksUntilProofDeadline = proofWindowCloseHeight - currentBlockHeight
```

**Claim Lifecycle States:**

```go
// Claim states (from proof module)
CLAIM_PROOF_STATUS_UNKNOWN   // Initial state
CLAIM_PROOF_STATUS_PENDING   // Awaiting proof submission
CLAIM_PROOF_STATUS_ACCEPTED  // Proof submitted and valid
CLAIM_PROOF_STATUS_REJECTED  // Proof invalid or missing
```

---

### 📱 Application/Consumer Workflows

#### 4. Service Cost Analysis & Budgeting

**Question**: "How much will my API usage cost and how can I optimize it?"

**Required Data:**

- Application address and usage patterns
- Historical tokens burned per session/service
- Overservicing incidents and additional costs
- Stake burn rate and refill frequency

**Events to Monitor:**

- `EventApplicationOverserviced` - Unexpected cost events
- `EventClaimSettled` - Actual costs incurred
- Application stake changes

**Parameters:**

- `compute_units_to_tokens_multiplier` - Base cost multiplier
- Service-specific `compute_units_per_relay` - Service pricing
- `mint_equals_burn_claim_distribution` - Cost allocation

**Analysis Workflow:**

1. Calculate historical cost per relay by service
2. Identify overservicing patterns and costs
3. Analyze stake burn rate and efficiency
4. Project future costs based on usage plans
5. Recommend optimal stake amounts

**Cost Calculation Examples:**

```bash
# Monitor application stake burns
pocketd q bank balances <application_address>

# Track overservicing events
pocketd q tx --query="message.action='/pocket.tokenomics.EventApplicationOverserviced'"

# Check service pricing
pocketd q service show-service <service_id>
```

**Cost Per Relay Calculation:**

```go
// Base cost (without difficulty)
baseCostPerRelay = serviceComputeUnitsPerRelay * computeUnitsToTokensMultiplier / 
                   computeUnitCostGranularity

// Actual cost (including difficulty)
actualCostPerRelay = baseCostPerRelay * currentDifficultyMultiplier

// Example: Ethereum mainnet RPC
// - CUPR: 1, CUTTM: 42,000,000, Granularity: 1,000,000
// - Difficulty: 1.5x
actualCost = 1 * 42,000,000 / 1,000,000 * 1.5 = 63 uPOKT per relay
```

**Optimization Strategies:**

1. **Stake Optimization:**
   ```go
   // Calculate optimal stake to avoid overservicing
   optimalStake = estimatedMonthlyUsage * safetyMultiplier
   safetyMultiplier = 1.2 // 20% buffer
   ```

2. **Service Selection:**
   ```go
   // Compare cost efficiency across services
   costEfficiency = serviceQualityScore / actualCostPerRelay
   ```

---

#### 5. Service Quality & Economic Efficiency

**Question**: "Am I getting good value for my spend?"

**Required Data:**

- Cost per successful relay vs market rates
- Service reliability and response times
- Supplier performance metrics
- Comparative cost analysis with other applications

**Analysis Workflow:**

1. Value analysis: cost vs performance correlation
2. Supplier performance comparison
3. Geographic and temporal cost variations
4. Service configuration optimization recommendations

**Quality Metrics:**

```bash
# Track proof submission success rates
pocketd q proof list-proofs --supplier-operator-address <address>

# Monitor supplier slashing events
pocketd q tx --query="message.action='/pocket.tokenomics.EventSupplierSlashed'"

# Check service reliability via claim settlement rates
pocketd q proof list-claims --session-end-height <height>
```

**Benchmarking Methods:**

```go
// Supplier performance metrics
proofSuccessRate = successfulProofs / totalProofsRequired
claimSettlementRate = settledClaims / totalClaimsSubmitted
revenuePerStakedToken = totalRevenue / supplierStake

// Service quality comparison
serviceReliability = settledClaims / totalClaimsForService
avgCostPerRelay = totalCost / totalRelays
qualityScore = serviceReliability * responseTimeScore * accuracyScore
```

**Cost-Quality Analysis:**

```go
// Value calculation
valueScore = (qualityScore * weightQuality + 
             reliabilityScore * weightReliability) / actualCostPerRelay

// Comparative analysis
relativeCost = yourCostPerRelay / networkAverageCostPerRelay
relativeQuality = yourQualityScore / networkAverageQualityScore
```

---

### 🌐 Network Analytics Workflows

#### 6. Protocol Economics Health Check

**Question**: "How healthy are the protocol economics?"

**Required Data:**

- Total supply changes (minting vs burning)
- Distribution across stakeholder types
- Inflation rate trends
- Network activity metrics

**Events to Monitor:**

- All mint/burn events across TLMs
- `EventTokenomicsParamsUpdated` - Parameter changes
- Claim settlement success rates
- Participant activity levels

**Parameters:**

- `global_inflation_per_claim` - Network inflation rate
- `mint_allocation_percentages` - Stakeholder distributions
- `mint_equals_burn_claim_distribution` - Settlement distributions

**Analysis Workflow:**

1. Token flow analysis and supply tracking
2. Stakeholder distribution analysis
3. Network activity correlation with economics
4. Security and sustainability metrics

**Network Health Metrics:**

```bash
# Total supply tracking
pocketd q bank total

# Token distribution analysis
pocketd q distribution community-pool
pocketd q bank balances <dao_address>

# Active participant counts
pocketd q application list-applications
pocketd q supplier list-suppliers
```

**Supply Analysis:**

```go
// Token flow tracking
totalMinted = globalInflationMints + mintEqualsBurnMints
totalBurned = applicationStakeBurns + penaltyBurns
netInflation = totalMinted - totalBurned

// Network activity correlation
activityInflationRatio = totalRelaysSettled / netInflation
supplyGrowthRate = (currentSupply - previousSupply) / previousSupply
```

**Sustainability Indicators:**

```go
// Economic sustainability
stakeUtilizationRate = totalStakedTokens / totalSupply
rewardDistributionHealth = supplierRewards / (supplierRewards + daoRewards + proposerRewards)

// Network security
validatorStakeRatio = totalValidatorStake / totalSupply
slashingRate = totalSlashingEvents / totalClaims
```

**Health Score Calculation:**

```go
// Composite health score (0-100)
healthScore = (activityScore * 0.3) + (sustainabilityScore * 0.4) + 
              (securityScore * 0.3)

// Where each component is normalized to 0-100 range
activityScore = min(100, (actualActivity / targetActivity) * 100)
sustainabilityScore = min(100, (1 - abs(netInflation / targetInflation)) * 100)
securityScore = min(100, (1 - slashingRate) * 100)
```

---

#### 7. Parameter Impact Analysis

**Question**: "What would happen if we change parameter X?"

**Required Data:**

- Historical parameter values and change impacts
- Stakeholder behavior responses to changes
- Network activity correlation with parameter values

**Analysis Workflow:**

1. Impact modeling across stakeholder types
2. Scenario analysis (best/worst/expected cases)
3. Security and sustainability implications
4. Implementation timing recommendations

**Parameter Impact Modeling:**

```bash
# Current parameter values
pocketd q tokenomics params
pocketd q shared params
pocketd q service params

# Parameter change history
pocketd q tx --query="message.action='/pocket.tokenomics.MsgUpdateParams'"
```

**Sensitivity Analysis Examples:**

```go
// Global inflation impact
old_inflation := 0.1  // 10%
new_inflation := 0.05 // 5%
revenue_impact := (new_inflation - old_inflation) / old_inflation * 100 // -50%

// CUTTM (pricing) impact
old_cuttm := 42000000
new_cuttm := 21000000  // 50% reduction
cost_impact := (new_cuttm - old_cuttm) / old_cuttm * 100 // -50%

// Distribution percentage impact
old_supplier_pct := 0.7
new_supplier_pct := 0.8
supplier_revenue_impact := (new_supplier_pct - old_supplier_pct) / old_supplier_pct * 100 // +14.3%
```

**Scenario Modeling:**

```go
// Best case scenario
bestCaseRevenue := avgRelays * 1.5 * maxDifficultyMultiplier * 
                   maxSupplierAllocation * currentPricing

// Worst case scenario
worstCaseRevenue := avgRelays * 0.5 * minDifficultyMultiplier * 
                    minSupplierAllocation * currentPricing

// Expected case
expectedRevenue := avgRelays * currentDifficultyMultiplier * 
                   currentSupplierAllocation * currentPricing
```

**Implementation Timing:**

```go
// Parameter change lead time
governanceProposalPeriod := 7 * 24 * 60 * 60 // 7 days in seconds
implementationDelay := 3 * 24 * 60 * 60      // 3 days buffer
totalChangeTime := governanceProposalPeriod + implementationDelay

// Optimal timing windows
optimalChangeBlock := currentBlock + (totalChangeTime / avgBlockTime)
offSeasonTiming := isLowActivityPeriod(optimalChangeBlock)
```

---

## Common Data Requirements

### Core Events to Track

#### **Claim and Settlement Events**

```go
// EventClaimCreated - When supplier submits claim
type EventClaimCreated struct {
    Claim                    prooftypes.Claim
    NumRelays               uint64
    NumClaimedComputeUnits  uint64
    NumEstimatedComputeUnits uint64
    ClaimedUpokt            cosmos.base.v1beta1.Coin
}

// EventClaimSettled - When claim is successfully settled
type EventClaimSettled struct {
    Claim                    prooftypes.Claim
    ProofRequirement        ProofRequirementReason
    NumRelays               uint64
    NumClaimedComputeUnits  uint64
    NumEstimatedComputeUnits uint64
    ClaimedUpokt            cosmos.base.v1beta1.Coin
}

// EventClaimExpired - When claim expires without settlement
type EventClaimExpired struct {
    Claim                    prooftypes.Claim
    ExpirationReason        ClaimExpirationReason
    NumRelays               uint64
    NumClaimedComputeUnits  uint64
    NumEstimatedComputeUnits uint64
    ClaimedUpokt            cosmos.base.v1beta1.Coin
}
```

#### **Proof Events**

```go
// EventProofSubmitted - When supplier submits proof
type EventProofSubmitted struct {
    Claim                    prooftypes.Claim
    NumRelays               uint64
    NumClaimedComputeUnits  uint64
    NumEstimatedComputeUnits uint64
    ClaimedUpokt            cosmos.base.v1beta1.Coin
}

// EventProofValidityChecked - Proof validation result
type EventProofValidityChecked struct {
    Claim          prooftypes.Claim
    BlockHeight    uint64
    FailureReason  string
}
```

#### **Tokenomics Events**

```go
// EventApplicationOverserviced - When app exceeds available stake
type EventApplicationOverserviced struct {
    ApplicationAddr       string
    SupplierOperatorAddr string
    ExpectedBurn         cosmos.base.v1beta1.Coin
    EffectiveBurn        cosmos.base.v1beta1.Coin
}

// EventSupplierSlashed - When supplier is penalized
type EventSupplierSlashed struct {
    Claim                 prooftypes.Claim
    ProofMissingPenalty  cosmos.base.v1beta1.Coin
}

// EventApplicationReimbursementRequest - Global inflation reimbursement
type EventApplicationReimbursementRequest struct {
    ApplicationAddr      string
    SupplierOperatorAddr string
    SupplierOwnerAddr    string
    ServiceId           string
    SessionId           string
    Amount              cosmos.base.v1beta1.Coin
}
```

#### **Service Events**

```go
// EventRelayMiningDifficultyUpdated - Difficulty adjustment
type EventRelayMiningDifficultyUpdated struct {
    ServiceId              string
    PrevTargetHashHex     string
    NewTargetHashHex      string
    PrevNumRelaysEma      uint64
    NewNumRelaysEma       uint64
}
```

### Key Parameters

#### **Tokenomics Module Parameters**

```go
type Params struct {
    // DAO reward address for token distribution
    DaoRewardAddress string // e.g., "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
    
    // Global inflation percentage per claim settlement
    GlobalInflationPerClaim float64 // e.g., 0.1 (10%)
    
    // Distribution percentages for global inflation
    MintAllocationPercentages MintAllocationPercentages {
        Dao:         0.1,   // 10% to DAO
        Proposer:    0.05,  // 5% to block proposer
        Supplier:    0.7,   // 70% to supplier
        SourceOwner: 0.15,  // 15% to service owner
        Application: 0.0,   // 0% to application
    }
    
    // Distribution percentages for mint=burn TLM
    MintEqualsBurnClaimDistribution MintEqualsBurnClaimDistribution {
        Dao:         0.1,   // 10% to DAO
        Proposer:    0.05,  // 5% to block proposer
        Supplier:    0.7,   // 70% to supplier
        SourceOwner: 0.15,  // 15% to service owner
        Application: 0.0,   // 0% to application
    }
}
```

#### **Shared Module Parameters**

```go
type SharedParams struct {
    // Token conversion parameters
    ComputeUnitsToTokensMultiplier uint64 // Default: 42,000,000 (42M pPOKT/CU)
    ComputeUnitCostGranularity    uint64 // Default: 1,000,000 (1M pPOKT = 1 uPOKT)
    
    // Session timing parameters
    NumBlocksPerSession           uint64 // Default: 4 blocks
    GracePeriodEndOffsetBlocks   uint64 // Default: 1 block
    
    // Claim and proof window parameters
    ClaimWindowOpenOffsetBlocks  uint64 // Default: 1 block
    ClaimWindowCloseOffsetBlocks uint64 // Default: 4 blocks
    ProofWindowOpenOffsetBlocks  uint64 // Default: 0 blocks
    ProofWindowCloseOffsetBlocks uint64 // Default: 4 blocks
    
    // Unbonding periods
    ApplicationUnbondingPeriodSessions uint64 // Default: 1 session
    SupplierUnbondingPeriodSessions   uint64 // Default: 1 session
    GatewayUnbondingPeriodSessions    uint64 // Default: 1 session
}
```

#### **Service-Specific Parameters**

```go
type Service struct {
    Id                   string // e.g., "ethereum"
    Name                string // e.g., "Ethereum Mainnet"
    ComputeUnitsPerRelay uint64 // e.g., 1 (1 CU per relay)
    OwnerAddress        string // Service owner address
}
```

#### **Proof Module Parameters**

```go
type ProofParams struct {
    // Proof requirement parameters
    ProofRequestProbability   float64 // e.g., 0.25 (25% probability)
    ProofRequirementThreshold cosmos.base.v1beta1.Coin // e.g., 1000000 uPOKT
    
    // Penalty and fee parameters
    ProofMissingPenalty cosmos.base.v1beta1.Coin // e.g., 320000000 uPOKT
    ProofSubmissionFee  cosmos.base.v1beta1.Coin // e.g., 1000000 uPOKT
}
```

#### **Service Module Parameters**

```go
type ServiceParams struct {
    // Service management parameters
    AddServiceFee   cosmos.base.v1beta1.Coin // e.g., 1000000000 uPOKT
    TargetNumRelays uint64                   // e.g., 100 (target relays per session)
}
```

### Data Collection Points

#### **1. Historical Claims Data**

```bash
# Query claims by time range
pocketd q proof list-claims --session-end-height <height>

# Query specific claim details
pocketd q proof show-claim <session_id> <supplier_address>
```

**Key Fields:**
- `session_header`: Session information (height, service, application)
- `supplier_operator_address`: Supplier that submitted the claim
- `root_hash`: Merkle root of relay data
- `proof_validation_status`: Current proof status
- `num_relays`: Number of relays in claim
- `num_claimed_compute_units`: Actual compute units claimed
- `num_estimated_compute_units`: Difficulty-adjusted compute units
- `claimed_upokt`: Token amount claimed

#### **2. Proof Submissions Data**

```bash
# Query proof details
pocketd q proof show-proof <session_id> <supplier_address>

# Query proof parameters
pocketd q proof params
```

**Key Metrics:**
- Proof submission success rate
- Proof window timing compliance
- Proof validation failures
- Penalty amounts assessed

#### **3. Parameter History Tracking**

```bash
# Query current parameters
pocketd q tokenomics params
pocketd q shared params
pocketd q service params
pocketd q proof params

# Track parameter changes
pocketd q tx --query="message.action='/pocket.tokenomics.MsgUpdateParams'"
```

#### **4. Token Flow Analysis**

```bash
# Track minting events
pocketd q tx --query="message.action='/cosmos.bank.v1beta1.MsgMint'"

# Track burning events
pocketd q tx --query="message.action='/cosmos.bank.v1beta1.MsgBurn'"

# Monitor balance changes
pocketd q bank balances <address>
```

#### **5. Stake Movement Tracking**

```bash
# Application stake changes
pocketd q application list-applications
pocketd q application show-application <address>

# Supplier stake changes
pocketd q supplier list-suppliers
pocketd q supplier show-supplier <address>

# Slashing events
pocketd q tx --query="message.action='/pocket.tokenomics.EventSupplierSlashed'"
```

#### **6. Service Metrics Collection**

```bash
# Service information
pocketd q service list-services
pocketd q service show-service <service_id>

# Relay mining difficulty
pocketd q service relay-mining-difficulty <service_id>
pocketd q service relay-mining-difficulty-all
```

**Key Metrics:**
- Relay count per session
- Difficulty multiplier trends
- EMA calculations
- Target vs actual relay counts

#### **7. Network State Monitoring**

```bash
# Active participants
pocketd q application list-applications
pocketd q supplier list-suppliers
pocketd q gateway list-gateways

# Session assignments
pocketd q session list-sessions
```

#### **8. Real-time Monitoring Commands**

```bash
# Monitor live events
pocketd q tx --query="message.action='/pocket.tokenomics.EventClaimSettled'" --order_by=desc

# Track specific supplier activity
pocketd q tx --query="message.sender='<supplier_address>'" --order_by=desc

# Monitor parameter changes
pocketd q tx --query="message.action='/pocket.tokenomics.MsgUpdateParams'" --order_by=desc
```

#### **9. REST API Endpoints**

```bash
# REST API alternatives
curl "http://localhost:1317/pokt-network/poktroll/tokenomics/params"
curl "http://localhost:1317/pokt-network/poktroll/proof/claim"
curl "http://localhost:1317/pokt-network/poktroll/service/relay_mining_difficulty"
```

### Query Performance Tips

1. **Use height-based queries** for historical data to avoid timeouts
2. **Batch queries** when collecting large datasets
3. **Use specific filters** (address, session_id) to reduce response size
4. **Monitor block height** to understand data freshness
5. **Cache frequently accessed data** (parameters, service configs)

### Common Query Patterns

```bash
# Daily revenue analysis
for height in $(seq $START_HEIGHT $END_HEIGHT); do
    pocketd q proof list-claims --session-end-height $height --supplier-operator-address $SUPPLIER
done

# Monthly parameter tracking
pocketd q tx --query="message.action='/pocket.tokenomics.MsgUpdateParams'" \
    --page 1 --limit 100 --order_by desc

# Service performance comparison
for service in "ethereum" "polygon" "arbitrum"; do
    pocketd q service relay-mining-difficulty $service
done
```
