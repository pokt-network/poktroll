# Modular Summit Demo <!-- omit in toc -->

- [Clone the repos](#clone-the-repos)
- [Build dependencies \& run tests](#build-dependencies--run-tests)
- [Run a LocalNet](#run-a-localnet)
  - [Explore \& scale the cluster](#explore--scale-the-cluster)
- [Use the network](#use-the-network)
  - [Send some relays](#send-some-relays)
  - [Run E2E Tests](#run-e2e-tests)
  - [Verify a claim was created](#verify-a-claim-was-created)
- [E2E](#e2e)
  - [E2E Relay Test](#e2e-relay-test)
  - [Side Quest - Makefile](#side-quest---makefile)
  - [E2E - Stress Test](#e2e---stress-test)
- [DevNet](#devnet)
  - [Make a change](#make-a-change)
  - [Trigger a DevNet](#trigger-a-devnet)
  - [Inspect](#inspect)

## Clone the repos

```bash
git clone https://github.com/pokt-network/poktroll.git
```

## Build dependencies & run tests

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

## Run a LocalNet

Start up a new LocalNet:

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

## Use the network

This verifies complete usage

### Send some relays

Send a `JSON RPC` request to an `anvil` node:

```bash
make send_relay_sovereign_app_JSONRPC
```

### Run E2E Tests

Send a `REST` request to an `LLM` node:

```bash
make send_relay_sovereign_app_REST
```

### Verify a claim was created

Go to [relayminer1?term=claim](http://localhost:10350/r/relayminer1/overview?term=claim) and verify that claims were created.

## E2E

### E2E Relay Test

Run the following:

```bash
make test_e2e_relay
```

And note that you were prompted to run this command before re-running.

```bash
make acc_initialize_pubkeys
```

You can find the test in `relay.feature`

### Side Quest - Makefile

Run `make` to see all our helpers.

### E2E - Stress Test

Run the stress test:

```bash
make test_load_relays_stress_localnet
```

Update `localnet_config.yaml`

See the results in [protocol-stress-test](http://localhost:3003/d/ddkakqetrti4gb/protocol-stress-test?orgId=1&refresh=5s).

## DevNet

### Make a change

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
4. Show the GitHub message pop up

Example: https://github.com/pokt-network/poktroll/pull/650

### Inspect

1. GCP
2. Grafana
