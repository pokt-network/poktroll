# Protocol Upgrade Backward Compatibility Strategy

## Overview

This document outlines a methodology for performing protocol upgrades that affect
multiple components in the Pocket network:

- The onchain logic
- RelayMiners
- Application/Gateway instances

The strategy ensures backward compatibility during transitions where different network
participants may be running different software versions. This is especially critical
for participants that cannot be centrally coordinated to upgrade simultaneously.

## Upgrade Methodology

Upgrades are performed in a way that allows new and old components to coexist
and must be performed in this specific order:

### 1. **Gateway Logic (Application/Gateway)**
- **Upgrade approach**: Update Gateway logic to be backward compatible with old RelayMiners and onchain logic
- **Control level**: Full control - presently operated by Grove, allowing coordinated upgrades
- **Timing**:
  - Gateways must be capable of working with both old and new RelayMiners.
  - Gateways must be able to query non-upgraded full nodes.

### 2. **Onchain Logic (Validators/Full Nodes)**
- **Upgrade approach**: Update onchain logic to be backward compatible with old RelayMiners
- **Control level**: The upgrade becomes canonical once blocks are produced with the new logic
- **Timing**: Coordinated upgrade through network consensus

### 3. **RelayMiner Logic**
- **Upgrade approach**: RelayMiners upgrade independently when ready
- **Control level**: No control - operated by independent third parties
- **Timing**: Cannot be coordinated; upgrades happen at operator discretion

:::warning
**Release Classification Required**: Pocket releases should clearly indicate whether they are:
- **Off-chain only** (RelayMiner updates)
- **Onchain only** (Onchain updates)
- **Both** (RelayMiner and onchain updates)

This prevents RelayMiner operators from rushing to deploy new binaries before the corresponding
onchain upgrade is live, which would cause their RelayMiners to error due to protocol mismatches.
:::

:::notice
After all actors have been upgraded, the backward compatibility logic can be removed in a future release.
:::

## Why This Method Works

The backward compatibility approach works because it follows a **feature detection pattern**:

1. **Field-based Version Detection**: The presence or absence of specific fields in protocol messages indicates which version created the message
2. **Conditional Processing**: Each component uses conditional logic to handle both old and new message formats
3. **Graceful Degradation**: When encountering older message formats, components automatically fall back to legacy processing methods
4. **Forward Compatibility**: New components can process messages from older versions without requiring updates

This approach ensures:
- **Network continuity**: No service interruption during upgrades
- **Operational flexibility**: Decentralized operators can upgrade at their own pace
- **Risk mitigation**: Eliminates the need for synchronized network-wide upgrades

### Compatibility Matrix

| Gateway Version | Chain Version | RelayMiner Version | Payload Present | PayloadHash Present | Gateway Signature Verification      | Onchain Signature Verification      |
|-----------------|---------------|--------------------|-----------------|---------------------|-------------------------------------|-------------------------------------|
| **New**         | Old           | Old                | ✅              | ❌                  | ✅ Compatible (backward-compatible) | ✅ Compatible (aligned)             |
| **New**         | New           | Old                | ✅              | ❌                  | ✅ Compatible (backward-compatible) | ✅ Compatible (backward-compatible) |
| **New**         | New           | New                | ❌              | ✅                  | ✅ Compatible (aligned)             | ✅ Compatible (aligned)             |

## Example: Relay Response Signature Upgrade

The following example demonstrates this methodology applied to a relay response signature upgrade:

### Background
- **Old behavior**: RelayMiners sign the full response payload
- **New behavior**: RelayMiners sign only the payload hash (for efficiency and reduced SMST/proof size)
- Signature generation and verification logic is shared between all actors

### Backward Compatibility Implementation

`RelayResponse#GetSignableBytesHash`: Returns the hash of the signable bytes of the relay response

The upgrade uses conditional logic in the `GetSignableBytesHash()` method that automatically picks the right fields to generate the signable bytes based on available data:

```go
// Conditional logic that detects version and picks appropriate fields
// If res.PayloadHash is present, we are facing an upgraded RelayMiner
if res.PayloadHash != nil {
    res.Payload = nil  // New method: hash without payload, rely on PayloadHash
} else {
    // Old method: keep payload for hashing, PayloadHash remains nil
}
```

## Detailed Compatibility Cases

### Case 1: New Gateway + Old Chain + Old RelayMiner
**Scenario**: Gateway has been upgraded, but RelayMiner and onchain logic are still running old software

**Behavior**:
- RelayMiner sends response with full `Payload` (no `PayloadHash`)
- Gateway receives response with payload present and verifies signature against the full payload (backward-compatible logic)
- Chain receives a proof with payload present and verifies signature (aligned on old logic)
- ✅ **Result**: All components work correctly despite mixed versions

**Why it works**:
- The Gateway has backward-compatible logic that detects the absence of `PayloadHash` and falls back to using the full `Payload` for verification
- Onchain and RelayMiner logic are aligned, so they can process the full payload without issues

### Case 2: New Gateway + Upgraded Chain + Old RelayMiner
**Scenario**: Both onchain and Gateway are upgraded but RelayMiners are still running old software

**Behavior**:
- RelayMiner sends response with full `Payload` (no `PayloadHash`)
- Gateway receives response with payload present and verifies signature against the full payload (backward-compatible logic)
- Chain receives a proof with payload present and verifies signature against the full payload (backward-compatible logic)
- ✅ **Result**: All components work correctly despite mixed versions

**Why it works**: The signature verification logic detects the absence of `PayloadHash` and falls back to using the full `Payload` for verification.


### Case 3: New Gateway + Upgraded Chain + New RelayMiner
**Scenario**: All components are running new software

**Behavior**:
- RelayMiner computes payload hash and sends response with `PayloadHash` (no `Payload`)
- Gateway receives response with payload hash present and verifies signature against the payload hash (aligned with new logic)
- Chain receives a proof with payload hash present and verifies signature against the payload hash (aligned with new logic)
- ✅ **Result**: All components use the new efficient method

**Why it works**: All components use the new payload hash method consistently, achieving optimal performance.