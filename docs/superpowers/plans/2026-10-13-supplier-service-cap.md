# Supplier Service Cap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cap how many services a supplier can serve by its stake, shipped disabled (price 0) and tightened later by governance.

**Architecture:** Two `x/supplier` params feed one pure function, `AllowedServices`. The cap is checked when staking (`StakeSupplier`), enforced after a slash (`slashSupplierStake` → `EnforceServiceCap`), and enforced after a governance param change by a batched, cursor-driven EndBlocker sweep. Enforcement only ever schedules service deactivation at the next session start through the existing service config history, so session hydration and past sessions are untouched. Priority is the operator's stake message order, stored in a new `Supplier.service_priority` field.

**Tech Stack:** Go 1.27.2, Cosmos SDK v0.53.7, gogoproto, `make proto_regen`, `make go_mockgen`, testify, gocuke E2E.

**Spec:** `docs/superpowers/specs/2026-10-13-supplier-service-cap-design.md`

## Global Constraints

- Branch `feat/supplier-service-cap`, worktree `../poktroll-svccap`, based on `origin/r/v0.1.36`. Public branch: commit messages and comments must not mention unrelated security work.
- Toolchain: `GOTOOLCHAIN=go1.27.2`. Tests need build tag `test` (`-tags test`); integration tests `-tags test,integration`.
- `min_stake_included_services`: `uint64`, at least 1, default 1.
- `stake_per_additional_service`: `*cosmos.base.v1beta1.Coin`, denom `upokt`, amount >= 0, default 0 upokt. Nil or 0 means the cap is disabled.
- `AllowedServices`: disabled ⇒ unlimited; `stake < min_stake` ⇒ 0; else `included + floor((stake - min_stake) / price)`.
- Enforcement never unbonds a supplier and never panics. The EndBlocker sweep never returns an error.
- Deactivation always lands on `GetNextSessionStartHeight` of the current height.
- A config is "counted" when `DeactivationHeight == NoDeactivationHeight || DeactivationHeight > nextSessionStartHeight` (the same predicate `BeginSupplierUnbonding` uses).
- Upgrade handler writes `min_stake_included_services = 1` and `stake_per_additional_service = 0upokt`.
- Sweep batch size `K = 200` suppliers per block, a code constant.
- Run `make go_lint` (golangci-lint v2.14) before every commit that touches Go.

## Review Focus

1. **Owner stakes with zero services, cap enabled.** The supplier must stay staked and untouched by slash and sweep. Pinned in Task 4 and Task 6.
2. **Operator update with no stake field.** The cap check must use the stake `updateSupplier` filled in, not a nil `msg.Stake`. Pinned in Task 3.
3. **Bulk `MsgUpdateParams` from the existing JSON templates.** Today's templates lack the new fields, so a bulk update would set `included = 0` and be rejected. Templates must carry the new fields. Pinned in Task 1 (template) and Task 6 (bulk update triggers the sweep only on a cap-relevant change).
4. **Second slash on the same supplier in one settlement block.** It must see the first slash's deactivations, and the end-of-settlement flush must not undo them. Pinned in Task 5.
5. **Supplier already over the cap when params tighten mid-sweep.** The cursor must reset and the sweep must restart with the new values. Pinned in Task 6.

---

## File Structure

| File | Responsibility |
|---|---|
| `proto/pocket/supplier/params.proto` | Add the two params (fields 3, 4). |
| `proto/pocket/supplier/tx.proto` | Add `as_uint64 = 4` to `MsgUpdateParam.asType`. |
| `proto/pocket/shared/supplier.proto` | Add `repeated string service_priority = 7` to `Supplier`. |
| `proto/pocket/supplier/event.proto` | Add `ServiceCapReason` enum and `EventSupplierServicesCapped`. |
| `x/supplier/types/params.go` | Keys, defaults, validation for the new params. |
| `x/supplier/types/message_update_param.go` | Accept the new params in `MsgUpdateParam`. |
| `x/supplier/types/errors.go` | `ErrSupplierTooManyServices` (1110). |
| `x/supplier/types/service_cap.go` (new) | `AllowedServices`, `RequiredStake`, `IsCountedServiceConfig`, `RankServiceConfigs`, `ServicePriorityFromConfigs`. Pure, no store. |
| `x/supplier/types/keys.go` | `ServiceCapSweepKey`. |
| `x/supplier/keeper/msg_server_stake_supplier.go` | Write `service_priority`; cap check after the `min_stake` check. |
| `x/supplier/keeper/service_cap.go` (new) | `EnforceServiceCap`, sweep scheduling, `EndBlockerServiceCapSweep`. |
| `x/supplier/keeper/msg_server_update_param.go`, `msg_update_params.go` | Handle new params; schedule the sweep. |
| `x/supplier/module/abci.go` | Call the sweep in `EndBlocker`. |
| `x/session/keeper/session_hydrator.go` | Strip `ServicePriority` from session suppliers. |
| `x/tokenomics/types/expected_keepers.go` | Add `EnforceServiceCap` to `SupplierKeeper`. |
| `x/tokenomics/keeper/settle_pending_claims.go` | Call `EnforceServiceCap` after a slash that stays above `min_stake`. |
| `app/upgrades/vNEXT.go` | Seed the two params. |
| `config.yml`, `tools/scripts/params/**/supplier*.json` | New param values in LocalNet and governance templates. |
| `e2e/tests/parse_params_test.go`, `update_params_test.go`, `update_params.feature` | E2E param plumbing and scenarios. |
| `docusaurus/docs/1_operate/3_configs/3_supplier_staking_config.md` | Operator docs: list order is priority. |

---

### Task 1: Params, proto and plumbing

**Files:**
- Modify: `proto/pocket/supplier/params.proto`, `proto/pocket/supplier/tx.proto`, `proto/pocket/shared/supplier.proto`, `proto/pocket/supplier/event.proto`
- Modify: `x/supplier/types/params.go`, `x/supplier/types/message_update_param.go`, `x/supplier/types/errors.go`
- Modify: `x/supplier/keeper/msg_server_update_param.go`
- Modify: `config.yml`, `tools/scripts/params/params_templates/supplier_all.json`, `tools/scripts/params/bulk_params_{alpha,beta,main}/supplier_params.json`
- Create: `tools/scripts/params/params_templates/supplier_min_stake_included_services.json`, `tools/scripts/params/params_templates/supplier_stake_per_additional_service.json`
- Test: `x/supplier/types/params_test.go`, `x/supplier/keeper/msg_server_update_param_test.go`

**Interfaces:**
- Produces: `types.ParamMinStakeIncludedServices = "min_stake_included_services"`, `types.ParamStakePerAdditionalService = "stake_per_additional_service"`, `types.DefaultMinStakeIncludedServices uint64 = 1`, `types.DefaultStakePerAdditionalService = cosmostypes.NewInt64Coin("upokt", 0)`, `types.ValidateMinStakeIncludedServices(any) error`, `types.ValidateStakePerAdditionalService(any) error`, `Params.MinStakeIncludedServices uint64`, `Params.StakePerAdditionalService *cosmostypes.Coin`, `sharedtypes.Supplier.ServicePriority []string`, `types.ServiceCapReason` (`SERVICE_CAP_REASON_UNSPECIFIED=0`, `SERVICE_CAP_REASON_SLASH=1`, `SERVICE_CAP_REASON_PARAMS=2`), `types.EventSupplierServicesCapped`, `types.ErrSupplierTooManyServices`, `types.MsgUpdateParam_AsUint64`.

- [ ] **Step 1: Add the proto fields**

`proto/pocket/supplier/params.proto`, after `staking_fee`:

```proto
  // min_stake_included_services is the number of services covered by min_stake.
  // Must be at least 1.
  uint64 min_stake_included_services = 3 [(gogoproto.jsontag) = "min_stake_included_services", (gogoproto.moretags) = "yaml:\"min_stake_included_services\""];

  // stake_per_additional_service is the additional stake required for each service
  // beyond min_stake_included_services. Zero disables the service cap.
  cosmos.base.v1beta1.Coin stake_per_additional_service = 4 [(gogoproto.jsontag) = "stake_per_additional_service", (gogoproto.moretags) = "yaml:\"stake_per_additional_service\""];
```

`proto/pocket/supplier/tx.proto`, inside `MsgUpdateParam`:

```proto
  oneof asType {
    cosmos.base.v1beta1.Coin as_coin = 3;
    uint64 as_uint64 = 4 [(gogoproto.jsontag) = "as_uint64"];
  }
```

`proto/pocket/shared/supplier.proto`, after `service_config_history = 6`:

```proto
  // Service IDs of the supplier's most recent stake message, in message order.
  // When stake does not cover every service, services earlier in this list are kept.
  repeated string service_priority = 7;
```

`proto/pocket/supplier/event.proto`, append:

```proto
// ServiceCapReason is why a supplier's services were capped.
enum ServiceCapReason {
  SERVICE_CAP_REASON_UNSPECIFIED = 0;
  // The supplier was slashed and its stake no longer covers its services.
  SERVICE_CAP_REASON_SLASH = 1;
  // Governance changed the service cap params.
  SERVICE_CAP_REASON_PARAMS = 2;
}

// EventSupplierServicesCapped is emitted when services beyond what a supplier's
// stake covers are scheduled for deactivation at the next session start.
message EventSupplierServicesCapped {
  string operator_address = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string owner_address = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  cosmos.base.v1beta1.Coin stake = 3 [(gogoproto.jsontag) = "stake"];
  uint64 allowed_services = 4 [(gogoproto.jsontag) = "allowed_services"];
  repeated string deactivated_service_ids = 5 [(gogoproto.jsontag) = "deactivated_service_ids"];
  ServiceCapReason reason = 6 [(gogoproto.jsontag) = "reason"];
  int64 effective_height = 7 [(gogoproto.jsontag) = "effective_height"];
}
```

- [ ] **Step 2: Regenerate**

Run: `make proto_regen`
Expected: exit 0; `git status` shows regenerated `*.pb.go` and `api/pocket/**` files.

- [ ] **Step 3: Write the failing params tests**

Append to `x/supplier/types/params_test.go` (create it if missing, `package types`):

```go
func TestValidateMinStakeIncludedServices(t *testing.T) {
	require.NoError(t, ValidateMinStakeIncludedServices(uint64(1)))
	require.NoError(t, ValidateMinStakeIncludedServices(uint64(1000)))
	require.ErrorIs(t, ValidateMinStakeIncludedServices(uint64(0)), ErrSupplierParamInvalid)
	require.ErrorIs(t, ValidateMinStakeIncludedServices("1"), ErrSupplierParamInvalid)
}

func TestValidateStakePerAdditionalService(t *testing.T) {
	zero := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 5_000_000_000)
	wrongDenom := cosmostypes.NewInt64Coin("stake", 1)
	require.NoError(t, ValidateStakePerAdditionalService(&zero))
	require.NoError(t, ValidateStakePerAdditionalService(&price))
	require.ErrorIs(t, ValidateStakePerAdditionalService(&wrongDenom), ErrSupplierParamInvalid)
	require.ErrorIs(t, ValidateStakePerAdditionalService((*cosmostypes.Coin)(nil)), ErrSupplierParamInvalid)
}

func TestDefaultParamsValidate(t *testing.T) {
	require.NoError(t, DefaultParams().Validate())
	params := DefaultParams()
	params.MinStakeIncludedServices = 0
	require.ErrorIs(t, params.Validate(), ErrSupplierParamInvalid)
}
```

Imports: `testing`, `cosmostypes "github.com/cosmos/cosmos-sdk/types"`, `"github.com/stretchr/testify/require"`, `"github.com/pokt-network/poktroll/app/pocket"`.

- [ ] **Step 4: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/types/ -run 'TestValidateMinStakeIncludedServices|TestValidateStakePerAdditionalService|TestDefaultParamsValidate'`
Expected: FAIL, `undefined: ValidateMinStakeIncludedServices`.

- [ ] **Step 5: Implement params**

In `x/supplier/types/params.go`, extend the `var` block:

```go
	KeyMinStakeIncludedServices      = []byte("MinStakeIncludedServices")
	ParamMinStakeIncludedServices    = "min_stake_included_services"
	DefaultMinStakeIncludedServices  = uint64(1)
	KeyStakePerAdditionalService     = []byte("StakePerAdditionalService")
	ParamStakePerAdditionalService   = "stake_per_additional_service"
	DefaultStakePerAdditionalService = cosmostypes.NewInt64Coin("upokt", 0) // 0 disables the service cap
```

Replace `NewParams` and `DefaultParams`:

```go
func NewParams(
	minStake *cosmostypes.Coin,
	stakingFee *cosmostypes.Coin,
	minStakeIncludedServices uint64,
	stakePerAdditionalService *cosmostypes.Coin,
) Params {
	return Params{
		MinStake:                  minStake,
		StakingFee:                stakingFee,
		MinStakeIncludedServices:  minStakeIncludedServices,
		StakePerAdditionalService: stakePerAdditionalService,
	}
}

func DefaultParams() Params {
	return NewParams(
		&DefaultMinStake,
		&DefaultStakingFee,
		DefaultMinStakeIncludedServices,
		&DefaultStakePerAdditionalService,
	)
}
```

Add to `ParamSetPairs`:

```go
		paramtypes.NewParamSetPair(
			KeyMinStakeIncludedServices,
			&p.MinStakeIncludedServices,
			ValidateMinStakeIncludedServices,
		),
		paramtypes.NewParamSetPair(
			KeyStakePerAdditionalService,
			&p.StakePerAdditionalService,
			ValidateStakePerAdditionalService,
		),
```

Add to `Validate()` before `return nil`:

```go
	if err := ValidateMinStakeIncludedServices(p.MinStakeIncludedServices); err != nil {
		return err
	}

	if err := ValidateStakePerAdditionalService(p.StakePerAdditionalService); err != nil {
		return err
	}
```

Add the validators:

```go
// ValidateMinStakeIncludedServices validates the MinStakeIncludedServices param.
// At least one service is included: a staked supplier exists to serve.
func ValidateMinStakeIncludedServices(includedAny any) error {
	included, ok := includedAny.(uint64)
	if !ok {
		return ErrSupplierParamInvalid.Wrapf("invalid parameter type: %T", includedAny)
	}

	if included < 1 {
		return ErrSupplierParamInvalid.Wrapf("min_stake_included_services must be at least 1: got %d", included)
	}

	return nil
}

// ValidateStakePerAdditionalService validates the StakePerAdditionalService param.
// Zero is valid and disables the service cap.
func ValidateStakePerAdditionalService(priceAny any) error {
	if _, err := paramAsPositiveuPOKT(priceAny, "stake_per_additional_service"); err != nil {
		return err
	}

	return nil
}
```

In `x/supplier/types/errors.go`, append:

```go
	ErrSupplierTooManyServices      = sdkerrors.Register(ModuleName, 1110, "stake does not cover the number of services")
```

- [ ] **Step 6: Run params tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/types/`
Expected: PASS.

- [ ] **Step 7: Write the failing MsgUpdateParam tests**

Append to `x/supplier/keeper/msg_server_update_param_test.go`:

```go
func TestMsgUpdateParam_UpdateMinStakeIncludedServices(t *testing.T) {
	k, msgSrv, ctx := setupMsgServer(t)
	require.NoError(t, k.SetParams(ctx, suppliertypes.DefaultParams()))

	_, err := msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamMinStakeIncludedServices,
		AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: 35},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(35), k.GetParams(ctx).MinStakeIncludedServices)

	_, err = msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamMinStakeIncludedServices,
		AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: 0},
	})
	require.Error(t, err)
	require.Equal(t, uint64(35), k.GetParams(ctx).MinStakeIncludedServices)
}

func TestMsgUpdateParam_UpdateStakePerAdditionalService(t *testing.T) {
	k, msgSrv, ctx := setupMsgServer(t)
	require.NoError(t, k.SetParams(ctx, suppliertypes.DefaultParams()))

	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 5_000_000_000)
	_, err := msgSrv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakePerAdditionalService,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &price},
	})
	require.NoError(t, err)
	require.Equal(t, price.Amount, k.GetParams(ctx).StakePerAdditionalService.Amount)
}
```

- [ ] **Step 8: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/keeper/ -run 'TestMsgUpdateParam_Update(MinStakeIncludedServices|StakePerAdditionalService)'`
Expected: FAIL, `unsupported param`.

- [ ] **Step 9: Wire MsgUpdateParam**

`x/supplier/types/message_update_param.go`, `NewMsgUpdateParam` switch:

```go
	case uint64:
		asTypeIface = &MsgUpdateParam_AsUint64{AsUint64: t}
```

`ValidateBasic` switch, before `default`:

```go
	case ParamMinStakeIncludedServices:
		if err := genericParamTypeIs[*MsgUpdateParam_AsUint64](msg); err != nil {
			return err
		}
		return ValidateMinStakeIncludedServices(msg.GetAsUint64())
	case ParamStakePerAdditionalService:
		if err := genericParamTypeIs[*MsgUpdateParam_AsCoin](msg); err != nil {
			return err
		}
		return ValidateStakePerAdditionalService(msg.GetAsCoin())
```

`x/supplier/keeper/msg_server_update_param.go` switch, before `default`:

```go
	case suppliertypes.ParamMinStakeIncludedServices:
		logger = logger.With("min_stake_included_services", msg.GetAsUint64())
		params.MinStakeIncludedServices = msg.GetAsUint64()
	case suppliertypes.ParamStakePerAdditionalService:
		logger = logger.With("stake_per_additional_service", msg.GetAsCoin())
		params.StakePerAdditionalService = msg.GetAsCoin()
```

- [ ] **Step 10: Update config and templates**

`config.yml`, supplier `params:` after `staking_fee`:

```yaml
        # Services covered by min_stake. At least 1.
        min_stake_included_services: 1
        # Stake per service beyond the included ones. 0 disables the service cap.
        stake_per_additional_service:
          amount: "0"
          denom: upokt
```

In `tools/scripts/params/params_templates/supplier_all.json` and each `tools/scripts/params/bulk_params_{alpha,beta,main}/supplier_params.json`, add inside `"params"` (keep each file's existing `min_stake` / `staking_fee` values unchanged):

```json
          "min_stake_included_services": "1",
          "stake_per_additional_service": {
            "amount": "0",
            "denom": "upokt"
          }
```

Create `tools/scripts/params/params_templates/supplier_min_stake_included_services.json`:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.supplier.MsgUpdateParam",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "min_stake_included_services",
        "as_uint64": "1"
      }
    ]
  }
}
```

Create `tools/scripts/params/params_templates/supplier_stake_per_additional_service.json`:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.supplier.MsgUpdateParam",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "stake_per_additional_service",
        "as_coin": {
          "amount": "0",
          "denom": "upokt"
        }
      }
    ]
  }
}
```

Check each JSON parses: `for f in tools/scripts/params/params_templates/supplier_*.json tools/scripts/params/bulk_params_*/supplier_params.json; do python3 -m json.tool "$f" >/dev/null || echo "BAD $f"; done`
Expected: no `BAD` lines.

- [ ] **Step 11: Run the module tests and lint**

Run: `GOTOOLCHAIN=go1.27.2 go build ./... && GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/... && make go_lint`
Expected: PASS, lint 0 issues. If any caller of `types.NewParams` fails to compile, pass `types.DefaultMinStakeIncludedServices, &types.DefaultStakePerAdditionalService` for the new arguments.

- [ ] **Step 12: Commit**

```bash
git add proto api x config.yml tools/scripts/params
git commit -m "[Supplier] Add service cap params"
```

---

### Task 2: `AllowedServices` and ranking helpers

**Files:**
- Create: `x/supplier/types/service_cap.go`
- Test: `x/supplier/types/service_cap_test.go`

**Interfaces:**
- Consumes: `Params.MinStake`, `Params.MinStakeIncludedServices`, `Params.StakePerAdditionalService`, `DefaultMinStake` (Task 1).
- Produces:
  - `func (p Params) ServiceCapEnabled() bool`
  - `func AllowedServices(stake cosmostypes.Coin, params Params) (allowed uint64, limited bool)` (`limited == false` means unlimited)
  - `func RequiredStake(numServices uint64, params Params) cosmostypes.Coin`
  - `func IsCountedServiceConfig(cfg *sharedtypes.ServiceConfigUpdate, nextSessionStartHeight int64) bool`
  - `func RankServiceConfigs(configs []*sharedtypes.ServiceConfigUpdate, priority []string) []*sharedtypes.ServiceConfigUpdate`
  - `func ServicePriorityFromConfigs(services []*sharedtypes.SupplierServiceConfig) []string`

- [ ] **Step 1: Write the failing tests**

`x/supplier/types/service_cap_test.go`:

```go
package types

import (
	"math"
	"testing"

	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func upokt(amount int64) cosmostypes.Coin { return cosmostypes.NewInt64Coin(pocket.DenomuPOKT, amount) }

func capParams(minStake, price int64, included uint64) Params {
	minStakeCoin, priceCoin := upokt(minStake), upokt(price)
	return Params{MinStake: &minStakeCoin, StakePerAdditionalService: &priceCoin, MinStakeIncludedServices: included}
}

func TestAllowedServices(t *testing.T) {
	tests := []struct {
		desc        string
		stake       int64
		params      Params
		wantAllowed uint64
		wantLimited bool
	}{
		{"price 0 disables the cap", 10, capParams(10, 0, 1), 0, false},
		{"nil price disables the cap", 10, Params{MinStake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: sdkmath.NewInt(10)}, MinStakeIncludedServices: 1}, 0, false},
		{"below min stake allows none", 9, capParams(10, 5, 1), 0, true},
		{"exactly min stake allows included", 10, capParams(10, 5, 3), 3, true},
		{"one upokt short of the next step", 14, capParams(10, 5, 3), 3, true},
		{"exactly the next step", 15, capParams(10, 5, 3), 4, true},
		{"many steps", 60, capParams(10, 5, 1), 11, true},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			allowed, limited := AllowedServices(upokt(test.stake), test.params)
			require.Equal(t, test.wantLimited, limited)
			require.Equal(t, test.wantAllowed, allowed)
		})
	}
}

func TestAllowedServices_SaturatesInsteadOfOverflowing(t *testing.T) {
	huge := cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: sdkmath.NewIntFromUint64(math.MaxUint64).MulRaw(4)}
	allowed, limited := AllowedServices(huge, capParams(1, 1, math.MaxUint64-1))
	require.True(t, limited)
	require.Equal(t, uint64(math.MaxUint64), allowed)
}

func TestRequiredStake(t *testing.T) {
	params := capParams(10, 5, 3)
	require.Equal(t, upokt(10), RequiredStake(0, params))
	require.Equal(t, upokt(10), RequiredStake(3, params))
	require.Equal(t, upokt(15), RequiredStake(4, params))
	require.Equal(t, upokt(30), RequiredStake(7, params))
	// Required stake for N services is always allowed to serve N.
	for n := uint64(1); n <= 20; n++ {
		allowed, _ := AllowedServices(RequiredStake(n, params), params)
		require.GreaterOrEqual(t, allowed, n)
	}
}

func cfg(serviceId string, deactivationHeight int64) *sharedtypes.ServiceConfigUpdate {
	return &sharedtypes.ServiceConfigUpdate{
		Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
		DeactivationHeight: deactivationHeight,
	}
}

func TestIsCountedServiceConfig(t *testing.T) {
	require.True(t, IsCountedServiceConfig(cfg("a", sharedtypes.NoDeactivationHeight), 100))
	require.True(t, IsCountedServiceConfig(cfg("a", 120), 100))
	require.False(t, IsCountedServiceConfig(cfg("a", 100), 100))
	require.False(t, IsCountedServiceConfig(cfg("a", 80), 100))
}

func TestRankServiceConfigs(t *testing.T) {
	configs := []*sharedtypes.ServiceConfigUpdate{cfg("eth", 0), cfg("base", 0), cfg("sol", 0), cfg("arb", 0)}

	ranked := RankServiceConfigs(configs, []string{"sol", "eth"})
	require.Equal(t, []string{"sol", "eth", "arb", "base"}, serviceIds(ranked))

	// Empty priority (supplier staked before the field existed) falls back to service ID order.
	ranked = RankServiceConfigs(configs, nil)
	require.Equal(t, []string{"arb", "base", "eth", "sol"}, serviceIds(ranked))

	// The input slice is not reordered, and returned pointers are the input pointers.
	require.Equal(t, []string{"eth", "base", "sol", "arb"}, serviceIds(configs))
	require.Same(t, configs[0], RankServiceConfigs(configs, []string{"eth"})[0])
}

func TestServicePriorityFromConfigs(t *testing.T) {
	services := []*sharedtypes.SupplierServiceConfig{{ServiceId: "sol"}, {ServiceId: "eth"}}
	require.Equal(t, []string{"sol", "eth"}, ServicePriorityFromConfigs(services))
	require.Empty(t, ServicePriorityFromConfigs(nil))
}

func serviceIds(configs []*sharedtypes.ServiceConfigUpdate) []string {
	ids := make([]string, 0, len(configs))
	for _, config := range configs {
		ids = append(ids, config.Service.ServiceId)
	}
	return ids
}
```

- [ ] **Step 2: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/types/ -run 'TestAllowedServices|TestRequiredStake|TestIsCountedServiceConfig|TestRankServiceConfigs|TestServicePriorityFromConfigs'`
Expected: FAIL, `undefined: AllowedServices`.

- [ ] **Step 3: Implement**

`x/supplier/types/service_cap.go`:

```go
package types

import (
	"cmp"
	"math"
	"slices"

	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ServiceCapEnabled reports whether stake limits the number of services.
// A nil or zero stake_per_additional_service disables the cap.
func (p Params) ServiceCapEnabled() bool {
	return p.StakePerAdditionalService != nil && p.StakePerAdditionalService.IsPositive()
}

// minStakeOrDefault mirrors the nil fallback used by the unbonding paths.
func (p Params) minStakeOrDefault() cosmostypes.Coin {
	if p.MinStake == nil {
		return DefaultMinStake
	}
	return *p.MinStake
}

// AllowedServices returns how many services the given stake covers.
// limited is false when the cap is disabled, in which case allowed is meaningless.
// A stake below min_stake covers no services.
func AllowedServices(stake cosmostypes.Coin, params Params) (allowed uint64, limited bool) {
	if !params.ServiceCapEnabled() {
		return 0, false
	}

	minStake := params.minStakeOrDefault()
	if stake.Amount.LT(minStake.Amount) {
		return 0, true
	}

	additional := stake.Amount.Sub(minStake.Amount).Quo(params.StakePerAdditionalService.Amount)
	if !additional.IsUint64() || additional.Uint64() > math.MaxUint64-params.MinStakeIncludedServices {
		return math.MaxUint64, true
	}

	return params.MinStakeIncludedServices + additional.Uint64(), true
}

// RequiredStake returns the stake needed to serve numServices services.
// With the cap disabled it is min_stake.
func RequiredStake(numServices uint64, params Params) cosmostypes.Coin {
	minStake := params.minStakeOrDefault()
	if !params.ServiceCapEnabled() || numServices <= params.MinStakeIncludedServices {
		return minStake
	}

	additional := sdkmath.NewIntFromUint64(numServices - params.MinStakeIncludedServices)
	return cosmostypes.NewCoin(minStake.Denom, minStake.Amount.Add(additional.Mul(params.StakePerAdditionalService.Amount)))
}

// IsCountedServiceConfig reports whether a config counts against the cap: it is
// active, or pending activation, and not deactivated by the next session start.
func IsCountedServiceConfig(config *sharedtypes.ServiceConfigUpdate, nextSessionStartHeight int64) bool {
	return config.DeactivationHeight == sharedtypes.NoDeactivationHeight ||
		config.DeactivationHeight > nextSessionStartHeight
}

// RankServiceConfigs returns configs ordered by the supplier's service priority.
// Services missing from priority (suppliers staked before the field existed) rank
// after listed ones, by service ID. The input slice is not modified; the returned
// slice holds the same pointers.
func RankServiceConfigs(
	configs []*sharedtypes.ServiceConfigUpdate,
	priority []string,
) []*sharedtypes.ServiceConfigUpdate {
	// Lookup only: map iteration order never reaches the output.
	rank := make(map[string]int, len(priority))
	for i, serviceId := range priority {
		if _, seen := rank[serviceId]; !seen {
			rank[serviceId] = i
		}
	}

	ranked := slices.Clone(configs)
	slices.SortStableFunc(ranked, func(a, b *sharedtypes.ServiceConfigUpdate) int {
		rankA, listedA := rank[a.Service.ServiceId]
		rankB, listedB := rank[b.Service.ServiceId]
		switch {
		case listedA && listedB:
			return cmp.Compare(rankA, rankB)
		case listedA:
			return -1
		case listedB:
			return 1
		default:
			return cmp.Compare(a.Service.ServiceId, b.Service.ServiceId)
		}
	})

	return ranked
}

// ServicePriorityFromConfigs returns the service IDs in message order.
func ServicePriorityFromConfigs(services []*sharedtypes.SupplierServiceConfig) []string {
	priority := make([]string, 0, len(services))
	for _, service := range services {
		priority = append(priority, service.ServiceId)
	}
	return priority
}
```

- [ ] **Step 4: Run tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/types/`
Expected: PASS.

- [ ] **Step 5: Lint and commit**

```bash
make go_lint
git add x/supplier/types/service_cap.go x/supplier/types/service_cap_test.go
git commit -m "[Supplier] Add AllowedServices and service ranking"
```

---

### Task 3: Cap check and priority on stake

**Files:**
- Modify: `x/supplier/keeper/msg_server_stake_supplier.go` (`createSupplier` ~line 62, `updateSupplier` ~line 343, `StakeSupplier` after the `min_stake` check ~line 262)
- Modify: `testutil/keeper/supplier.go` (expose the service keeper), `x/supplier/keeper/msg_server_stake_supplier_test.go` (`newSupplierStakeMsg` expected supplier)
- Test: `x/supplier/keeper/msg_server_stake_supplier_service_cap_test.go` (new), `x/migration/keeper/msg_server_claim_morse_supplier_test.go`

**Interfaces:**
- Consumes: `AllowedServices`, `RequiredStake`, `IsCountedServiceConfig`, `ServicePriorityFromConfigs` (Task 2); `ErrSupplierTooManyServices` (Task 1).
- Produces: `Supplier.ServicePriority` populated on every stake message that lists services; `func (k Keeper) checkServiceCap(ctx context.Context, supplier *sharedtypes.Supplier, msg *suppliertypes.MsgStakeSupplier) error`.

- [ ] **Step 1: Expose the service keeper in the test keeper**

`testutil/keeper/supplier.go`: add a field to `SupplierModuleKeepers`:

```go
	// ServiceKeeper registers services used by tests.
	ServiceKeeper servicekeeper.Keeper
```

and set `ServiceKeeper: serviceKeeper,` in the returned `SupplierModuleKeepers{...}` literal.

In `x/supplier/keeper/msg_server_stake_supplier_test.go`, `newSupplierStakeMsg`, set the expected priority so `require.EqualValues(t, expectedSupplier, &supplier)` (line ~427) keeps passing:

```go
	expectedSupplier = &sharedtypes.Supplier{
		OwnerAddress:         ownerAddr,
		OperatorAddress:      operatorAddr,
		Stake:                &initialStake,
		Services:             make([]*sharedtypes.SupplierServiceConfig, 0),
		ServiceConfigHistory: serviceConfigHistory,
		ServicePriority:      serviceIds,
	}
```

If an existing test then fails because `serviceIds` is nil versus an empty slice, use `suppliertypes.ServicePriorityFromConfigs(services)` instead, which always returns a non-nil slice; match whatever `createSupplier` stores.

- [ ] **Step 1b: Write the failing tests**

`x/supplier/keeper/msg_server_stake_supplier_service_cap_test.go` (`package keeper_test`). Reuse `newSupplierStakeMsg`, `setStakeMsgSigner` and `setBlockHeightToNextSessionStart` from `msg_server_stake_supplier_test.go`; read their signatures there before writing the calls. The helper below builds a message with several services; it relies on the test keeper's `SetService` for each ID, as the existing tests do for `"svcId"`.

```go
package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	capMinStake = int64(1_000_000)
	capPrice    = int64(500_000)
)

// enableServiceCap sets min_stake=1 POKT, included=1, price=0.5 POKT.
func enableServiceCap(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) {
	t.Helper()
	params := k.GetParams(ctx)
	minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capMinStake)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capPrice)
	params.MinStake = &minStake
	params.MinStakeIncludedServices = 1
	params.StakePerAdditionalService = &price
	require.NoError(t, k.SetParams(ctx, params))
}

// multiServiceStakeMsg returns an operator-signed stake msg for the given services,
// registering each service first.
func multiServiceStakeMsg(
	t *testing.T,
	k keepertest.SupplierModuleKeepers,
	ctx cosmostypes.Context,
	ownerAddr, operatorAddr string,
	stake int64,
	serviceIds ...string,
) *suppliertypes.MsgStakeSupplier {
	t.Helper()
	for _, serviceId := range serviceIds {
		k.ServiceKeeper.SetService(ctx, sharedtypes.Service{Id: serviceId, ComputeUnitsPerRelay: 1, OwnerAddress: sample.AccAddressBech32()})
	}
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, stake, serviceIds...)
	return stakeMsg
}

func TestStakeSupplier_ServiceCap_RejectsOneOverAndAcceptsAtCap(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// 1 POKT covers 1 service; 3 services need 2 POKT.
	overMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake, "svc1", "svc2", "svc3")
	_, err := srv.StakeSupplier(ctx, overMsg)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())
	require.ErrorContains(t, err, "2000000upokt")

	atCapMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+2*capPrice, "svc1", "svc2", "svc3")
	_, err = srv.StakeSupplier(ctx, atCapMsg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_DisabledAllowsAnyCount(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	// Default params: price 0, cap disabled.
	msg := multiServiceStakeMsg(t, k, sdkCtx, sample.AccAddressBech32(), sample.AccAddressBech32(), 1_000_000, "svc1", "svc2", "svc3", "svc4")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_WritesServicePriorityInMessageOrder(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, 1_000_000, "sol", "eth", "base")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	// Reloaded from the store, not the in-memory supplier.
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.Equal(t, []string{"sol", "eth", "base"}, stored.ServicePriority)

	// A stake-only update keeps the priority.
	stakeOnly := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(2_000_000)},
	}
	_, err = srv.StakeSupplier(ctx, stakeOnly)
	require.NoError(t, err)
	stored, _ = k.GetDehydratedSupplier(ctx, operatorAddr)
	require.Equal(t, []string{"sol", "eth", "base"}, stored.ServicePriority)
}

func TestStakeSupplier_ServiceCap_NonCustodialFlow(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// 1. Owner stakes at min stake with no services.
	ownerMsg := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	}
	_, err := srv.StakeSupplier(ctx, ownerMsg)
	require.NoError(t, err)

	// 2. Operator sets 2 services without a stake field: rejected, stake covers 1.
	operatorMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, 0, "svc1", "svc2")
	operatorMsg.Stake = nil
	setStakeMsgSigner(operatorMsg, operatorAddr)
	_, err = srv.StakeSupplier(ctx, operatorMsg)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())

	// 3. Owner tops up; the operator retries and succeeds.
	ownerMsg.Stake = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake + capPrice)}
	_, err = srv.StakeSupplier(ctx, ownerMsg)
	require.NoError(t, err)
	operatorMsg.Stake = nil
	_, err = srv.StakeSupplier(ctx, operatorMsg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_StakeOnlyDecreaseBelowServicesRejected(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+capPrice, "svc1", "svc2")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	stakeOnly := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	}
	_, err = srv.StakeSupplier(ctx, stakeOnly)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())
}
```

`newSupplierStakeMsg` signs with the operator when services are listed and with the owner otherwise, which is what the non-custodial test needs.

In `x/migration/keeper/msg_server_claim_morse_supplier_test.go`, add one case modelled on the existing successful claim test: set supplier params with the cap enabled so the claimed stake covers fewer services than the claim lists, and assert the claim fails with `ErrSupplierTooManyServices`. Read the existing test first and copy its setup verbatim; change only the supplier params and the expected error.

- [ ] **Step 2: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/keeper/ -run 'TestStakeSupplier_ServiceCap'`
Expected: FAIL (no rejection; `ServicePriority` empty).

- [ ] **Step 3: Write `service_priority`**

In `createSupplier`, add to the `sharedtypes.Supplier{...}` literal:

```go
		ServicePriority:         suppliertypes.ServicePriorityFromConfigs(msg.Services),
```

In `updateSupplier`, right after the `if len(msg.Services) == 0 { return nil }` early return:

```go
	supplier.ServicePriority = suppliertypes.ServicePriorityFromConfigs(msg.Services)
```

- [ ] **Step 4: Add the check**

In `StakeSupplier`, directly after the `min_stake` check block (the one ending `return nil, status.Error(codes.InvalidArgument, err.Error())` after `"must stake at least %s"`):

```go
	// The stake must cover the supplier's services. msg.Stake is the stake that
	// will apply: updateSupplier filled it in when the message carried none.
	if err = k.checkServiceCap(ctx, &supplier, msg); err != nil {
		logger.Info(fmt.Sprintf("ERROR: %s", err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
```

Add the method at the end of the file:

```go
// checkServiceCap returns an error if msg.Stake does not cover the supplier's
// services: the services in the message, or for a stake-only update, the
// services that are active or pending activation.
func (k Keeper) checkServiceCap(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	msg *suppliertypes.MsgStakeSupplier,
) error {
	params := k.GetParams(ctx)
	allowed, limited := suppliertypes.AllowedServices(*msg.Stake, params)
	if !limited {
		return nil
	}

	numServices := uint64(len(msg.Services))
	if numServices == 0 {
		sharedParams := k.sharedKeeper.GetParams(ctx)
		currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
		nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, currentHeight)
		for _, config := range supplier.ServiceConfigHistory {
			if suppliertypes.IsCountedServiceConfig(config, nextSessionStartHeight) {
				numServices++
			}
		}
	}

	if numServices <= allowed {
		return nil
	}

	return suppliertypes.ErrSupplierTooManyServices.Wrapf(
		"%d services require %s; supplier stake is %s",
		numServices, suppliertypes.RequiredStake(numServices, params), msg.Stake,
	)
}
```

- [ ] **Step 5: Run tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/... ./x/migration/...`
Expected: PASS, including every pre-existing stake test (the cap is disabled by default).

- [ ] **Step 6: Lint and commit**

```bash
make go_lint
git add x/supplier/keeper x/migration/keeper
git commit -m "[Supplier] Check the service cap when staking"
```

---

### Task 4: `EnforceServiceCap`

**Files:**
- Create: `x/supplier/keeper/service_cap.go`
- Modify: `x/session/keeper/session_hydrator.go` (`getServiceConfigsSuppliers`)
- Test: `x/supplier/keeper/service_cap_test.go`

**Interfaces:**
- Consumes: `AllowedServices`, `IsCountedServiceConfig`, `RankServiceConfigs` (Task 2); `ServiceCapReason`, `EventSupplierServicesCapped` (Task 1); keeper `getSupplierServiceConfigUpdates`, `SetAndIndexDehydratedSupplier`.
- Produces: `func (k Keeper) EnforceServiceCap(ctx context.Context, supplier *sharedtypes.Supplier, reason suppliertypes.ServiceCapReason) (deactivatedServiceIds []string)`.

- [ ] **Step 1: Write the failing tests**

`x/supplier/keeper/service_cap_test.go` (`package keeper_test`), reusing `enableServiceCap` and `multiServiceStakeMsg` from Task 3:

```go
package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeThenLower stakes 3 services in the given order with enough stake for 3,
// then lowers the stored stake to cover `covered` services (as a slash would).
func stakeThenLower(t *testing.T, covered int64, serviceIds ...string) (keepertest.SupplierModuleKeepers, cosmostypes.Context, sharedtypes.Supplier) {
	t.Helper()
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+int64(len(serviceIds)-1)*capPrice, serviceIds...)
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	supplier, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	supplier.Stake = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake + (covered-1)*capPrice)}
	k.SetDehydratedSupplier(ctx, supplier)
	return k, sdkCtx, supplier
}

func countedServiceIds(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, operatorAddr string) []string {
	t.Helper()
	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	ids := make([]string, 0)
	for _, config := range supplier.ServiceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, next) {
			ids = append(ids, config.Service.ServiceId)
		}
	}
	return ids
}

func TestEnforceServiceCap_KeepsOperatorOrderThroughTheStore(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH)
	require.Equal(t, []string{"base"}, deactivated)
	require.ElementsMatch(t, []string{"sol", "eth"}, countedServiceIds(t, k, ctx, supplier.OperatorAddress))

	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx.EventManager().Events())
	require.Len(t, events, 1)
	require.Equal(t, []string{"base"}, events[0].GetDeactivatedServiceIds())
	require.Equal(t, uint64(2), events[0].GetAllowedServices())
	require.Equal(t, next, events[0].GetEffectiveHeight())
	require.Equal(t, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH, events[0].GetReason())
}

func TestEnforceServiceCap_FallbackOrderForEmptyPriority(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	supplier.ServicePriority = nil // as for a supplier staked before the field existed
	k.SetDehydratedSupplier(ctx, supplier)

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	require.Equal(t, []string{"sol"}, deactivated) // service ID order: base, eth, sol
}

func TestEnforceServiceCap_IdempotentAndUnderCapIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	require.Len(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH), 1)
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))

	k2, ctx2, underCap := stakeThenLower(t, 3, "a", "b", "c")
	require.Empty(t, k2.EnforceServiceCap(ctx2, &underCap, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx2.EventManager().Events()))
}

func TestEnforceServiceCap_NeverUnbondsAndSkipsUnbonding(t *testing.T) {
	// Stake below min stake: left to the unbonding path, nothing deactivated.
	k, ctx, supplier := stakeThenLower(t, 0, "sol", "eth")
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	stored, _ := k.GetDehydratedSupplier(ctx, supplier.OperatorAddress)
	require.False(t, stored.IsUnbonding())

	// Unbonding supplier: skipped.
	k2, ctx2, unbonding := stakeThenLower(t, 1, "sol", "eth")
	unbonding.UnstakeSessionEndHeight = 10
	require.Empty(t, k2.EnforceServiceCap(ctx2, &unbonding, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
}

func TestEnforceServiceCap_ZeroServicesUntouched(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	})
	require.NoError(t, err)

	supplier, _ := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.Empty(t, k.EnforceServiceCap(sdkCtx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events()))
}

func TestEnforceServiceCap_DisabledIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 1, "sol", "eth", "base")
	params := k.GetParams(ctx)
	params.StakePerAdditionalService = nil
	require.NoError(t, k.SetParams(ctx, params))
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
}
```

Note: `stakeThenLower(t, 0, ...)` sets stake to `capMinStake - capPrice`, which is below min stake.

- [ ] **Step 2: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/keeper/ -run TestEnforceServiceCap`
Expected: FAIL, `k.EnforceServiceCap undefined`.

- [ ] **Step 3: Implement**

`x/supplier/keeper/service_cap.go`:

```go
package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// EnforceServiceCap schedules deactivation, at the next session start, of the
// supplier's services beyond what its stake covers, keeping services in
// service_priority order. It never unbonds a supplier: a stake below min_stake
// is left to the existing unbonding path. It never panics or returns an error.
//
// supplier must be the caller's live copy (in settlement, the settlement
// context's cached pointer), since callers may write it again afterwards.
func (k Keeper) EnforceServiceCap(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	reason suppliertypes.ServiceCapReason,
) (deactivatedServiceIds []string) {
	if supplier.IsUnbonding() || supplier.Stake == nil {
		return nil
	}

	allowed, limited := suppliertypes.AllowedServices(*supplier.Stake, k.GetParams(ctx))
	if !limited || allowed == 0 {
		return nil
	}

	sharedParams := k.sharedKeeper.GetParams(ctx)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())

	serviceConfigHistory := k.getSupplierServiceConfigUpdates(ctx, supplier.OperatorAddress, "")
	counted := make([]*sharedtypes.ServiceConfigUpdate, 0, len(serviceConfigHistory))
	for _, config := range serviceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, nextSessionStartHeight) {
			counted = append(counted, config)
		}
	}
	if uint64(len(counted)) <= allowed {
		return nil
	}

	// Ranked holds the same pointers as serviceConfigHistory.
	ranked := suppliertypes.RankServiceConfigs(counted, supplier.ServicePriority)
	for _, config := range ranked[allowed:] {
		config.DeactivationHeight = nextSessionStartHeight
		deactivatedServiceIds = append(deactivatedServiceIds, config.Service.ServiceId)
	}

	supplierToIndex := *supplier
	supplierToIndex.ServiceConfigHistory = serviceConfigHistory
	k.SetAndIndexDehydratedSupplier(ctx, supplierToIndex)

	event := &suppliertypes.EventSupplierServicesCapped{
		OperatorAddress:       supplier.OperatorAddress,
		OwnerAddress:          supplier.OwnerAddress,
		Stake:                 supplier.Stake,
		AllowedServices:       allowed,
		DeactivatedServiceIds: deactivatedServiceIds,
		Reason:                reason,
		EffectiveHeight:       nextSessionStartHeight,
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(event); err != nil {
		k.Logger().Error(fmt.Sprintf("failed to emit EventSupplierServicesCapped: %+v; %s", event, err))
	}

	return deactivatedServiceIds
}
```

- [ ] **Step 4: Strip `service_priority` from sessions**

In `x/session/keeper/session_hydrator.go`, `getServiceConfigsSuppliers`, next to `supplier.Services = ...`:

```go
		// Session consumers (gateways, RelayMiners) do not need the priority list.
		supplier.ServicePriority = nil
```

- [ ] **Step 5: Run tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/... ./x/session/...`
Expected: PASS.

- [ ] **Step 6: Lint and commit**

```bash
make go_lint
git add x/supplier/keeper/service_cap.go x/supplier/keeper/service_cap_test.go x/session/keeper/session_hydrator.go
git commit -m "[Supplier] Enforce the service cap at the next session start"
```

---

### Task 5: Enforce after a slash

**Files:**
- Modify: `x/tokenomics/types/expected_keepers.go` (`SupplierKeeper` interface ~line 108)
- Modify: `x/tokenomics/keeper/settle_pending_claims.go` (`slashSupplierStake`, the `else` branch with `SetDehydratedSupplier` ~line 845)
- Test: `x/tokenomics/keeper/settlement_context_test.go`

**Interfaces:**
- Consumes: `EnforceServiceCap` (Task 4).
- Produces: `SupplierKeeper.EnforceServiceCap(ctx context.Context, supplier *sharedtypes.Supplier, reason suppliertypes.ServiceCapReason) []string` on the tokenomics expected keeper.

- [ ] **Step 1: Extend the interface and regenerate mocks**

In `x/tokenomics/types/expected_keepers.go`, `SupplierKeeper` setters:

```go
	EnforceServiceCap(ctx context.Context, supplier *sharedtypes.Supplier, reason suppliertypes.ServiceCapReason) (deactivatedServiceIds []string)
```

Run: `make go_mockgen`
Expected: exit 0.

- [ ] **Step 2: Write the failing tests**

Add to `x/tokenomics/keeper/settlement_context_test.go`, modelled on `TestSettlePendingClaims_SlashBelowMinStake_UnbondsSupplier` (line ~996): reuse its claim, app, session and settlement setup, with two changes.

1. Before settling, re-stake the claim's supplier with services `[claimService, "svc_extra_1", "svc_extra_2"]` (register the two extra services with `s.keepers.SetService`), set supplier params `MinStake = supplierStakeAmt - 2*price`, `MinStakeIncludedServices = 1`, `StakePerAdditionalService = price`, and set `ProofMissingPenalty = price` so one slash removes exactly one price step. Choose `price = supplierStakeAmt / 10`.
2. Assertions after `SettlePendingClaims`:

```go
	// One slash: stake covers 2 of 3 services; the last listed service is deactivated
	// at the next session start and the supplier is NOT unbonding.
	slashed, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.False(t, slashed.IsUnbonding())
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, sessionProofWindowCloseHeight)
	deactivatedAt := make(map[string]int64)
	for _, config := range slashed.ServiceConfigHistory {
		if config.DeactivationHeight != sharedtypes.NoDeactivationHeight {
			deactivatedAt[config.Service.ServiceId] = config.DeactivationHeight
		}
	}
	require.Equal(t, map[string]int64{"svc_extra_2": next}, deactivatedAt)

	capped := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events())
	require.Len(t, capped, 1)
	require.Equal(t, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH, capped[0].GetReason())

	// The claimed session is unaffected: the supplier is still in it.
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sessionProofWindowCloseHeight)
	require.Contains(t, getSessionSuppliers(sdkCtx, currentSessionStartHeight), supplierOperatorAddr)
```

Name it `TestSettlePendingClaims_Slash_CapsServices`.

Add `TestSettlePendingClaims_TwoSlashesSameBlock_CapsTwice`: same setup, but upsert a second claim for the same supplier in a second session that expires at the same settlement height (use the fixture's second session or build one with `testtree.NewClaim` against another app). Assert that after one `SettlePendingClaims` call the stake dropped by `2*price`, both `svc_extra_2` and `svc_extra_1` are deactivated at `next`, and two `EventSupplierServicesCapped` events were emitted. This proves the second slash saw the first one's deactivation and the end-of-settlement flush did not undo the indexes.

Add `TestSettlePendingClaims_Slash_UnderCap_Unchanged`: the same setup as `TestSettlePendingClaims_Slash_CapsServices` but with the supplier staked for only the claim's service; after the slash, assert no service config has a deactivation height and no `EventSupplierServicesCapped` was emitted.

Add `TestSettlePendingClaims_Slash_CapDisabled_Unchanged`: the existing below-min-stake test's setup with the cap at default (price 0) and a penalty that leaves stake above min stake; assert no service config has a deactivation height and no `EventSupplierServicesCapped` was emitted.

- [ ] **Step 3: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/tokenomics/keeper/ -run 'TestSettlementContext|TestSettlePendingClaims_.*Cap'`
(Use the suite runner name from the top of `settlement_context_test.go`, e.g. `-run 'TestSettlePendingClaims/TestSettlePendingClaims_Slash_CapsServices'`.)
Expected: FAIL, nothing deactivated.

- [ ] **Step 4: Call `EnforceServiceCap`**

In `slashSupplierStake`, replace:

```go
	} else {
		// Only update the dehydrated supplier, since the service config will remain unchanged.
		k.supplierKeeper.SetDehydratedSupplier(ctx, *supplierToSlash)
	}
```

with:

```go
	} else {
		k.supplierKeeper.SetDehydratedSupplier(ctx, *supplierToSlash)
		// The lower stake may no longer cover every service: deactivate the excess
		// from the next session on. Uses the cached pointer, which the settlement
		// flush writes back later; the flush does not touch the service config indexes.
		k.supplierKeeper.EnforceServiceCap(ctx, supplierToSlash, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH)
	}
```

- [ ] **Step 5: Run tokenomics and integration tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/tokenomics/... && GOTOOLCHAIN=go1.27.2 go test -tags test,integration ./tests/integration/tokenomics/...`
Expected: PASS.

- [ ] **Step 6: Lint and commit**

```bash
make go_lint
git add x/tokenomics testutil
git commit -m "[Tokenomics] Cap a slashed supplier's services"
```

---

### Task 6: Param-change sweep

**Files:**
- Modify: `x/supplier/types/keys.go` (add `ServiceCapSweepKey`)
- Modify: `x/supplier/keeper/service_cap.go` (sweep functions)
- Modify: `x/supplier/keeper/msg_server_update_param.go`, `x/supplier/keeper/msg_update_params.go`
- Modify: `x/supplier/module/abci.go`
- Test: `x/supplier/keeper/service_cap_sweep_test.go` (new)

**Interfaces:**
- Consumes: `EnforceServiceCap` (Task 4).
- Produces:
  - `const ServiceCapSweepBatchSize = 200` (keeper package)
  - `func (k Keeper) ScheduleServiceCapSweep(ctx context.Context)`
  - `func (k Keeper) IsServiceCapSweepPending(ctx context.Context) bool`
  - `func (k Keeper) EndBlockerServiceCapSweep(ctx context.Context) (numProcessed int)`
  - `func serviceCapParamsChanged(before, after suppliertypes.Params) bool`

- [ ] **Step 1: Write the failing tests**

`x/supplier/keeper/service_cap_sweep_test.go`:

```go
package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeNSuppliers stakes n suppliers at min stake with 2 services each, cap disabled.
func stakeNSuppliers(t *testing.T, n int) (keepertest.SupplierModuleKeepers, cosmostypes.Context, []string) {
	t.Helper()
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	operators := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
		msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake, fmt.Sprintf("a%d", i), fmt.Sprintf("b%d", i))
		_, err := srv.StakeSupplier(ctx, msg)
		require.NoError(t, err)
		operators = append(operators, operatorAddr)
	}
	return k, sdkCtx, operators
}

func updatePrice(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, price int64) {
	t.Helper()
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	coin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, price)
	_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakePerAdditionalService,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &coin},
	})
	require.NoError(t, err)
}

func TestServiceCapSweep_ScheduledByCapParamUpdateOnly(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	// staking_fee is not a cap param.
	fee := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakingFee,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &fee},
	})
	require.NoError(t, err)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	updatePrice(t, k, ctx, capPrice)
	require.True(t, k.IsServiceCapSweepPending(ctx))
}

func TestServiceCapSweep_BulkUpdateSchedulesOnlyOnCapChange(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	params := k.GetParams(ctx)

	fee := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 3)
	params.StakingFee = &fee
	_, err := srv.UpdateParams(ctx, &suppliertypes.MsgUpdateParams{Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(), Params: params})
	require.NoError(t, err)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	params.MinStakeIncludedServices = 5
	_, err = srv.UpdateParams(ctx, &suppliertypes.MsgUpdateParams{Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(), Params: params})
	require.NoError(t, err)
	require.True(t, k.IsServiceCapSweepPending(ctx))
}

func TestServiceCapSweep_ProcessesInBatchesAndClears(t *testing.T) {
	numSuppliers := keeper.ServiceCapSweepBatchSize + 5
	k, ctx, operators := stakeNSuppliers(t, numSuppliers)
	updatePrice(t, k, ctx, capPrice) // min stake now covers 1 of 2 services

	require.Equal(t, keeper.ServiceCapSweepBatchSize, k.EndBlockerServiceCapSweep(ctx))
	require.True(t, k.IsServiceCapSweepPending(ctx))
	require.Equal(t, 5, k.EndBlockerServiceCapSweep(ctx))
	require.False(t, k.IsServiceCapSweepPending(ctx))
	require.Equal(t, 0, k.EndBlockerServiceCapSweep(ctx))

	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 1, operatorAddr)
	}
}

func TestServiceCapSweep_ParamChangeMidSweepRestarts(t *testing.T) {
	numSuppliers := keeper.ServiceCapSweepBatchSize + 5
	k, ctx, operators := stakeNSuppliers(t, numSuppliers)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	setIncluded := func(included uint64) {
		_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
			Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			Name:      suppliertypes.ParamMinStakeIncludedServices,
			AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: included},
		})
		require.NoError(t, err)
	}

	// Loose cap first: min stake covers both services, so the first batch changes nothing.
	setIncluded(2)
	updatePrice(t, k, ctx, capPrice)
	require.Equal(t, keeper.ServiceCapSweepBatchSize, k.EndBlockerServiceCapSweep(ctx))
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 2)
	}

	// Tighten mid-sweep: the cursor resets, so the suppliers already visited are
	// visited again with the new value.
	setIncluded(1)
	processed := 0
	for k.IsServiceCapSweepPending(ctx) {
		processed += k.EndBlockerServiceCapSweep(ctx)
	}
	require.Equal(t, numSuppliers, processed)
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 1, operatorAddr)
	}
}

func TestServiceCapSweep_DisabledCapClearsWithoutWork(t *testing.T) {
	k, ctx, operators := stakeNSuppliers(t, 3)
	updatePrice(t, k, ctx, capPrice)
	updatePrice(t, k, ctx, 0) // disabled again before the sweep ran
	require.Equal(t, 0, k.EndBlockerServiceCapSweep(ctx))
	require.False(t, k.IsServiceCapSweepPending(ctx))
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 2)
	}
}

func TestServiceCapSweep_SkipsZeroServiceAndUnbondingSuppliers(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, zeroServicesOperator := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: zeroServicesOperator,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	})
	require.NoError(t, err)

	updatePrice(t, k, ctx, capPrice)
	for k.IsServiceCapSweepPending(ctx) {
		k.EndBlockerServiceCapSweep(ctx)
	}
	stored, found := k.GetDehydratedSupplier(ctx, zeroServicesOperator)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
}
```

- [ ] **Step 2: Run to see them fail**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/keeper/ -run TestServiceCapSweep`
Expected: FAIL, undefined sweep functions.

- [ ] **Step 3: Add the store key**

`x/supplier/types/keys.go`, after `ParamsKey`:

```go
// ServiceCapSweepKey marks a pending service cap sweep. Its value is the store key
// of the last supplier processed; empty means the sweep starts from the beginning.
var ServiceCapSweepKey = []byte("ServiceCapSweep/cursor")
```

- [ ] **Step 4: Implement the sweep**

Append to `x/supplier/keeper/service_cap.go` (add imports `bytes`, `storetypes "cosmossdk.io/store/types"`, `"cosmossdk.io/store/prefix"`, `"github.com/cosmos/cosmos-sdk/runtime"`):

```go
// ServiceCapSweepBatchSize bounds the suppliers visited per block by the sweep.
// ponytail: fixed batch; tune from a mainnet-state measurement before the cap is enabled.
const ServiceCapSweepBatchSize = 200

// ScheduleServiceCapSweep starts (or restarts) a sweep over all suppliers.
// Call it whenever min_stake, min_stake_included_services or
// stake_per_additional_service changes, including from upgrade handlers.
func (k Keeper) ScheduleServiceCapSweep(ctx context.Context) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store.Set(suppliertypes.ServiceCapSweepKey, []byte{})
}

// IsServiceCapSweepPending reports whether a sweep is in progress.
func (k Keeper) IsServiceCapSweepPending(ctx context.Context) bool {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	return store.Has(suppliertypes.ServiceCapSweepKey)
}

// EndBlockerServiceCapSweep enforces the service cap on up to
// ServiceCapSweepBatchSize suppliers, in store key order, continuing from the
// cursor. It clears the sweep when the supplier store is exhausted or the cap is
// disabled. It never returns an error: an error from an EndBlocker halts the chain.
func (k Keeper) EndBlockerServiceCapSweep(ctx context.Context) (numProcessed int) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	cursor := store.Get(suppliertypes.ServiceCapSweepKey)
	if cursor == nil {
		return 0
	}

	if !k.GetParams(ctx).ServiceCapEnabled() {
		store.Delete(suppliertypes.ServiceCapSweepKey)
		return 0
	}

	// Collect the batch first: no writes while the supplier store iterator is open.
	supplierStore := prefix.NewStore(store, suppliertypes.KeyPrefix(suppliertypes.SupplierOperatorKeyPrefix))
	var start []byte
	if len(cursor) > 0 {
		start = append(bytes.Clone(cursor), 0x00)
	}
	iterator := supplierStore.Iterator(start, nil)
	batch := make([]sharedtypes.Supplier, 0, ServiceCapSweepBatchSize)
	var lastKey []byte
	for ; iterator.Valid() && len(batch) < ServiceCapSweepBatchSize; iterator.Next() {
		var supplier sharedtypes.Supplier
		if err := k.cdc.Unmarshal(iterator.Value(), &supplier); err != nil {
			k.Logger().Error(fmt.Sprintf("service cap sweep: skipping undecodable supplier at key %x: %v", iterator.Key(), err))
			lastKey = bytes.Clone(iterator.Key())
			continue
		}
		batch = append(batch, supplier)
		lastKey = bytes.Clone(iterator.Key())
	}
	exhausted := !iterator.Valid()
	iterator.Close()

	for i := range batch {
		k.EnforceServiceCap(ctx, &batch[i], suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	}

	if exhausted {
		store.Delete(suppliertypes.ServiceCapSweepKey)
	} else {
		store.Set(suppliertypes.ServiceCapSweepKey, lastKey)
	}

	return len(batch)
}

// serviceCapParamsChanged reports whether any param that sets the cap changed.
func serviceCapParamsChanged(before, after suppliertypes.Params) bool {
	return !coinsEqual(before.MinStake, after.MinStake) ||
		before.MinStakeIncludedServices != after.MinStakeIncludedServices ||
		!coinsEqual(before.StakePerAdditionalService, after.StakePerAdditionalService)
}

func coinsEqual(a, b *cosmostypes.Coin) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.IsEqual(*b)
}
```

Note: the `exhausted` test is evaluated after the loop stops. When the loop stopped because the batch filled, `iterator.Valid()` reports whether more suppliers follow; when it stopped because the store ended, it is false. A store with exactly `ServiceCapSweepBatchSize` remaining suppliers takes one extra block that processes 0 and clears the flag; the batch test above asserts this through its third call returning 0.

`Coin.IsEqual` exists in SDK v0.53.7 and compares denom and amount.

- [ ] **Step 5: Schedule from param updates**

`x/supplier/keeper/msg_server_update_param.go`: capture `paramsBefore := k.GetParams(ctx)` right after `params := k.GetParams(ctx)` (it is a value copy, but `MinStake`/`StakePerAdditionalService` are pointers replaced, not mutated, by the switch, so the copy stays valid). After the successful `k.SetParams(ctx, params)`:

```go
	if serviceCapParamsChanged(paramsBefore, params) {
		k.ScheduleServiceCapSweep(ctx)
	}
```

`x/supplier/keeper/msg_update_params.go`: capture `paramsBefore := k.GetParams(ctx)` before `k.SetParams(ctx, msg.Params)`, and after it succeeds:

```go
	if serviceCapParamsChanged(paramsBefore, msg.Params) {
		k.ScheduleServiceCapSweep(ctx)
	}
```

- [ ] **Step 6: Call from the EndBlocker**

`x/supplier/module/abci.go`, `EndBlocker`, after the prune block and before `return nil`:

```go
	numSuppliersSwept := k.EndBlockerServiceCapSweep(ctx)
	if numSuppliersSwept > 0 {
		logger.Info(fmt.Sprintf("service cap sweep visited %d suppliers", numSuppliersSwept))
	}
```

- [ ] **Step 7: Run tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./x/supplier/...`
Expected: PASS.

- [ ] **Step 8: Lint and commit**

```bash
make go_lint
git add x/supplier
git commit -m "[Supplier] Sweep suppliers after a service cap param change"
```

---

### Task 7: Upgrade handler, E2E params, docs

**Files:**
- Modify: `app/upgrades/vNEXT.go`
- Create: `app/upgrades/vNEXT_test.go`
- Modify: `e2e/tests/parse_params_test.go` (`newSupplierMsgUpdateParams`, `newSupplierMsgUpdateParam`), `e2e/tests/update_params_test.go` (supplier assertion ~line 522), `e2e/tests/update_params.feature`
- Modify: `docusaurus/docs/1_operate/3_configs/3_supplier_staking_config.md`

**Interfaces:**
- Consumes: Task 1 params and defaults.

- [ ] **Step 1: Write the failing upgrade test**

Follow the pattern of `app/upgrades/v0.1.35_test.go` (`TestUpgrade_0_1_35_SeedsOverservicingBonusMultiplier`): it exercises the param-seeding function directly with a keeper from `testkeeper`. Extract the seeding into a function so it can be tested without a full `*keepers.Keepers`.

`app/upgrades/vNEXT_test.go`:

```go
package upgrades

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestUpgrade_NEXT_SeedsServiceCapParams(t *testing.T) {
	k, ctx := testkeeper.SupplierKeeper(t)

	// Simulate a pre-upgrade chain: params without the new fields.
	params := k.GetParams(ctx)
	params.MinStakeIncludedServices = 0
	params.StakePerAdditionalService = nil
	require.NoError(t, k.SetParams(ctx, params))

	logger := cosmostypes.UnwrapSDKContext(ctx).Logger()
	require.NoError(t, seedServiceCapParams(ctx, logger, *k.Keeper))

	seeded := k.GetParams(ctx)
	require.Equal(t, uint64(1), seeded.MinStakeIncludedServices)
	require.True(t, seeded.StakePerAdditionalService.IsZero())
	require.False(t, seeded.ServiceCapEnabled())
	require.NoError(t, seeded.Validate())
	require.False(t, k.IsServiceCapSweepPending(ctx))
}
```

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./app/upgrades/ -run TestUpgrade_NEXT_SeedsServiceCapParams`
Expected: FAIL, `undefined: seedServiceCapParams`.

- [ ] **Step 2: Implement the handler change**

In `app/upgrades/vNEXT.go`, add the function (imports: `cosmoslog "cosmossdk.io/log"`, `supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"`, `suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"`):

```go
// seedServiceCapParams writes the service cap params so they pass Validate().
// The cap ships disabled (price 0): behavior is unchanged until governance sets a price.
// No sweep is scheduled, since nothing can be over a disabled cap.
func seedServiceCapParams(ctx context.Context, logger cosmoslog.Logger, supplierKeeper supplierkeeper.Keeper) error {
	params := supplierKeeper.GetParams(ctx)
	params.MinStakeIncludedServices = suppliertypes.DefaultMinStakeIncludedServices
	price := suppliertypes.DefaultStakePerAdditionalService
	params.StakePerAdditionalService = &price

	if err := params.Validate(); err != nil {
		logger.Error("Invalid supplier params after seeding the service cap", "error", err)
		return err
	}
	if err := supplierKeeper.SetParams(ctx, params); err != nil {
		logger.Error("Failed to set supplier params", "error", err)
		return err
	}

	logger.Info("Seeded service cap params (disabled)", "params", params)
	return nil
}
```

Call it inside the returned handler, before `return vm, nil`:

```go
			if err := seedServiceCapParams(ctx, logger, keepers.SupplierKeeper); err != nil {
				return vm, err
			}
```

`keepers.SupplierKeeper` is a value (`suppliermodulekeeper.Keeper`, `app/keepers/types.go:81`), so `seedServiceCapParams` takes the keeper by value.

Extend the `Upgrade_NEXT` doc comment with:

```go
//   - Supplier service cap (x/supplier). CONSENSUS-BREAKING (state): two new params,
//     min_stake_included_services = 1 and stake_per_additional_service = 0upokt
//     (cap disabled), a new Supplier.service_priority field written by every stake
//     message that lists services, a slash-path call that is a no-op while the cap
//     is disabled, and a param-change sweep. Behavior is unchanged until governance
//     sets a price. An upgrade handler that later changes min_stake,
//     min_stake_included_services or stake_per_additional_service with SetParams
//     MUST also call SupplierKeeper.ScheduleServiceCapSweep.
```

Add to the "Gas changes" list:

```go
//   - MsgStakeSupplier: reads the supplier params once more (service cap check).
//   - Supplier records grow by service_priority after the supplier restakes, so
//     reads of those records (claims, proofs, settlement) cost slightly more gas.
```

Add to the "Event changes" list:

```go
//   - New EventSupplierServicesCapped.
```

- [ ] **Step 3: Run upgrade tests**

Run: `GOTOOLCHAIN=go1.27.2 go test -tags test ./app/...`
Expected: PASS, including `app/upgrades_registration_test.go`.

- [ ] **Step 4: E2E param plumbing**

`e2e/tests/parse_params_test.go`, `newSupplierMsgUpdateParams` switch:

```go
		case suppliertypes.ParamMinStakeIncludedServices:
			msgUpdateParams.Params.MinStakeIncludedServices = paramValue.value.(uint64)
		case suppliertypes.ParamStakePerAdditionalService:
			msgUpdateParams.Params.StakePerAdditionalService = paramValue.value.(*cosmostypes.Coin)
```

`uint64` rows are parsed as `uint64` (`parse_params_test.go`, `case "uint64"`).

`newSupplierMsgUpdateParam` switch:

```go
	case "uint64":
		msg = proto.Message(&suppliertypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &suppliertypes.MsgUpdateParam_AsUint64{
				AsUint64: param.value.(uint64),
			},
		})
```

`e2e/tests/update_params_test.go`, supplier case: read the two new params from `s.expectedModuleParams` when present, else defaults, and include them in the expected `suppliertypes.Params`:

```go
		includedServices := suppliertypes.DefaultMinStakeIncludedServices
		if param, ok := s.expectedModuleParams[moduleName][suppliertypes.ParamMinStakeIncludedServices]; ok {
			includedServices = param.value.(uint64)
		}
		pricePerService := &suppliertypes.DefaultStakePerAdditionalService
		if param, ok := s.expectedModuleParams[moduleName][suppliertypes.ParamStakePerAdditionalService]; ok {
			pricePerService = param.value.(*cosmostypes.Coin)
		}
```

and set `MinStakeIncludedServices: includedServices, StakePerAdditionalService: pricePerService` in the asserted params. If `min_stake` / `staking_fee` are also absent from a scenario's table, apply the same "default when absent" pattern to them so a scenario that sets only the new params does not panic.

`e2e/tests/update_params.feature`, append:

```gherkin
    # The supplier service cap ships disabled (price 0); governance enables it.
    Scenario: The supplier service cap can be set by governance
        Given the user has the pocketd binary installed
        When the "supplier" module parameters are set as follows
            | name                         | value   | type   |
            | min_stake_included_services  | 35      | uint64 |
            | stake_per_additional_service | 5000000 | coin   |
        Then all "supplier" module params should be updated

    Scenario: A supplier service cap with zero included services is rejected
        Given the user has the pocketd binary installed
        Then the "supplier" module parameter update is rejected with the error "min_stake_included_services"
            | name                        | value | type   |
            | min_stake_included_services | 0     | uint64 |
```

Check the existing `coin` rows in the feature files for the exact value format (amount only vs `5000000upokt`) and match it. The scenario resets params afterwards if the suite does so for others; follow the file's existing pattern.

Compile the E2E package: `GOTOOLCHAIN=go1.27.2 go vet -tags e2e ./e2e/tests/`
Expected: exit 0.

- [ ] **Step 5: Operator docs**

In `docusaurus/docs/1_operate/3_configs/3_supplier_staking_config.md`, under the `services` section, add:

```markdown
:::note Service order is service priority

When the network's service cap is enabled, your stake covers a limited number of
services: `min_stake` covers `min_stake_included_services`, and each additional
`stake_per_additional_service` covers one more. If your stake stops covering all
your services (after a slash, or when governance tightens the cap), the services
**last** in this list are deactivated at the next session start, and an
`EventSupplierServicesCapped` event lists them. Restake with more stake to restore
them, or restake with a shorter list to choose which ones you keep.

Suppliers staked before this feature keep their services in service ID order until
they next restake with services.
:::
```

- [ ] **Step 6: Full verification**

Run, in order:

```bash
GOTOOLCHAIN=go1.27.2 go build ./...
make go_lint
GOTOOLCHAIN=go1.27.2 go test -count=1 -timeout 30m -tags test,integration ./...
```

Expected: build OK, lint 0 issues, all packages `ok`.

- [ ] **Step 7: Commit**

```bash
git add app/upgrades e2e docusaurus
git commit -m "[Upgrade] Seed the supplier service cap params disabled"
```

- [ ] **Step 8: Manual gates (not code; record results in the PR description)**

1. LocalNet E2E: `make localnet_up`, then `make test_e2e_params` and `make test_e2e_supplier`. Expected: green.
2. Local upgrade test from v0.1.35 (same procedure as the v0.1.35 local upgrade test): after the upgrade height, `pocketd q supplier params` shows `min_stake_included_services: 1` and `stake_per_additional_service: 0upokt`; staking and settlement behave as before.

---

## Deferred (spec "Rollout", not part of this release's code)

These must be done before the first governance enablement, not before v0.1.36:

- Measure one sweep batch on a mainnet database copy and adjust `ServiceCapSweepBatchSize`.
- Impact script listing, for proposed values, each supplier's allowed count, current count and the services it would lose.
- `pocketd` warning when listed services exceed what the stake covers.
- pocketdex indexing of `EventSupplierServicesCapped`.
