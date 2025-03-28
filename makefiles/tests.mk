#############
### Tests ###
#############

.PHONY: test_e2e_env
test_e2e_env: warn_message_acc_initialize_pubkeys ## Setup the default env vars for E2E tests
	export POCKET_NODE=$(POCKET_NODE) && \
	export PATH_URL=$(PATH_URL) && \
	export POCKETD_HOME=../../$(POCKETD_HOME)

.PHONY: test_e2e
test_e2e: test_e2e_env ## Run all E2E tests
	go test -count=1 -v ./e2e/tests/... -tags=e2e,test

.PHONY: test_e2e_verbose
test_e2e_verbose: test_e2e_env ## Run all E2E tests with verbose debug output
	E2E_DEBUG_OUTPUT=true go test -count=1 -v ./e2e/tests/... -tags=e2e,test

.PHONY: test_e2e_relay
test_e2e_relay: test_e2e_env ## Run only the E2E suite that exercises the relay life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=relay.feature

.PHONY: test_e2e_app
test_e2e_app: test_e2e_env ## Run only the E2E suite that exercises the application life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_app.feature

.PHONY: test_e2e_supplier
test_e2e_supplier: test_e2e_env ## Run only the E2E suite that exercises the supplier life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_supplier.feature

.PHONY: test_e2e_gateway
test_e2e_gateway: test_e2e_env ## Run only the E2E suite that exercises the gateway life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_gateway.feature

.PHONY: test_e2e_session
test_e2e_session: test_e2e_env ## Run only the E2E suite that exercises the session (i.e. claim/proof) life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=session.feature

.PHONY: test_e2e_tokenomics
test_e2e_tokenomics: test_e2e_env ## Run only the E2E suite that exercises the session & tokenomics settlement
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=0_settlement.feature

.PHONY: test_e2e_params
test_e2e_params: test_e2e_env ## Run only the E2E suite that exercises parameter updates for all modules
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=update_params.feature

.PHONY: test_e2e_migration
test_e2e_migration: test_e2e_env ## Run only the E2E suite that exercises the migration module
	go test -v ./e2e/tests/migration_steps_test.go -tags=e2e,manual

.PHONY: test_load_relays_stress_custom
test_load_relays_stress_custom: ## Run the stress test for E2E relays using custom manifest. "loadtest_manifest_example.yaml" manifest is used by default. Set `LOAD_TEST_CUSTOM_MANIFEST` environment variable to use the different manifest.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run LoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/$(LOAD_TEST_CUSTOM_MANIFEST)

.PHONY: test_load_relays_stress_localnet
test_load_relays_stress_localnet: test_e2e_env warn_message_local_stress_test ## Run the stress test for E2E relays on LocalNet.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run TestLoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/loadtest_manifest_localnet.yaml

.PHONY: test_load_relays_stress_localnet_single_supplier
test_load_relays_stress_localnet_single_supplier: test_e2e_env warn_message_local_stress_test ## Run the stress test for E2E relays on LocalNet using exclusively one supplier.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run TestSingleSupplierLoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/loadtest_manifest_localnet_single_supplier.yaml

.PHONY: test_verbose
test_verbose: check_go_version ## Run all go tests verbosely
	go test -count=1 -v -race -tags test ./...

# NB: buildmode=pie is necessary to avoid linker errors on macOS.
# It is not compatible with `-race`, which is why it's omitted here.
# See ref for more details: https://github.com/golang/go/issues/54482#issuecomment-1251124908
.PHONY: test_all
test_all: warn_flaky_tests check_go_version test_gen_fixtures ## Run all go tests showing detailed output only on failures
	go test -count=1 -buildmode=pie -tags test ./...

.PHONY: test_all_with_integration
test_all_with_integration: check_go_version ## Run all go tests, including those with the integration
	go test -count=1 -v -race -tags test,integration ./...

# We are explicitly using an env variable rather than a build tag to keep flaky
# tests in line with non flaky tests and use it as a way to easily turn them
# on and off without maintaining extra files.
.PHONY: test_all_with_integration_and_flaky
test_all_with_integration_and_flaky: check_go_version ## Run all go tests, including those with the integration and flaky tests
	INCLUDE_FLAKY_TESTS=true go test -count=1 -v -race -tags test,integration ./...

.PHONY: test_integration
test_integration: check_go_version ## Run only the in-memory integration "unit" tests
	go test -count=1 -v -race -tags test,integration ./tests/integration/...

.PHONY: itest
itest: check_go_version ## Run tests iteratively (see usage for more)
	./tools/scripts/itest.sh $(filter-out $@,$(MAKECMDGOALS))

# Verify there are no compile errors in pkg/relayer/miner/gen directory.
# This is done by overriding the build tags through passing a `*.go` argument to `go test`.
# .go files in that directory contain a `go:build ignore` directive to avoid introducing
# unintentional churn in the randomly generated fixtures.
.PHONY: test_gen_fixtures
test_gen_fixtures: check_go_version ## Run all go tests verbosely
	go test -count=1 -v -race ./pkg/relayer/miner/gen/*.go
