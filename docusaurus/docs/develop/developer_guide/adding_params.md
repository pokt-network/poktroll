---
sidebar_position: 5
title: Adding On-Chain Module Parameters
---

# Adding On-Chain Module Parameters <!-- omit in toc -->

- [Step-by-Step Instructions](#step-by-step-instructions)
  - [0. If the Module Doesn't Already Support a `MsgUpdateParam` Message](#0-if-the-module-doesnt-already-support-a-msgupdateparam-message)
    - [0.1 Scaffold the `MsgUpdateParam` Message](#01-scaffold-the-msgupdateparam-message)
    - [0.2 Update the `MsgUpdateParam` Message Fields](#02-update-the-msgupdateparam-message-fields)
    - [0.3 Comment Out AutoCLI](#03-comment-out-autocli)
    - [0.4. Update the DAO Genesis Authorizations JSON File](#04-update-the-dao-genesis-authorizations-json-file)
    - [0.5 Update the `NewMsgUpdateParam` Constructor and `MsgUpdateParam#ValidateBasic()`](#05-update-the-newmsgupdateparam-constructor-and-msgupdateparamvalidatebasic)
    - [0.6 Update the Module's `msgServer#UpdateParam()` Handler](#06-update-the-modules-msgserverupdateparam-handler)
    - [0.7 Update Module's Params Test Suite `ModuleParamConfig`](#07-update-modules-params-test-suite-moduleparamconfig)
    - [1. Define the Parameter in the Protocol Buffers File](#1-define-the-parameter-in-the-protocol-buffers-file)
    - [2 Update the Parameter Integration Tests](#2-update-the-parameter-integration-tests)
      - [2.1 Add a valid param](#21-add-a-valid-param)
      - [2.2 Check for `as_<type>` on `MsgUpdateParam`](#22-check-for-as_type-on-msgupdateparam)
    - [3. Update the Default Parameter Values](#3-update-the-default-parameter-values)
      - [3.1 Go Source Defaults](#31-go-source-defaults)
      - [3.2 Genesis Configuration Parameter Defaults](#32-genesis-configuration-parameter-defaults)
    - [4. Update the Makefile and Supporting JSON Files](#4-update-the-makefile-and-supporting-json-files)
      - [4.1 Update the Makefile](#41-update-the-makefile)
      - [4.2 Create a new JSON File for the Individual Parameter Update](#42-create-a-new-json-file-for-the-individual-parameter-update)
      - [4.3 Update the JSON File for Updating All Parameters for the Module](#43-update-the-json-file-for-updating-all-parameters-for-the-module)
    - [5. Parameter Validation](#5-parameter-validation)
      - [5.1 New Parameter Validation](#51-new-parameter-validation)
      - [5.2 Parameter Validation in Workflow](#52-parameter-validation-in-workflow)
    - [6. Add the Parameter to `ParamSetPairs()`](#6-add-the-parameter-to-paramsetpairs)
    - [7. Update Unit Tests](#7-update-unit-tests)
      - [7.1 Parameter Validation Tests](#71-parameter-validation-tests)
      - [7.2 Parameter Update Tests](#72-parameter-update-tests)
    - [8. Add Parameter Case to Switch Statements](#8-add-parameter-case-to-switch-statements)
      - [8.1 `MsgUpdateParam#ValidateBasic()`](#81-msgupdateparamvalidatebasic)
      - [8.2 `msgServer#UpdateParam()`](#82-msgserverupdateparam)

Adding a new on-chain module parameter involves multiple steps to ensure that the
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
- **New parameter name**: `new_param`
- **Default value**: `42`

When following these steps, be sure to substitute these example values with your own!
:::

### 0. If the Module Doesn't Already Support a `MsgUpdateParam` Message

In order to support **individual parameter updates**, the module MUST have a `MsgUpdateParam` message.
If the module doesn't already support this message, it will need to be added.

:::tip
At any point, you can always run `go test ./x/examplemod/...` to check whether everything is working or locate outstanding necessary changes.
:::

#### 0.1 Scaffold the `MsgUpdateParam` Message

Use `ignite` to scaffold a new `MsgUpdateParam` message for the module.
Additional flags are used for convenience:

```bash
ignite scaffold message update-param --module examplemod --signer authority name as_type --response params
```

:::info
If you experience errors like these:
```
✘ Error while running command /home/bwhite/go/bin/buf generate /tmp/proto-sdk2893110128/cosmos/upgrade/v1beta1/tx.proto ...: {..."message":"import \"gogoproto/gogo.proto\": file does not exist"}
: exit status 100
```
```
✘ Error while running command go mod tidy: go: 
...                                                                                              
go: github.com/pokt-network/poktroll/api/poktroll/examplemod imports                                                                                   
        cosmossdk.io/api/poktroll/shared: module cosmossdk.io/api@latest found (v0.7.6), but does not contain package cosmossdk.io/api/poktroll/shared
```
Then try running `make proto_clean_pulsar`.
:::

#### 0.2 Update the `MsgUpdateParam` Message Fields

Update the `MsgUpdateParam` message fields in the module's `tx.proto` file to include the following comments and protobuf options:

```protobuf
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

#### 0.3 Comment Out AutoCLI

When scaffolding the `MsgUpdateParam` message, lines are added to `x/examplemod/module/autocli.go`.
Since governance parameters aren't updated via `poktrolld` CLI, comment out these new lines:

```go
  // ...
  Tx: &autocliv1.ServiceCommandDescriptor{
      Service:              modulev1.Msg_ServiceDesc.ServiceName,
      EnhanceCustomCommand: true, // only required if you want to use the custom command
      RpcCommandOptions:    []*autocliv1.RpcCommandOptions{
          // ...
+         //	{
+         //		RpcMethod:      "UpdateParam",
+         //		Use:            "update-param [name] [as-type]",
+         //		Short:          "Send a update-param tx",
+         //		PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "name"}, {ProtoField: "asType"}},
+         //	},
          // this line is used by ignite scaffolding # autocli/tx
      },
  },
  // ...
```

#### 0.4. Update the DAO Genesis Authorizations JSON File

Add a grant (array element) to `tools/scripts/authz/dao_genesis_authorizations.json` with the `authorization.msg` typeURL for this module's `MsgUpdateType`:

```json
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

```go
- func NewMsgUpdateParam(authority string, name string, asType string) *MsgUpdateParam {
+ func NewMsgUpdateParam(authority string, name string, asType any) *MsgUpdateParam {
+ 	var asTypeIface isMsgUpdateParam_AsType
+
+ 	switch t := asType.(type) {
+ 	default:
+ 		panic(fmt.Sprintf("unexpected param value type: %T", asType))
+ 	}
+
    return &MsgUpdateParam{
	    Authority: authority,
		Name: name,
-       AsType: asType,
+       AsType: asTypeIface,
    }
  }

  func (msg *MsgUpdateParam) ValidateBasic() error {
        _, err := cosmostypes.AccAddressFromBech32(msg.Authority)
        if err != nil {
                return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
        }
+
+       // Parameter value cannot be nil.
+       if msg.AsType == nil {
+               return ErrGatewayParamInvalid.Wrap("missing param AsType")
+       }
+
+       // Parameter name must be supported by this module.
+       switch msg.Name {
+       default:
+               return ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
+       }

return nil
}
```
#### 0.6 Update the Module's `msgServer#UpdateParam()` Handler

Prepare `x/examplemod/keeper/msg_server_update_param.go` to handle parameter updates by type:

```go
+ // UpdateParam updates a single parameter in the proof module and returns
+ // all active parameters.
  func (k msgServer) UpdateParam(
    ctx context.Context,
    msg *types.MsgUpdateParam,
  ) (*types.MsgUpdateParamResponse, error) {
-   ctx := sdk.UnwrapSDKContext(goCtx)
-
-   // TODO: Handling the message
-   _ = ctx
-
+   if err := msg.ValidateBasic(); err != nil {
+       return nil, err
+   }
+
+ 	if k.GetAuthority() != msg.Authority {
+ 		return nil, types.ErrProofInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
+ 	}
+
+ 	params := k.GetParams(ctx)
+
+ 	switch msg.Name {
+ 	default:
+ 		return nil, types.ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
+ 	}
+
+ 	if err := k.SetParams(ctx, params); err != nil {
+ 		return nil, err
+ 	}
+
+ 	updatedParams := k.GetParams(ctx)
+ 	return &types.MsgUpdateParamResponse{
+ 		Params: &updatedParams,
+ 	}, nil
  }
```

#### 0.7 Update Module's Params Test Suite `ModuleParamConfig`

Add `MsgUpdateParam` & `MsgUpdateParamResponse` to the module's `ModuleParamConfig#ParamsMsg`:

```go
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

Open the appropriate `.proto` file for your module (e.g., `params.proto`) and define the new parameter.

```protobuf
  message Params {
    // Other existing parameters...
  
+   // Description of the new parameter.
+   uint64 new_parameter = 3 [(gogoproto.jsontag) = "new_parameter", (gogoproto.moretags) = "yaml:\"new_parameter\""];
  }
```

:::tip
Don't forget to run `make proto_regen` to update generated protobuf go code.
:::

### 2 Update the Parameter Integration Tests 

Integration tests which cover parameter updates utilize the `ModuleParamConfig`s defined in [`testutil/integration/params/param_configs.go`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/suites/param_configs.go) to dynamically (i.e. using reflection) construct and send parameter update messages in a test environment.
When adding parameters to a module, it is necessary to update that module's `ModuleParamConfig` to include the new parameter, othwerwise it will not be covered by the integration test suite.

#### 2.1 Add a valid param

Update `ModuleParamConfig#ValidParams` to include a valid and non-default value for the new parameter:

```go
  ExamplemodModuleParamConfig = ModuleParamConfig{
      // ...
      ValidParams: gatewaytypes.Params{
+         NewParameter: 420,
      },
      // ...
  }
```

#### 2.2 Check for `as_<type>` on `MsgUpdateParam`

Ensure an `as_<type>` field exists on `MsgUpdateParam` corresponding to the type of the new parameter:

```proto
 message MsgUpdateParam {
   ...
   oneof as_type {
-    // Add `as_<type>` fields for each type in this module's Params type; e.g.:
+    int64 as_int64 = 3 [(gogoproto.jsontag) = "as_int64"];
   }
 }
```

### 3. Update the Default Parameter Values

#### 3.1 Go Source Defaults

In the corresponding Go file (e.g., `x/examplemod/types/params.go`), define the default
value, key, and parameter name for the new parameter and include the default in the
`NewParams` and `DefaultParams` functions:

```go
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

### 3.2 Genesis Configuration Parameter Defaults

Add the new parameter to the genesis configuration file (e.g., `config.yml`):

```yaml
  genesis:
    examplemod:
      params:
        # Other existing parameters...
  
+       new_parameter: 42
```

### 4. Update the Makefile and Supporting JSON Files

#### 4.1 Update the Makefile

Add a new target in the `Makefile` to update the new parameter.
Below is an example of adding the make target corresponding to the `shared` module's `num_blocks_per_session` param:

```makefile
.PHONY: params_update_examplemod_new_parameter
params_update_examplemod_new_parameter: ## Update the examplemod module new_parameter param
  poktrolld tx authz exec ./tools/scripts/params/examplemod_new_parameter.json $(PARAM_FLAGS)
```

:::warning
Reminder to substitute `examplemod` and `new_parameter` with your module and param names!
:::

#### 4.2 Create a new JSON File for the Individual Parameter Update

Create a new JSON file (e.g., `proof_new_parameter_name.json`) in the tools/scripts/params directory to specify how to update the new parameter:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/poktroll.examplemod.MsgUpdateParam", // Replace module name
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "new_parameter", // Replace new parameter name
        "as_int64": "42"         // Replace default value
      }
    ]
  }
}
```

#### 4.3 Update the JSON File for Updating All Parameters for the Module

Add a line to the existing module's `MsgUpdateParam` JSON file (e.g., `proof_all.json`)
with the default value for the new parameter.

```json
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

### 5. Parameter Validation

#### 5.1 New Parameter Validation

Implement a validation function for the new parameter in `x/examplemod/types/params.go`:

```go
+ func ValidateNewParameter(v interface{}) error {
+   _, ok := v.(int64)
+   if !ok {
+     return fmt.Errorf("invalid parameter type: %T", v)
+   }
+   return nil
+ }
```

#### 5.2 Parameter Validation in Workflow

Integrate the usage of the new `ValidateNewParameter` function in the corresponding
`Params#Validate()` function where this is used:

```go
  func (params *Params) Validate() error {
    // ...
+   if err := ValidateNewParameter(params.NewParameter); err != nil {
+     return err
+   }
    // ...
  }
```

### 6. Add the Parameter to `ParamSetPairs()`

Include the new parameter in the `ParamSetPairs` function return:

```go
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

### 7. Update Unit Tests

#### 7.1 Parameter Validation Tests

Add unit tests which exercise validation of the new parameter(s) in `x/examplemod/keeper/params_test.go`:
Ensure there is a test function for each parameter which covers all cases of invalid input

```go
  func TestGetParams(t *testing.T) {
    // ...
  }
  
+ func TestParams_ValidateNewParameter(t *testing.T) {
+   tests := []struct {
+     desc string
+     newParameter interface{}
+     expectedErr error
+   }{
+     {
+       desc: "invalid type",
+       newParameter: "420",
+       expectedErr: fmt.Errorf("invalid parameter type: string"),
+     },
+     {
+       desc: "valid newParameterName",
+       newParameter: int64(420),
+     },
+   }
+ 
+   for _, tt := range tests {
+     t.Run(tt.desc, func(t *testing.T) {
+       err := ValidateNewParameter(tt.newParameter)
+       if tt.expectedErr != nil {
+         require.Error(t, err)
+         require.Contains(t, err.Error(), tt.expectedErr.Error())
+       } else {
+         require.NoError(t, err)
+       }
+     })
+   }
+ }
```

#### 7.2 Parameter Update Tests

Add test cases to `x/examplemod/keeper/msg_update_params_test.go` to ensure coverage over any invalid parameter configurations.
Add a case for the "minimal params" if some subset of the params are "required".
If one already exist, update it if applicable; e.g.:

```go
+ {
+     desc: "valid: send minimal params", // For parameters which MUST NEVER be their zero value or nil.
+     input: &examplemodtypes.MsgUpdateParams{
+         Authority: k.GetAuthority(),
+         Params: examplemodtypes.Params{
+             NewParameter: 42, 
+         },
+     },
+     shouldError: false,
+ },
```

Add unit tests which exercise individual parameter updates in `x/examplemod/keeper/msg_server_update_param_test.go`.
These tests assert that the value of a given parameter is:

```go
+ func TestMsgUpdateParam_UpdateNewParameterOnly(t *testing.T) {
+   var expectedNewParameter uint64 = 420
+ 
+   // Set the parameters to their default values
+   k, msgSrv, ctx := setupMsgServer(t)
+   defaultParams := prooftypes.DefaultParams()
+   require.NoError(t, k.SetParams(ctx, defaultParams))
+ 
+   // Ensure the default values are different from the new values we want to set
+   require.NotEqual(t, expectedNewParameter, defaultParams.NewParameter)
+ 
+   // Update the new parameter
+   updateParamMsg := &examplemodtypes.MsgUpdateParam{
+     Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
+     Name: examplemodtypes.ParamNewParameter,
+     AsType: &examplemodtypes.MsgUpdateParam_AsInt64{AsInt64: int64(expectedNewParameter)},
+   }
+   res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
+   require.NoError(t, err)
+ 
+   require.Equal(t, expectedNewParameter, res.Params.NewParameter)
+ 
+   // IMPORTANT!: THIS TEST SHOULD ALSO ASSERT THAT ALL OTHER PARAMS OF THE SAME MODULE REMAIN UNCHANGED
+ }
```

### 8. Add Parameter Case to Switch Statements

#### 8.1 `MsgUpdateParam#ValidateBasic()`

Add the parameter name (e.g. `ParamNameNewParameter`) to a new case in the switch in `MsgUpdateParam#ValidateBasic()` in `x/examplemod/types/message_update_param.go`:

```go
  func NewMsgUpdateParam(authority string, name string, asType any) *MsgUpdateParam {
        // ...
        switch t := asType.(type) {
+       case int64:
+               asTypeIface = &MsgUpdateParam_AsCoin{AsInt64: t}
        default:
                panic(fmt.Sprintf("unexpected param value type: %T", asType))
        }
        // ...
  }

  func (msg *MsgUpdateParam) ValidateBasic() error {
        // ...
        switch msg.Name {
+       case ParamNewParameter:
+               return msg.paramTypeIsInt64()
        default:
                return ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
        }
  }
+
+ func (msg *MsgUpdateParam) paramTypeIsInt64() error {
+       _, ok := msg.AsType.(*MsgUpdateParam_AsInt64)
+       if !ok {
+               return ErrExamplemodParamInvalid.Wrapf(
+                       "invalid type for param %q expected %T type: %T",
+                       msg.Name, &MsgUpdateParam_AsInt64{}, msg.AsType,
+               )
+       }
+
+       return nil
+ }
```

#### 8.2 `msgServer#UpdateParam()`

Add the parameter name (e.g. `ParamNameNewParameter`) to a new case in the switch statement in `msgServer#UpdateParam()` in `x/examplemod/keeper/msg_server_update_param.go`:

```go
  // UpdateParam updates a single parameter in the proof module and returns
  // all active parameters.
  func (k msgServer) UpdateParam(
      ctx context.Context,
      msg *types.MsgUpdateParam,
  ) (*types.MsgUpdateParamResponse, error) {
    // ...
  	switch msg.Name {
+ 	case types.ParamNewParameter:
+ 		asType, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
+ 		if !ok {
+ 			return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
+ 		}
+ 		newParameter := value.AsInt64
+
+ 		if err := types.ValidateNewParameter(newParameter); err != nil {
+ 			return nil, err
+ 		}
+
+ 		params.NewParameter = newParameter
  	default:
  		return nil, types.ErrExamplemodParamInvalid.Wrapf("unsupported param %q", msg.Name)
  	}
    // ...
  }
```
