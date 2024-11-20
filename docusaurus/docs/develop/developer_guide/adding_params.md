---
sidebar_position: 5
title: Adding On-Chain Module Parameters
---

# Adding On-Chain Module Parameters <!-- omit in toc -->

- [Step-by-Step Instructions](#step-by-step-instructions)
  - [1. Define the Parameter in the Protocol Buffers File](#1-define-the-parameter-in-the-protocol-buffers-file)
  - [2 Update the Parameter Integration Tests](#2-update-the-parameter-integration-tests)
  - [3. Update the Default Parameter Values](#3-update-the-default-parameter-values)
  - [4. Add Parameter Default to Genesis Configuration](#4-add-parameter-default-to-genesis-configuration)
  - [5. Modify the Makefile](#5-modify-the-makefile)
  - [6. Create a new JSON File for the Individual Parameter Update](#6-create-a-new-json-file-for-the-individual-parameter-update)
  - [7. Update the JSON File for Updating All Parameters for the Module](#7-update-the-json-file-for-updating-all-parameters-for-the-module)
  - [8. Parameter Validation](#8-parameter-validation)
    - [8.1 New Parameter Validation](#81-new-parameter-validation)
    - [8.2 Parameter Validation in Workflow](#82-parameter-validation-in-workflow)
  - [9. Add the Parameter to `ParamSetPairs()`](#9-add-the-parameter-to-paramsetpairs)
  - [10. Update Unit Tests](#10-update-unit-tests)
    - [10.1 Parameter Validation Tests](#101-parameter-validation-tests)
    - [10.2 Parameter Update Tests](#102-parameter-update-tests)
  - [11. Implement individual parameter updates](#11-implement-individual-parameter-updates)
    - [11.1 Add `ParamNameNewParameterName` to `MsgUpdateParam#ValidateBasic()` in `x/types/message_update_param.go`](#111-add-paramnamenewparametername-to-msgupdateparamvalidatebasic-in-xtypesmessage_update_paramgo)
    - [11.2 Add `ParamNameNewParameterName` to `msgServer#UpdateParam()` in `x/keeper/msg_server_update_param.go`](#112-add-paramnamenewparametername-to-msgserverupdateparam-in-xkeepermsg_server_update_paramgo)

Adding a new on-chain module parameter involves multiple steps to ensure that the
parameter is properly integrated into the system. This guide will walk you through
the process using a generic approach, illustrated by adding a parameter to the `proof` module.

See [pokt-network/poktroll#595](https://github.com/pokt-network/poktroll/pull/595) for a real-world example.

:::note

TODO_POST_MAINNET(@bryanchriswhite): Once the next version of `ignite` is out, leverage:
https://github.com/ignite/cli/issues/3684#issuecomment-2299796210

:::

## Step-by-Step Instructions

### 1. Define the Parameter in the Protocol Buffers File

Open the appropriate `.proto` file for your module (e.g., `params.proto`) and define the new parameter.

```protobuf
message Params {
  // Other existing parameters...

  // Description of the new parameter.
  uint64 new_parameter_name = 3 [(gogoproto.jsontag) = "new_parameter_name"];
}
```

### 2 Update the Parameter Integration Tests

// TODO_DOCUMENT(@bryanchriswhite, #826)

### 3. Update the Default Parameter Values

In the corresponding Go file (e.g., `x/<module>/types/params.go`), define the default value, key, and parameter name for the
new parameter and include the default in the `NewParams` and `DefaultParams` functions.

```go
var (
  // Other existing parameters...

  KeyNewParameterName = []byte("NewParameterName")
  ParamNewParameterName = "new_parameter_name"
  DefaultNewParameterName uint64 = 100 // Example default value
)

func NewParams(
  // Other existing parameters...
  newParameterName uint64,
) Params {
  return Params{
    // Other existing parameters...
    NewParameterName: newParameterName,
  }
}

func DefaultParams() Params {
  return NewParams(
    // Other existing default parameters...
    DefaultNewParameterName,
  )
}
```

### 4. Add Parameter Default to Genesis Configuration

Add the new parameter to the genesis configuration file (e.g., `config.yml`).

```yaml
genesis:
  proof:
    params:
      # Other existing parameters...

      new_parameter_name: 100
```

### 5. Modify the Makefile

Add a new target in the `Makefile` to update the new parameter.

```makefile
.PHONY: params_update_proof_new_parameter_name
params_update_proof_new_parameter_name: ## Update the proof module new_parameter_name param
  poktrolld tx authz exec ./tools/scripts/params/proof_new_parameter_name.json $(PARAM_FLAGS)
```

### 6. Create a new JSON File for the Individual Parameter Update

Create a new JSON file (e.g., `proof_new_parameter_name.json`) in the tools/scripts/params
directory to specify how to update the new parameter.

```json
{
  "body": {
    "messages": [
      {
        "@type": "/poktroll.proof.MsgUpdateParam",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "new_parameter_name",
        "as_int64": "100"
      }
    ]
  }
}
```

### 7. Update the JSON File for Updating All Parameters for the Module

Add a line to the existing module's `MsgUpdateParam` JSON file (e.g., `proof_all.json`)
with the default value for the new parameter.

```json
{
  "body": {
    "messages": [
      {
        "@type": "/poktroll.proof.MsgUpdateParams",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "params": {
          "proof_request_probability": "0.25",
          "proof_requirement_threshold": {
            "denom": "upokt",
            "amount": "20000000"
          },
          "proof_missing_penalty": {
            "amount": "320000000",
            "denom": "upokt"
          },
          "proof_submission_fee": {
            "amount": "1000000",
            "denom": "upokt"
          },
          "new_parameter_name": "100" // Add this line
        }
      }
    ]
  }
}
```

### 8. Parameter Validation

#### 8.1 New Parameter Validation

Implement a validation function for the new parameter in the corresponding `params.go`
file you've been working on.

```go
func ValidateNewParameterName(v interface{}) error {
  _, ok := v.(uint64)
  if !ok {
    return fmt.Errorf("invalid parameter type: %T", v)
  }
  return nil
}
```

#### 8.2 Parameter Validation in Workflow

Integrate the usage of the new `ValidateNewParameterName` function in the corresponding
`Params#ValidateBasic()` function where this is used.

```go
func (params *Params) ValidateBasic() error {
  // ...
  if err := ValidateNewParameterName(params.NewParameterName); err != nil {
    return err
  }
  // ...
}
```

### 9. Add the Parameter to `ParamSetPairs()`

Include the new parameter in the `ParamSetPairs` function.

```go
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
  return paramtypes.ParamSetPairs{
    // Other existing parameters...

    paramtypes.NewParamSetPair(
      KeyNewParameterName,
      &p.NewParameterName,
      ValidateNewParameterName,
    ),
  }
}
```

### 10. Update Unit Tests

Add tests which exercise validation of the new parameter in your test files
(e.g., `/types/params_test.go` and `msg_server_update_param_test.go`).

#### 10.1 Parameter Validation Tests

```go
func TestParams_ValidateNewParameterName(t *testing.T) {
  tests := []struct {
    desc string
    newParameterName interface{}
    expectedErr error
  }{
    {
      desc: "invalid type",
      newParameterName: int64(-1),
      expectedErr: fmt.Errorf("invalid parameter type: int64"),
    },
    {
      desc: "valid newParameterName",
      newParameterName: uint64(100),
    },
  }

  for _, tt := range tests {
    t.Run(tt.desc, func(t *testing.T) {
      err := ValidateNewParameterName(tt.newParameterName)
      if tt.expectedErr != nil {
        require.Error(t, err)
        require.Contains(t, err.Error(), tt.expectedErr.Error())
      } else {
        require.NoError(t, err)
      }
    })
  }
}
```

#### 10.2 Parameter Update Tests

The example presented below corresponds to `/keeper/msg_server_update_param_test.go`.

```go
func TestMsgUpdateParam_UpdateNewParameterNameOnly(t *testing.T) {
  var expectedNewParameterName uint64 = 100

  // Set the parameters to their default values
  k, msgSrv, ctx := setupMsgServer(t)
  defaultParams := prooftypes.DefaultParams()
  require.NoError(t, k.SetParams(ctx, defaultParams))

  // Ensure the default values are different from the new values we want to set
  require.NotEqual(t, expectedNewParameterName, defaultParams.NewParameterName)

  // Update the new parameter
  updateParamMsg := &prooftypes.MsgUpdateParam{
    Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    Name: prooftypes.ParamNewParameterName,
    AsType: &prooftypes.MsgUpdateParam_AsInt64{AsInt64: int64(expectedNewParameterName)},
  }
  res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
  require.NoError(t, err)

  require.Equal(t, expectedNewParameterName, res.Params.NewParameterName)

  // READ ME: THIS TEST SHOULD ALSO ASSERT THAT ALL OTHER PARAMS OF THE SAME MODULE REMAIN UNCHANGED
}
```

### 11. Implement individual parameter updates

#### 11.1 Add `ParamNameNewParameterName` to `MsgUpdateParam#ValidateBasic()` in `x/types/message_update_param.go`

```go
  // Parameter name must be supported by this module.
  switch msg.Name {
  case ParamNumBlocksPerSession,
    ParamNewParameterName:
    return msg.paramTypeIsInt64()
```

#### 11.2 Add `ParamNameNewParameterName` to `msgServer#UpdateParam()` in `x/keeper/msg_server_update_param.go`

```go
case types.ParamNewParameterName:
  value, ok := msg.AsType.(*types.MsgUpdateParam_AsInt64)
  if !ok {
    return nil, types.ErrProofParamInvalid.Wrapf("unsupported value type for %s param: %T", msg.Name, msg.AsType)
  }
  newParameter := uint64(value.AsInt64)

  if err := types.ValidateNewParameter(newParameter); err != nil {
    return nil, err
  }

  params.NewParameter = newParameter
```
