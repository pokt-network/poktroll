---
sidebar_position: 5
title: Adding On-Chain Module Parameters
---

https://docusaurus.io/docs/next/markdown-features/code-blocks#supported-languages

https://github.com/FormidableLabs/prism-react-renderer/blob/master/packages/generate-prism-languages/index.ts#L9-L23

https://prismjs.com/#supported-languages

# Adding On-Chain Module Parameters <!-- omit in toc -->

- [Adding a New On-Chain Module Parameter](#adding-a-new-on-chain-module-parameter)
  - [Step-by-Step Instructions](#step-by-step-instructions)

## Adding a New On-Chain Module Parameter

Adding a new on-chain module parameter involves multiple steps to ensure that the parameter is properly integrated into
the system. This guide will walk you through the process using a generic approach, illustrated by adding a parameter to
the "proof" module.

See [pokt-network/poktroll#595](https://github.com/pokt-network/poktroll/pull/595) for a real-world example.

### Step-by-Step Instructions

1. **Define the Parameter in the Protocol Buffers File**

   Open the appropriate `.proto` file for your module (e.g., `params.proto`) and define the new parameter.

   ```protobuf
   message Params {
     // Other existing parameters...

     // Description of the new parameter.
     uint64 new_parameter_name = 3 [(gogoproto.jsontag) = "new_parameter_name"];
   }
   ```

1. **Update the Parameter E2E Tests**

   Update the E2E test files (e.g., `update_params.feature` and `update_params_test.go`) to include scenarios that test
   the new parameter.

   ```gherkin
      # NB: If you are reading this and the proof module has parameters
      # that are not being updated in this test, please update the test.
      Scenario: An authorized user updates all "proof" module params
      Given the user has the pocketd binary installed
      And all "proof" module params are set to their default values
      And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "/poktroll.proof.MsgUpdateParams" message exists
      When the "pnf" account sends an authz exec message to update all "proof" module params
      | name                | value | type  |
      | new_parameter_name  | 100   | int64 |
    Then all "proof" module params should be updated
   ```

   ```gherkin
      # NB: If you are reading this and any module has parameters that
      # are not being updated in this test, please update the test.
      Scenario Outline: An authorized user updates individual <module> module params
      Given the user has the pocketd binary installed
      And all "<module>" module params are set to their default values
      And an authz grant from the "gov" "module" account to the "pnf" "user" account for the "<message_type>" message exists
      When the "pnf" account sends an authz exec message to update "<module>" the module param
      | name         | value         | type         |
      | <param_name> | <param_value> | <param_type> |
      Then the "<module>" module param "<param_name>" should be updated

          Examples:
            | module | message_type                   | param_name         | param_value | param_type |
            | proof  | /poktroll.proof.MsgUpdateParam | new_parameter_name | 100         | int64      |
   ```

   ```go
   func (s *suite) newProofMsgUpdateParams(params paramsMap) cosmostypes.Msg {
     msgUpdateParams := &prooftypes.MsgUpdateParam{
       Params: &prooftypes.Params{},
     }

     for paramName, paramValue := range params {
       switch paramName {
       case prooftypes.ParamNewParameterName:
         msgUpdateParams.Params.NewParameterName = uint64(paramValue.value.(int64))
       default:
         s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
       }
     }

     return msgUpdateParams
   }
   ```

1. **Update the Default Parameter Values**

   In the corresponding Go file (e.g., `params.go`), define the default value for the new parameter and include it in
   the `NewParams` and `DefaultParams` functions.

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

1. **Add Parameter Default to Genesis Configuration**

   Add the new parameter to the genesis configuration file (e.g., `config.yml`).

   ```yaml
   genesis:
     proof:
       params:
         # Other existing parameters...

         new_parameter_name: 100
   ```

1. **Modify the Makefile**

   Add a new target in the `Makefile` to update the new parameter.

   ```makefile
   .PHONY: params_update_proof_new_parameter_name
   params_update_proof_new_parameter_name: ## Update the proof module new_parameter_name param
     poktrolld tx authz exec ./tools/scripts/params/proof_new_parameter_name.json $(PARAM_FLAGS)
   ```

1. **Create a new JSON File for the Individual Parameter Update**

   Create a new JSON file (e.g., `proof_new_parameter_name.json`) in the appropriate directory to specify how to update
   the new parameter.

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

1. **Update the JSON File for Updating All Parameters for the Module**

   Add a line to the existing module's `MsgUpdateParam` JSON file (e.g., `proof_all.json`) with the default value for
   the new parameter.

   ```json
   {
     "body": {
       "messages": [
         {
           "@type": "/poktroll.proof.MsgUpdateParams",
           "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
           "params": {
             "min_relay_difficulty_bits": "0",
             "proof_request_probability": "0.25",
             "proof_requirement_threshold": "20",
             "new_parameter_name": "100" // Add this line
           }
         }
       ]
     }
   }
   ```

1. **Update Tests**

   Add tests to validate the new parameter in your test files (e.g., `params_test.go`
   and `msg_server_update_param_test.go`).

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
   }
   ```

1. **Add Parameter Validation**

   Implement a validation function for the new parameter in the same Go file.

   ```go
   func ValidateNewParameterName(v interface{}) error {
     _, ok := v.(uint64)
     if !ok {
       return fmt.Errorf("invalid parameter type: %T", v)
     }
     return nil
   }
   ```

1. **Add the Parameter to `ParamSetPairs()`**

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
