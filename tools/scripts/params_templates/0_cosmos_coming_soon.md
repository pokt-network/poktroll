# Cosmos Param Updates

Create a file for each of these:

- /cosmos.auth.v1beta1.Msg/UpdateParams:
- /cosmos.bank.v1beta1.Msg/UpdateParams:
- /cosmos.consensus.v1.Msg/UpdateParams:
- /cosmos.crisis.v1beta1.Msg/UpdateParams:
- /cosmos.distribution.v1beta1.Msg/UpdateParams:
- /cosmos.gov.v1.Msg/UpdateParams:
- /cosmos.mint.v1beta1.Msg/UpdateParams:
- /cosmos.protocolpool.v1.Msg/UpdateParams:
- /cosmos.slashing.v1beta1.Msg/UpdateParams:
- /cosmos.staking.v1beta1.Msg/UpdateParams:

Make target will:

- Retrieve onchain params
- Write them to a local .json file
- Prompt user to update the param they need
- Show the command to submit the appropriate message

```json
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.application.MsgUpdateParams",
        "authority": "",
        "params": {
          "TODO"
        }
      }
    ]
  }
}
```
