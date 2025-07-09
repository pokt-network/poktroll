---
title: Inspecting Tokenomics Data
sidebar_position: 7
---

:::warning Work in Progress
This document is a work-in-progress guide for analyzing tokenomics data. It provides workflows and data requirements for different stakeholder types. Implementation details and specific queries will be added as tooling develops.
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

**TODO**: Add specific query examples and calculation formulas

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

**TODO**: Add historical data aggregation methods and forecasting formulas

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

**TODO**: Add claim status tracking queries and deadline calculations

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

**TODO**: Add cost calculation formulas and optimization strategies

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

**TODO**: Add quality metrics definition and benchmarking methods

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

**TODO**: Add network health metrics and sustainability indicators

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

**TODO**: Add parameter sensitivity analysis and modeling tools

---

## Common Data Requirements

### Core Events to Track

```go
// Claims and Settlements
EventClaimCreated
EventClaimSettled
EventClaimExpired
EventProofSubmitted
EventProofUpdated

// Tokenomics Operations
EventApplicationOverserviced
EventSupplierSlashed
EventRelayMiningDifficultyUpdated
EventTokenomicsParamsUpdated

// Token Operations (from bank module)
EventTransfer
EventBurn
EventMint
```

### Key Parameters

```go
// Tokenomics Module Parameters
type Params struct {
    MintAllocationPercentages       MintAllocationPercentages
    DaoRewardAddress                string
    GlobalInflationPerClaim         float64
    MintEqualsBurnClaimDistribution MintEqualsBurnClaimDistribution
}

// Shared Module Parameters
type SharedParams struct {
    ComputeUnitsToTokensMultiplier uint64
    // ... other shared params
}

// Service-specific parameters
type Service struct {
    ComputeUnitsPerRelay uint64
    // ... other service config
}
```

### Data Collection Points

1. **Historical Claims**: Session details, compute units, settlement amounts
2. **Proof Submissions**: Success rates, timing, penalties
3. **Parameter History**: Network parameter changes over time
4. **Token Flows**: Mints, burns, transfers by reason/module
5. **Stake Movements**: Staking, unstaking, slashing events
6. **Service Metrics**: Relay counts, pricing, quality scores
7. **Network State**: Active participants, session assignments
