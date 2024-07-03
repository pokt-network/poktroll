# Modular Summit Demo <!-- omit in toc -->

- [Clone the poktroll repos](#clone-the-poktroll-repos)
- [Build deps \& run tests](#build-deps--run-tests)
- [LocalNet](#localnet)
  - [Explore \& scale the cluster](#explore--scale-the-cluster)
- [Real world E2E usage](#real-world-e2e-usage)
  - [Send a JSON RPC relay to an anvil node](#send-a-json-rpc-relay-to-an-anvil-node)
  - [Send a REST relay to an LLM](#send-a-rest-relay-to-an-llm)
  - [Verify claim creation \& settlement](#verify-claim-creation--settlement)
- [E2E Tests](#e2e-tests)
  - [E2E relay test](#e2e-relay-test)
  - [Side Quest - Makefile](#side-quest---makefile)
  - [E2E - stress test](#e2e---stress-test)
- [DevNet + GitHub PR](#devnet--github-pr)
  - [Make a code change](#make-a-code-change)
  - [Trigger a DevNet](#trigger-a-devnet)
  - [Inspect](#inspect)

## Clone the poktroll repos

```bash
git clone https://github.com/pokt-network/poktroll.git
```

## Build deps & run tests

Change directory:

```bash
cd ~/workspace/pocket/poktroll2
```

Review all of the available commands:

```bash
make
```

Build tests and run unit tests:

```bash
make go_develop_and_test
```

This builds everything and runs all our unit & integration (not E2E) tets.

## LocalNet

Start up a new LocalNet in a different directory:

```bash
cd ~/workspace/pocket/poktroll
make localnet_up
```

### Explore & scale the cluster

Go to [localhost:10350](http://localhost:10350/r/validator/overview) and:

- Inspect all the actors on the left hand side
- Note how the grafana dashboard is easily accessible for each one

Open `localnet_config.yaml` and:

- Scale the cluster by updating `1` to `2` in .
- Go back to [localhost:10350](http://localhost:10350/r/validator/overview) and
  see the number of actors scale.

## Real world E2E usage

At this point we have a complete network running using all of the actors we need.

See the related slides about the actor overview.

Let's interact with the network in an end-to-end fashion.

### Send a JSON RPC relay to an anvil node

Send a `JSON RPC` request to an `anvil` node:

```bash
make send_relay_sovereign_app_JSONRPC
```

### Send a REST relay to an LLM

Send a `REST` request to an `LLM` node:

```bash
make send_relay_sovereign_app_REST
```

Look in the `Makefile` to see what this command does.

### Verify claim creation & settlement

Go to [relayminer1?term=claim](http://localhost:10350/r/relayminer1/overview?term=claim) and verify that claims were created.

Go to [validator/overview?term=settled](http://localhost:10350/r/validator/overview?term=settled) and verify that the claims were settled.

## E2E Tests

### E2E relay test

Run the following command to validate the E2E relay test.

```bash
make test_e2e_relay
```

And note that you were prompted to run this command before re-running.

```bash
make acc_initialize_pubkeys
```

The full test can be found in `relay.feature`

### Side Quest - Makefile

Run `make` to see all our helpers.

### E2E - stress test

Run the stress test:

```bash
make test_load_relays_stress_localnet
```

Note the warning message, so make sure to update the `localnet_config.yaml` file.

See the results in [protocol-stress-test](http://localhost:3003/d/ddkakqetrti4gb/protocol-stress-test?orgId=1&refresh=5s).

## DevNet + GitHub PR

### Make a code change

Going to add `modular summit` to a log line in the validator.

1. Go to `settle_pending_claims.go` and search for `settled %d`...
2. Add `live at modular summit!` to the log line.
3. Go back to [localhost:10350](http://localhost:10350/r/validator/overview?term=claim) and see things hot reload
4. Run `make send_relay_sovereign_app_JSONRPC`
5. Search for `modular summit` in the logs [here](http://localhost:10350/r/validator/overview?term=modular+summ)

### Trigger a DevNet

1. Go to the PR
2. Add the `devnet-test-e2e` label
3. Show the other labels automatically added
4. Run `make trigger_ci`
5. Show the GitHub message pop up

Example: https://github.com/pokt-network/poktroll/pull/650

### Inspect

1. GCP link
2. Grafana link
