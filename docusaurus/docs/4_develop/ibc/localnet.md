# LocalNet IBC Development

## Overview

The Poktroll LocalNet provides a complete IBC testing environment with multiple blockchain networks running locally. This allows developers to test cross-chain functionality without relying on external testnets.

## Quick Start

### Prerequisites
- Ensure you can run LocalNet according to the [Developer Guide -> Walkthrough](../developer_guide/walkthrough.md)
- Docker and Kubernetes tools installed
- Basic understanding of IBC concepts

### Setup Steps

1. **Start LocalNet**
   ```bash
   make localnet_up
   ```

2. **Verify Networks are Running**
   - Wait for Pocket and counterparty validators to produce blocks
   - Check `localnet_config.yaml` for enabled counterparty networks

3. **Initialize IBC Connections**
   - Look for IBC pair setup services (`🏗️ Pokt-><counterparty>`) in Tilt UI
   - If any service shows persistent backoff errors, manually trigger it

4. **Start Relayers**
   - Once IBC setup completes successfully, restart the `🔁 Hermes Relayer` service
   - Monitor relayer logs for successful packet transmission

<table>
<tbody>
<tr>
<td style={{verticalAlign: "top"}}>

![ibc_tilt_resources.png](img/ibc_tilt_resources.png)

</td>
<td style={{verticalAlign: "top"}}>

![ibc_tilt_setup.png](img/ibc_tilt_setup.png)

</td>
</tr>
</tbody>
</table>

## Understanding IBC Setup

### IBC Components
For each blockchain pair, IBC requires three components:
- **Client**: Light client tracking the remote chain's state  
- **Connection**: Authenticated connection between two chains
- **Channel**: Application-specific communication pathway

### Automated Setup Process
LocalNet provides automated IBC setup via Tilt services:
- Format: `🏗️ Pokt-><counterparty>` (e.g., `🏗️ Pokt->Axelar`)
- Services auto-start after dependencies are ready
- Automatic retry with exponential backoff on failures
- Manual triggering available via Tilt UI

:::warning Concurrent Setup
Avoid running multiple IBC pair setups simultaneously as they may interfere with account sequence numbers. While Hermes will auto-retry with correct sequences, this may affect setup time.
:::

The following uses the `pocket-axelar` pair as an example.

Example setup log output:

<details>
<summary>

```
2025-06-26T11:40:41.166006Z  INFO ThreadId(01) running Hermes v1.13.1+5e403dd
2025-06-26T11:40:45.211899Z  INFO ThreadId(01) Creating new clients, new connection, and a new channel with order ORDER_UNORDERED
2025-06-26T11:40:50.240782Z  INFO ThreadId(01) foreign_client.create{client=axelar->pocket:07-tendermint-0}: 🍭 client was created successfully id=07-tendermint-0
2025-06-26T11:41:06.131924Z  INFO ThreadId(01) foreign_client.create{client=pocket->axelar:07-tendermint-0}: 🍭 client was created successfully id=07-tendermint-0
2025-06-26T11:41:09.671319Z  INFO ThreadId(01) 🥂 pocket => OpenInitConnection(OpenInit { Attributes { connection_id: connection-0, client_id: 07-tendermint-0, counterparty_connection_id: None, counterparty_client_id: 07-tendermint-0 } }) at height 0-22
2025-06-26T11:41:56.934211Z  INFO ThreadId(01) 🥂 axelar => OpenTryConnection(OpenTry { Attributes { connection_id: connection-0, client_id: 07-tendermint-0, counterparty_connection_id: connection-0, counterparty_client_id: 07-tendermint-0 } }) at height 0-19
2025-06-26T11:42:23.508663Z  INFO ThreadId(01) 🥂 pocket => OpenAckConnection(OpenAck { Attributes { connection_id: connection-0, client_id: 07-tendermint-0, counterparty_connection_id: connection-0, counterparty_client_id: 07-tendermint-0 } }) at height 0-58
2025-06-26T11:42:58.297181Z  INFO ThreadId(01) 🥂 axelar => OpenConfirmConnection(OpenConfirm { Attributes { connection_id: connection-0, client_id: 07-tendermint-0, counterparty_connection_id: connection-0, counterparty_client_id: 07-tendermint-0 } }) at height 0-31
2025-06-26T11:43:06.326874Z  INFO ThreadId(01) connection handshake already finished for Connection { delay_period: 0ns, a_side: ConnectionSide { chain: BaseChainHandle { chain_id: pocket }, client_id: 07-tendermint-0, connection_id: connection-0 }, b_side: ConnectionSide { chain: BaseChainHandle { chain_id: axelar }, client_id: 07-tendermint-0, connection_id: connection-0 } }
2025-06-26T11:43:08.162131Z  INFO ThreadId(01) 🎊  pocket => OpenInitChannel(OpenInit { port_id: transfer, channel_id: channel-0, connection_id: None, counterparty_port_id: transfer, counterparty_channel_id: None }) at height 0-79
2025-06-26T11:43:19.232828Z  INFO ThreadId(01) 🎊  axelar => OpenTryChannel(OpenTry { port_id: transfer, channel_id: channel-0, connection_id: connection-0, counterparty_port_id: transfer, counterparty_channel_id: channel-0 }) at height 0-35
2025-06-26T11:43:53.673647Z  INFO ThreadId(01) 🎊  pocket => OpenAckChannel(OpenAck { port_id: transfer, channel_id: channel-0, connection_id: connection-0, counterparty_port_id: transfer, counterparty_channel_id: channel-0 }) at height 0-100
2025-06-26T11:44:11.728489Z  INFO ThreadId(01) 🎊  axelar => OpenConfirmChannel(OpenConfirm { port_id: transfer, channel_id: channel-0, connection_id: connection-0, counterparty_port_id: transfer, counterparty_channel_id: channel-0 }) at height 0-45
2025-06-26T11:44:19.747405Z  INFO ThreadId(01) channel handshake already finished for Channel { ordering: ORDER_UNORDERED, a_side: ChannelSide { chain: BaseChainHandle { chain_id: pocket }, client_id: 07-tendermint-0, connection_id: connection-0, port_id: transfer, channel_id: channel-0, version: None }, b_side: ChannelSide { chain: BaseChainHandle { chain_id: axelar }, client_id: 07-tendermint-0, connection_id: connection-0, port_id: transfer, channel_id: channel-0, version: None }, connection_delay: 0ns }
```

</summary>

```
SUCCESS Channel {
    ordering: Unordered,
    a_side: ChannelSide {
        chain: BaseChainHandle {
            chain_id: ChainId {
                id: "pocket",
                version: 0,
            },
            runtime_sender: Sender { .. },
        },
        client_id: ClientId(
            "07-tendermint-0",
        ),
        connection_id: ConnectionId(
            "connection-0",
        ),
        port_id: PortId(
            "transfer",
        ),
        channel_id: Some(
            ChannelId(
                "channel-0",
            ),
        ),
        version: None,
    },
    b_side: ChannelSide {
        chain: BaseChainHandle {
            chain_id: ChainId {
                id: "axelar",
                version: 0,
            },
            runtime_sender: Sender { .. },
        },
        client_id: ClientId(
            "07-tendermint-0",
        ),
        connection_id: ConnectionId(
            "connection-0",
        ),
        port_id: PortId(
            "transfer",
        ),
        channel_id: Some(
            ChannelId(
                "channel-0",
            ),
        ),
        version: None,
    },
    connection_delay: 0ns,
}
```

</details>


## Monitoring IBC State

Use these commands to inspect IBC state across your LocalNet networks:

### Clients
View light clients tracking remote chain states:
```bash
# List clients for any network
make ibc_list_<network>_clients

# Examples
make ibc_list_axelar_clients
make ibc_list_osmosis_clients
```

### Connections  
Check authenticated connections between chains:
```bash
# List connections for any network
make ibc_list_<network>_connections

# Examples
make ibc_list_axelar_connections
make ibc_list_pocket_connections
```

### Channels
Monitor communication channels for specific applications:
```bash
# List channels for any network
make ibc_list_<network>_channels

# Examples  
make ibc_list_axelar_channels
make ibc_list_osmosis_channels
```

### Interchain Accounts
Coming Soon: ICA testing documentation

## Testing IBC Functionality

### Token Transfers

Test bi-directional token transfers between Poktroll and counterparty chains:

#### Available Transfer Routes
```bash
# Pocket to counterparty chains
make ibc_test_transfer_pocket_to_axelar
make ibc_test_transfer_pocket_to_osmosis  
make ibc_test_transfer_pocket_to_agoric

# Counterparty chains to Pocket
make ibc_test_transfer_axelar_to_pocket
make ibc_test_transfer_osmosis_to_pocket
make ibc_test_transfer_agoric_to_pocket
```

#### Example: Transfer 1000upokt from Pocket to Axelar
```bash
$ make ibc_test_transfer_pocket_to_axelar
code: 0
codespace: ""
data: ""
events: []
gas_used: "0"
gas_wanted: "0"
height: "0"
info: ""
logs: []
raw_log: ""
timestamp: ""
tx: null
txhash: 35D669173501D1DCBB5441927A075BE813E18C1D0FE174DF80146E4E48D754CF

# Check that the TX on the source chain was successful
$ make query_tx_json_short HASH=35D669173501D1DCBB5441927A075BE813E18C1D0FE174DF80146E4E48D754CF
{
  "code": 0,  # <-- code 0 == success
  "log": "",  # <-- empty log == success; otherwise, log contains error details
  "timestamp": "2025-07-02T09:05:04Z",
  "height": "5457",
  "txhash": "35D669173501D1DCBB5441927A075BE813E18C1D0FE174DF80146E4E48D754CF"
}

# Hermes relayer logs:
2025-07-02T09:05:09.539879Z  INFO ThreadId(337) worker.batch{chain=pocket}:supervisor.handle_batch{chain=pocket}:supervisor.process_batch{chain=pocket}:worker.packet.cmd{src_chain=pocket src_port=transfer src_channel=channel-3 dst_chain=axelar}:relay{odata=b21976eb ->Destination @0-5457; len=1}: assembled batch of 2 message(s)
2025-07-02T09:05:09.556979Z  INFO ThreadId(337) worker.batch{chain=pocket}:supervisor.handle_batch{chain=pocket}:supervisor.process_batch{chain=pocket}:worker.packet.cmd{src_chain=pocket src_port=transfer src_channel=channel-3 dst_chain=axelar}:relay{odata=b21976eb ->Destination @0-5457; len=1}: response(s): 1; Ok:2D7A3168CBFBB0DC76D49D7AA1BF00810BA22C37C2A3BB3C56A3AF4A03CF3DE0 target_chain=axelar
2025-07-02T09:05:09.557018Z  INFO ThreadId(337) worker.batch{chain=pocket}:supervisor.handle_batch{chain=pocket}:supervisor.process_batch{chain=pocket}:worker.packet.cmd{src_chain=pocket src_port=transfer src_channel=channel-3 dst_chain=axelar}:relay{odata=b21976eb ->Destination @0-5457; len=1}: submitted
2025-07-02T09:05:12.876224Z  INFO ThreadId(30) worker.batch{chain=axelar}:supervisor.handle_batch{chain=axelar}:supervisor.process_batch{chain=axelar}:worker.client.misbehaviour{client=07-tendermint-0 src_chain=pocket dst_chain=axelar}:foreign_client.detect_misbehaviour_and_submit_evidence{client=pocket->axelar:07-tendermint-0}:foreign_client.detect_misbehaviour{client=pocket->axelar:07-tendermint-0}: No evidence of misbehavior detected for chain pocket
2025-07-02T09:05:12.976365Z  INFO ThreadId(87) worker.batch{chain=axelar}:supervisor.handle_batch{chain=axelar}:supervisor.process_batch{chain=axelar}:worker.client.misbehaviour{client=07-tendermint-0 src_chain=pocket dst_chain=axelar}:foreign_client.detect_misbehaviour_and_submit_evidence{client=pocket->axelar:07-tendermint-0}: client is valid
2025-07-02T09:05:17.680828Z  INFO ThreadId(284) worker.batch{chain=axelar}:supervisor.handle_batch{chain=axelar}:supervisor.process_batch{chain=axelar}:worker.packet.cmd{src_chain=axelar src_port=transfer src_channel=channel-0 dst_chain=pocket}:relay{odata=7f01e6e0 ->Destination @0-1563; len=1}: assembled batch of 2 message(s)
2025-07-02T09:05:18.034841Z  INFO ThreadId(284) worker.batch{chain=axelar}:supervisor.handle_batch{chain=axelar}:supervisor.process_batch{chain=axelar}:worker.packet.cmd{src_chain=axelar src_port=transfer src_channel=channel-0 dst_chain=pocket}:relay{odata=7f01e6e0 ->Destination @0-1563; len=1}: response(s): 1; Ok:FC2E41FEC8AF3EC03614B18013D579D36506E748C18533723BDC53513E98E9E4 target_chain=pocket
2025-07-02T09:05:18.034869Z  INFO ThreadId(284) worker.batch{chain=axelar}:supervisor.handle_batch{chain=axelar}:supervisor.process_batch{chain=axelar}:worker.packet.cmd{src_chain=axelar src_port=transfer src_channel=channel-0 dst_chain=pocket}:relay{odata=7f01e6e0 ->Destination @0-1563; len=1}: submitted
2025-07-02T09:05:21.473744Z  INFO ThreadId(31) worker.batch{chain=pocket}:supervisor.handle_batch{chain=pocket}:supervisor.process_batch{chain=pocket}:worker.client.misbehaviour{client=07-tendermint-6 src_chain=axelar dst_chain=pocket}:foreign_client.detect_misbehaviour_and_submit_evidence{client=axelar->pocket:07-tendermint-6}:foreign_client.detect_misbehaviour{client=axelar->pocket:07-tendermint-6}: No evidence of misbehavior detected for chain axelar
2025-07-02T09:05:21.573908Z  INFO ThreadId(43) worker.batch{chain=pocket}:supervisor.handle_batch{chain=pocket}:supervisor.process_batch{chain=pocket}:worker.client.misbehaviour{client=07-tendermint-6 src_chain=axelar dst_chain=pocket}:foreign_client.detect_misbehaviour_and_submit_evidence{client=axelar->pocket:07-tendermint-6}: client is valid

# Watch for the the destination account balance to include the new ics20 token
$ watch -n 2 "make ibc_query_axelar_balance"
$ make acc_balance_query ACC=pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4
Every 2,0s: make ibc_query_axelar_balance

balances:
- amount: "1000"
  # NOTE: the denom may be different.
  denom: ibc/43D77F2F7509D83FC29E32D9A00D45E5F1F848F137E59F84873CE83B9FF9B95C
- amount: "9990000000"
  denom: stake
- amount: "9999977141"
  denom: uaxl
pagination:
  next_key: null
  total: "0"
```


### Interchain Accounts

```
// TODO(@bryanchriswhite)
```


## Troubleshooting

### Common Issues

#### 1. Transaction Succeeds but Relayer Doesn't Relay
**Symptoms**: Source chain transaction shows success, but tokens don't appear on destination

**Solutions**:
- Check relayer logs in Tilt UI for error messages
- Restart the Hermes relayer service manually
- Verify RPC endpoints are accessible and returning packet data

#### 2. Account Sequence Mismatch Errors  
**Symptoms**: Relayer shows sequence mismatch errors in logs

**Causes**: 
- Concurrent transactions using the same account
- Account reuse between different services

**Solutions**:
- Use different accounts for testing vs relayer operations
- Wait for sequence number refresh (Hermes auto-retries)
- Reset LocalNet if issues persist

#### 3. IBC Setup Service Stuck
**Symptoms**: `🏗️ Pokt-><counterparty>` service shows persistent backoff

**Solutions**:
- Manually trigger the setup service in Tilt UI
- Check counterparty chain is producing blocks
- Verify network configurations in `localnet_config.yaml`

#### 4. Missing Packet Data Warnings
**Symptoms**: Hermes warns about unavailable packet data

**Solutions**:
- Verify RPC endpoints have required packet data
- Check [Hermes troubleshooting guide](https://hermes.informal.systems/advanced/troubleshooting/cross-comp-config.html#uncleared-pending-packets)
- Ensure proper RPC configuration for data availability

### Reset and Recovery
If issues persist, reset your LocalNet environment:
```bash
make localnet_down
make localnet_up
```

This will clean up any stuck IBC state and restart all services.

