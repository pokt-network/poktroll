---
sidebar_position: 5
title: Adding Onchain Module Parameters
---

# Adding Onchain Module Parameters <!-- omit in toc -->

- [Step-by-Step Instructions](#step-by-step-instructions)
  - [0. If the Module Doesn't Already Support a `MsgUpdateParam` Message](#0-if-the-module-doesnt-already-support-a-msgupdateparam-message)
    - [0.1. Scaffold the `MsgUpdateParam` Message](#01-scaffold-the-msgupdateparam-message)
    - [0.2. Update `MsgUpdateParam` and `MsgUpdateParamResponse` Fields](#02-update-msgupdateparam-and-msgupdateparamresponse-fields)
    - [0.3 Comment Out AutoCLI](#03-comment-out-autocli)
    - [0.4. Update the DAO Genesis Authorizations JSON File](#04-update-the-dao-genesis-authorizations-json-file)
    - [0.5 Update the `NewMsgUpdateParam` Constructor and `MsgUpdateParam#ValidateBasic()`](#05-update-the-newmsgupdateparam-constructor-and-msgupdateparamvalidatebasic)
    - [0.6 Update the Module's `msgServer#UpdateParam()` Handler](#06-update-the-modules-msgserverupdateparam-handler)
    - [0.7 Update Module's Params Test Suite `ModuleParamConfig`](#07-update-modules-params-test-suite-moduleparamconfig)
  - [1. Define the Parameter in the Protocol Buffers File](#1-define-the-parameter-in-the-protocol-buffers-file)
  - [2. Update the Default Parameter Values](#2-update-the-default-parameter-values)
    - [2.1 Go Source Defaults](#21-go-source-defaults)
    - [2.2 Genesis Configuration Parameter Defaults](#22-genesis-configuration-parameter-defaults)
  - [3. Parameter Validation](#3-parameter-validation)
    - [3.1 Define a Validation Function](#31-define-a-validation-function)
    - [3.2 Call it in the `Params#Validate()`](#32-call-it-in-the-paramsvalidate)
    - [3.3 Add a `ParamSetPair` to `ParamSetPairs()`](#33-add-a-paramsetpair-to-paramsetpairs)
  - [4. Add Parameter Case to Switch Statements](#4-add-parameter-case-to-switch-statements)
    - [4.1 `MsgUpdateParam#ValidateBasic()`](#41-msgupdateparamvalidatebasic)
    - [4.2 `msgServer#UpdateParam()`](#42-msgserverupdateparam)
  - [5. Update Unit Tests](#5-update-unit-tests)
    - [5.1 Parameter Validation Tests](#51-parameter-validation-tests)
    - [5.2 Parameter Update Tests](#52-parameter-update-tests)
  - [6. Update the Parameter Integration Tests](#6-update-the-parameter-integration-tests)
    - [6.1 Add a valid param](#61-add-a-valid-param)
    - [6.2 Check for `as_<type>` on `MsgUpdateParam`](#62-check-for-as_type-on-msgupdateparam)
    - [6.3 Update the module's `ModuleParamConfig`](#63-update-the-modules-moduleparamconfig)
  - [7. Update the Makefile and Supporting JSON Files](#7-update-the-makefile-and-supporting-json-files)
    - [7.1 Update the Makefile](#71-update-the-makefile)
    - [7.2 Create a new JSON File for the Individual Parameter Update](#72-create-a-new-json-file-for-the-individual-parameter-update)
    - [7.3 Update the JSON File for Updating All Parameters for the Module](#73-update-the-json-file-for-updating-all-parameters-for-the-module)

Adding a new onchain module parameter involves multiple steps to ensure that the
parameter is properly integrated into the system. This guide will walk you through
the process using a generic approach, illustrated by adding a parameter to the `proof` module.

See [pokt-network/poktroll#595](https://github.com/pokt-network/poktroll/pull/595) for a real-world example.

:::note

TODO_POST_MAINNET(@bryanchriswhite): Once the next version of `ignite` is out, leverage:
https://github.com/ignite/cli/issues/3684#issuecomment-2299796210

:::

## Step-by-Step Instructions

:::warning
The steps outlined below follow **the same example** where:

- **Module name**: `examplemod`
- **New parameter name**: `new_parameter`
- **Default value**: `int64(42)`

When following these steps, be sure to substitute these example values with your own!
:::

:::tip
At any point, you can always run `go test ./x/examplemod/...` to check whether everything is working or locate outstanding necessary changes.
:::

### 0. If the Module Doesn't Already Support a `MsgUpdateParam` Message

In order to support **individual parameter updates**, the module MUST have a `MsgUpdateParam` message.
If the module doesn't already support this message, it will need to be added.

#### 0.1. Scaffold the `MsgUpdateParam` Message

Use `ignite` to scaffold a new `MsgUpdateParam` message for the module.
Additional flags are used for convenience:

```bash
ignite scaffold message update-param --module examplemod --signer authority name as_type --response params
```

Try running `make proto_clean_pulsar` if you experience errors like these:

```bash
✘ Error while running command /home/bwhite/go/bin/buf generate /tmp/proto-sdk2893110128/cosmos/upgrade/v1beta1/tx.proto ...: {..."message":"import \"gogoproto/gogo.proto\": file does not exist"}
: exit status 100
```

```bash
✘ Error while running command go mod tidy: go:
...
go: github.com/pokt-network/poktroll/api/poktroll/examplemod imports
        cosmossdk.io/api/poktroll/shared: module cosmossdk.io/api@latest found (v0.7.6), but does not contain package cosmossdk.io/api/poktroll/shared
```

```bash
✘ Error while running command /home/bwhite/go/bin/buf dep update /home/bwhite/Projects/pokt/poktroll/proto: Failure: decode proto/buf.lock: no digest specified for module buf.build/cosmos/ics23
: exit status 1
```

#### 0.2. Update `MsgUpdateParam` and `MsgUpdateParamResponse` Fields

Update the `MsgUpdateParam` message fields in the module's `tx.proto` file (e.g. `proto/poktroll/examplemod/tx.proto`) to include the following comments and protobuf options:

```diff
+ // MsgUpdateParam is the Msg/UpdateParam request type to update a single param.
  message MsgUpdateParam {
    option (cosmos.msg.v1.signer) = "authority";
-   string authority = 1;
+
+   // authority is the address that controls the module (defaults to x/gov unless overwritten).
+   string authority = 1  [(cosmos_proto.scalar) = "cosmos.AddressString"];
+
    string name      = 2;
-   string asType    = 3;
+   oneof as_type {
+     // Add `as_<type>` fields for each type in this module's Params type; e.g.:
+     // int64 as_int64 = 3 [(gogoproto.jsontag) = "as_int64"];
+     // bytes as_bytes = 4 [(gogoproto.jsontag) = "as_bytes"];
+     // cosmos.base.v1beta1.Coin as_coin = 5 [(gogoproto.jsontag) = "as_coin"];
+   }
  }

  message MsgUpdateParamResponse {
```

Update the `MsgUpdateParamResponse` message field (`params`) in the same `tx.proto` file:

```diff
  message MsgUpdateParamResponse {
-   string params = 1;
+   Params params = 1;
  }
```

#### 0.3 Comment Out AutoCLI

When scaffolding the `MsgUpdateParam` message, generated code is added to `x/examplemod/module/autocli.go`.
Since governance parameters aren't updated via `poktrolld` CLI, comment out these new lines:

```diff
  // ...
  Tx: &autocliv1.ServiceCommandDescriptor{
      Service:              modulev1.Msg_ServiceDesc.ServiceName,
      EnhanceCustomCommand: true, // only required if you want to use the custom command
      RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
          // ...
+         //  {
+         //    RpcMethod:      "UpdateParam",
+         //    Use:            "update-param [name] [as-type]",
+         //    Short:          "Send a update-param tx",
+         /     PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
+         //  },
          // this line is used by ignite scaffolding # autocli/tx
      },
  },
  // ...
```

#### 0.4. Update the DAO Genesis Authorizations JSON File

Add a grant (array element) to `tools/scripts/authz/dao_genesis_authorizations.json` with the `authorization.msg` typeURL for this module's `MsgUpdateType`:

```diff
+ {
+   "granter": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
+   "grantee": "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
+   "authorization": {
+     "@type": "\/cosmos.authz.v1beta1.GenericAuthorization",
+     "msg": "\/poktroll.examplemod.MsgUpdateParam" // Replace examplemod with the module name
+   },
+   "expiration": "2500-01-01T00:00:00Z"
+ },
```

#### 0.5 Update the `NewMsgUpdateParam` Constructor and `MsgUpdateParam#ValidateBasic()`

Prepare `x/examplemod/types/message_update_param.go` to handle message construction and parameter validations by type:

```diff
- func NewMsgUpdateParam(authority string, name string, asType string) *MsgUpdateParam {
+ func NewMsgUpdateParam(authority string, name string, asType any) (*MsgUpdateParam, error) {
+   var asTypeIface isMsgUpdateParam_AsType
+
+   switch t := asType.(type) {
+   default:
+     return nil, ExamplemodParamInvalid.Wrapf("unexpected param value type: %T", asType)
+   }
+
    return &MsgUpdateParam{
      Authority: authority,
      Name: name,
-     AsType: asType,
-   }
+     AsType: asTypeIface,
+   }, nil
  }

  func (msg *MsgUpdateParam) ValidateBasic() error {
    _, err := cosmostypes.AccAddressFromBech32(msg.Authority)
    if err != nil {
      return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
    }
-
-   return nil
+
+   // Parameter value MUST NOT be nil.
+   if msg.AsType == nil {
+     return ErrExamplemodParamInvalid.Wrap("missing param AsType")
+   }
+
+   // Parameter name MUST be supported by this module.
+   switch msg.Name {
+   default:
+     return ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
+   }
  }
```

#### 0.6 Update the Module's `msgServer#UpdateParam()` Handler

Prepare `x/examplemod/keeper/msg_server_update_param.go` to handle parameter updates by type:

```diff
- func (k msgServer) UpdateParam(goCtx context.Context, msg *examplemodtypes.MsgUpdateParam) (*examplemodtypes.MsgUpdateParamResponse, error) {
-   ctx := sdk.UnwrapSDKContext(goCtx)
-
-   // TODO: Handling the message
-   _ = ctx
-
+ // UpdateParam updates a single parameter in the proof module and returns
+ // all active parameters.
+ func (k msgServer) UpdateParam(ctx context.Context, msg *examplemodtypes.MsgUpdateParam) (*examplemodtypes.MsgUpdateParamResponse, error) {
+   logger := k.logger.With(
+     "method", "UpdateParam",
+     "param_name", msg.Name,
+   )
+
+   if err := msg.ValidateBasic(); err != nil {
+     return nil, status.Error(codes.InvalidArgument, err.Error())
+   }
+
+   if k.GetAuthority() != msg.Authority {
+     return nil, status.Error(
+       codes.PermissionDenied,
+       examplemodtypes.ErrExamplemodInvalidSigner.Wrapf(
+         "invalid authority; expected %s, got %s",
+         k.GetAuthority(), msg.Authority,
+       ).Error(),
+     )
+   }
+
+   params := k.GetParams(ctx)
+
+   switch msg.Name {
+   default:
+     return nil, status.Error(
+       codes.InvalidArgument,
+       examplemodtypes.ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
+     )
+   }
+
+   // Perform a global validation on all params, which includes the updated param.
+   // This is needed to ensure that the updated param is valid in the context of all other params.
+   if err := params.Validate(); err != nil {
+     return nil, status.Error(codes.InvalidArgument, err.Error())
+   }
+
+   if err := k.SetParams(ctx, params); err != nil {
+     err = fmt.Errorf("unable to set params: %w", err)
+     logger.Error(err.Error())
+     return nil, status.Error(codes.Internal, err.Error())
+   }
+
+   updatedParams := k.GetParams(ctx)
+
+   return &types.MsgUpdateParamResponse{
+     Params: &updatedParams,
+   }, nil
  }
```

#### 0.7 Update Module's Params Test Suite `ModuleParamConfig`

Add `MsgUpdateParam` & `MsgUpdateParamResponse` to the module's `ModuleParamConfig#ParamsMsg` in `testutil/integration/suites/param_config.go`:

```diff
  ExamplemodModuleParamConfig = ModuleParamConfig{
    ParamsMsgs: ModuleParamsMessages{
      MsgUpdateParams:         gatewaytypes.MsgUpdateParams{},
      MsgUpdateParamsResponse: gatewaytypes.MsgUpdateParamsResponse{},
+     MsgUpdateParam:          gatewaytypes.MsgUpdateParam{},
+     MsgUpdateParamResponse:  gatewaytypes.MsgUpdateParamResponse{},
      QueryParamsRequest:      gatewaytypes.QueryParamsRequest{},
      QueryParamsResponse:     gatewaytypes.QueryParamsResponse{},
    },
    // ...
  }
```

### 1. Define the Parameter in the Protocol Buffers File

Define the new parameter in the module's `params.proto` file (e.g., `proto/poktroll/examplemod/params.proto`):

```diff
  message Params {
    // Other existing parameters...

+   // Description of the new parameter.
+   int64 new_parameter = 3 [(gogoproto.jsontag) = "new_parameter", (gogoproto.moretags) = "yaml:\"new_parameter\""];
  }
```

:::warning
Be sure to update the `gogoproto.jsontag` and `gogoproto.moretags` option values to match the new parameter name!
:::

:::tip
Don't forget to run `make proto_regen` to update generated protobuf go code.
:::

### 2. Update the Default Parameter Values

#### 2.1 Go Source Defaults

In the corresponding Go file (e.g., `x/examplemod/types/params.go`), define the default
value, key, and parameter name for the new parameter and include the default in the
`NewParams` and `DefaultParams` functions:

```diff
  var (
    // Other existing parameter keys, names, and defaults...

+   KeyNewParameter = []byte("NewParameter")
+   ParamNewParameter = "new_parameter"
+   DefaultNewParameter int64 = 42
  )

  func NewParams(
    // Other existing parameters...
+   newParameter int64,
  ) Params {
    return Params{
      // Other existing parameters...
+     NewParameter: newParameter,
    }
  }

  func DefaultParams() Params {
    return NewParams(
      // Other existing default parameters...
+     DefaultNewParameter,
    )
  }
```

#### 2.2 Genesis Configuration Parameter Defaults

Add the new parameter to the genesis configuration file (e.g., `config.yml`):

```diff
  genesis:
    examplemod:
      params:
        # Other existing parameters...

+       new_parameter: 42
```

### 3. Parameter Validation

#### 3.1 Define a Validation Function

Implement a validation function for the new parameter in `x/examplemod/types/params.go`:

```diff
+ // ValidateNewParameter validates the NewParameter param.
+ func ValidateNewParameter(newParamAny any) error {
+   newParam, ok := newParamAny.(int64)
+   if !ok {
+     return ErrExamplemodParamInvalid.Wrapf("invalid parameter type: %T", newParamAny)
+   }
+
+   // Any additional validation...
+
+   return nil
+ }
```

#### 3.2 Call it in the `Params#Validate()`

Integrate the usage of the new `ValidateNewParameter` function in the corresponding
`Params#Validate()` function where this is used:

```diff
  func (params *Params) Validate() error {
    // ...
+   if err := ValidateNewParameter(params.NewParameter); err != nil {
+     return err
+   }
    // ...
  }
```

#### 3.3 Add a `ParamSetPair` to `ParamSetPairs()`

Include a call to `NewParamSetPair()`, passing the parameter's key, value pointer, and validation function in the `ParamSetPairs` function return:

```diff
  func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
    return paramtypes.ParamSetPairs{
      // Other existing param set pairs...

+     paramtypes.NewParamSetPair(
+       KeyNewParameter,
+       &p.NewParameter,
+       ValidateNewParameter,
+     ),
    }
  }
```

### 4. Add Parameter Case to Switch Statements

#### 4.1 `MsgUpdateParam#ValidateBasic()`

Add the parameter type and name (e.g. `ParamNameNewParameter`) to new cases in the switch statements in `NewMsgUpdateParam()` and `MsgUpdateParam#ValidateBasic()` in `x/examplemod/types/message_update_param.go`:

```diff
  func NewMsgUpdateParam(authority string, name string, asType any) (*MsgUpdateParam, error) {
    // ...
    switch t := asType.(type) {
+   case int64:
+     asTypeIface = &MsgUpdateParam_AsCoin{AsInt64: t}
    default:
      return nil, ErrExamplemodParamInvalid.Wrapf("unexpected param value type: %T", asType))
    }
    // ...
  }

+ // ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures:
+ // 1. The parameter name is supported.
+ // 2. The parameter type matches the expected type for a given parameter name.
+ // 3. The parameter value is valid (according to its respective validation function).
  func (msg *MsgUpdateParam) ValidateBasic() error {
    // ...
    switch msg.Name {
+   case ParamNewParameter:
+     if err := genericParamTypeIs[*MsgUpdateParam_AsInt64](msg); err != nil {
+       return err
+     }
+     return ValidateNewParameter(msg.GetAsInt64())
    default:
      return ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
    }
  }
+
+ func genericParamTypeIs[T any](msg *MsgUpdateParam) error {
+   if _, ok := msg.AsType.(T); !ok {
+     return ErrParamInvalid.Wrapf(
+       "invalid type for param %q; expected %T, got %T",
+       msg.Name, *new(T), msg.AsType,
+     )
+   }
+
+   return nil
+ }

```

#### 4.2 `msgServer#UpdateParam()`

Add the parameter name (e.g. `ParamNameNewParameter`) to a new case in the switch statement in `msgServer#UpdateParam()` in `x/examplemod/keeper/msg_server_update_param.go`:

:::warning
Every error return from `msgServer` methods (e.g. `#UpdateParams()`) **SHOULD** be encapsulated in a gRPC status error!
:::

```diff
  // UpdateParam updates a single parameter in the proof module and returns
  // all active parameters.
  func (k msgServer) UpdateParam(
    ctx context.Context,
    msg *examplemodtypes.MsgUpdateParam,
  ) (*examplemodtypes.MsgUpdateParamResponse, error) {
    if err := msg.ValidateBasic(); err != nil {
      return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    // ...
    switch msg.Name {
+   case examplemodtypes.ParamNewParameter:
+     logger = logger.with("param_value", msg.GetAsInt64())
+     params.NewParameter = msg.GetAsInt64()
    default:
      return nil, status.Error(
        codes.InvalidArgument,
        examplemodtypes.ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name).Error(),
      )
    }
    // ...
    if err := params.Validate(); err != nil {
      return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    // ...
  }
```

### 5. Update Unit Tests

#### 5.1 Parameter Validation Tests

Add unit tests which exercise validation of the new parameter(s) in `x/examplemod/keeper/params_test.go`.
Ensure there is a test function for each parameter which covers all cases of invalid input:

```diff
  func TestGetParams(t *testing.T) {
    // ...
  }

+ func TestParams_ValidateNewParameter(t *testing.T) {
+   tests := []struct {
+     desc         string
+     newParameter any
+     expectedErr  error
+   }{
+     {
+       desc: "invalid type",
+       newParameter: "420",
+       expectedErr: examplemodtypes.ErrExamplemodParamInvalid.Wrapf("invalid parameter type: string"),
+     },
+     {
+       desc: "valid NewParameterName",
+       newParameter: int64(420),
+     },
+   }
+
+   for _, test := range tests {
+     t.Run(test.desc, func(t *testing.T) {
+       err := examplemodtypes.ValidateNewParameter(test.newParameter)
+       if test.expectedErr != nil {
+         require.Error(t, err)
+         require.Contains(t, err.Error(), test.expectedErr.Error())
+       } else {
+         require.NoError(t, err)
+       }
+     })
+   }
+ }
```

#### 5.2 Parameter Update Tests

Add test cases to `x/examplemod/keeper/msg_update_params_test.go` to ensure coverage over any invalid parameter combinations.
Add a case for the "minimal params" if some subset of the params are "required".
If one already exist, update it if applicable; e.g.:

```go
+ {
+   desc: "valid: send minimal params", // For parameters which MUST NEVER be their zero value or nil.
+   input: &examplemodtypes.MsgUpdateParams{
+     Authority: k.GetAuthority(),
+     Params: examplemodtypes.Params{
+       NewParameter: 42,
+     },
+   },
+   shouldError: false,
+ },
```

Add a unit test which exercise individually updating the new parameter in `x/examplemod/keeper/msg_server_update_param_test.go`.
This test asserts that updating was successful and that no other parameter was effected:

```go
+ func TestMsgUpdateParam_UpdateNewParameterOnly(t *testing.T) {
+   var expectedNewParameter int64 = 420
+
+   // Set the parameters to their default values
+   k, msgSrv, ctx := setupMsgServer(t)
+   defaultParams := examplemodtypes.DefaultParams()
+   require.NoError(t, k.SetParams(ctx, defaultParams))
+
+   // Ensure the default values are different from the new values we want to set
+   require.NotEqual(t, expectedNewParameter, defaultParams.NewParameter)
+
+   // Update the new parameter
+   updateParamMsg := &examplemodtypes.MsgUpdateParam{
+     Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
+     Name: examplemodtypes.ParamNewParameter,
+     AsType: &examplemodtypes.MsgUpdateParam_AsInt64{AsInt64: expectedNewParameter},
+   }
+   res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
+   require.NoError(t, err)
+   require.Equal(t, expectedNewParameter, res.Params.NewParameter)
+
+   // Ensure the other parameters are unchanged
+   testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(examplemodtypes.KeyNewParameter))
+ }
```

:::warning
If creating `msg_server_update_param_test.go`, be sure to:

1. use the `keeper_test` package (i.e. `package keeper_test`).
2. add the testutil keeper import: `testkeeper "github.com/pokt-network/poktroll/testutil/keeper"`

:::

Update `x/examplemod/types/message_update_param_test.go` to use the new `MsgUpdateParam#AsType` fields.
Start with the following cases and add those which cover all invalid values for the new param (and its `AsType`; e.g. `AsCoin` cannot be nil):

```diff
  func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
    tests := []struct {
-     name string
+     desc string
      msg  MsgUpdateParam
-     err  error
+     expectedErr  error
    }{
      {
-       name: "invalid address",
+       desc: "invalid: authority address invalid",
        msg: MsgUpdateParam{
          Authority:  "invalid_address",
+         Name: "",   // Doesn't matter for this test
+         AsType:     &MsgUpdateParam_AsInt64{AsInt64: 0},
        },
-       err: sdkerrors.ErrInvalidAddress,
+       expectedErr: sdkerrors.ErrInvalidAddress,
+     }, {
+       desc: "invalid: param name incorrect (non-existent)",
+       msg: MsgUpdateParam{
+         Authority: sample.AccAddress(),
+         Name:      "non_existent",
+         AsType:    &MsgUpdateParam_AsInt64{AsInt64: DefaultNewParameter},
+       },
+       expectedErr: ErrExamplemodParamInvalid,
      }, {
-       name: "valid address",
+       desc: "valid: correct address, param name, and type",
        msg: MsgUpdateParam{
          Authority: sample.AccAddress(),
+         Name: ParamNewParameter,
+         AsType: &MsgUpdateParam_AsInt64{AsInt64: DefaultNewParameter},
        },
      },
    }
    // ...
  }
```

### 6. Update the Parameter Integration Tests

Integration tests which cover parameter updates utilize the `ModuleParamConfig`s defined in [`testutil/integration/params/param_configs.go`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/suites/param_configs.go) to dynamically (i.e. using reflection) construct and send parameter update messages in a test environment.
When adding parameters to a module, it is necessary to update that module's `ModuleParamConfig` to include the new parameter, othwerwise it will not be covered by the integration test suite.

#### 6.1 Add a valid param

Update `ModuleParamConfig#ValidParams` to include a valid and non-default value for the new parameter in the module's `tx.proto` file (e.g. `proto/poktroll/examplemod/tx.proto`):

```diff
  ExamplemodModuleParamConfig = ModuleParamConfig{
      // ...
      ValidParams: examplemodtypes.Params{
+         NewParameter: 420,
      },
      // ...
  }
```

#### 6.2 Check for `as_<type>` on `MsgUpdateParam`

Ensure an `as_<type>` field exists on `MsgUpdateParam` corresponding to the type of the new parameter (`proto/poktroll/examplemod/tx.proto`):

```diff
 message MsgUpdateParam {
   ...
   oneof as_type {
-    // Add `as_<type>` fields for each type in this module's Params type; e.g.:
+    int64 as_int64 = 3 [(gogoproto.jsontag) = "as_int64"];
   }
 }
```

#### 6.3 Update the module's `ModuleParamConfig`

Ensure that all available `as_<type>` types for the module are present on the module's `ModuleParamConfig#ParamTypes` field:

```diff
  ExamplemodModuleParamConfig = ModuleParamConfig{
    // ...
    ValidParams: examplemodtypes.Params{},
+   ParamTypes: map[ParamType]any{
+     ParamTypeInt64: examplemodtypes.MsgUpdateParam_AsInt64{},
+   },
    DefaultParams:    examplemodtypes.DefaultParams(),
    // ...
  }
```

### 7. Update the Makefile and Supporting JSON Files

#### 7.1 Update the Makefile

Add a new target in `makefiles/params.mk` to update the new parameter:

```makefile
.PHONY: params_update_examplemod_new_parameter
params_update_examplemod_new_parameter: ## Update the examplemod module new_parameter param
  poktrolld tx authz exec ./tools/scripts/params/examplemod_new_parameter.json $(PARAM_FLAGS)
```

:::warning
Reminder to substitute `examplemod` and `new_parameter` with your module and param names!
:::

#### 7.2 Create a new JSON File for the Individual Parameter Update

Create a new JSON file (e.g., `proof_new_parameter_name.json`) in the tools/scripts/params directory to specify how to update the new parameter:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/poktroll.examplemod.MsgUpdateParam", // Replace module name
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "new_parameter", // Replace new parameter name
        "as_int64": "42" // Replace default value
      }
    ]
  }
}
```

#### 7.3 Update the JSON File for Updating All Parameters for the Module

Add a line to the existing module's `MsgUpdateParam` JSON file (e.g., `proof_all.json`)
with the default value for the new parameter.

```diff
  {
    "body": {
      "messages": [
        {
          "@type": "/poktroll.examplemod.MsgUpdateParams", // Replace module name
          "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
          "params": {
            // Other existing parameters...
+           "new_parameter": "42" // Replace name and default value
          }
        }
      ]
    }
  }
```
