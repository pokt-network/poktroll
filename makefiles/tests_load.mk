
.PHONY: test_load_relays_stress_custom
test_load_relays_stress_custom: ## Run the stress test for E2E relays using custom manifest. "loadtest_manifest_example.yaml" manifest is used by default. Set 'LOAD_TEST_CUSTOM_MANIFEST' environment variable to use the different manifest.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run LoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/$(LOAD_TEST_CUSTOM_MANIFEST)

.PHONY: test_load_relays_stress_localnet_single_supplier
test_load_relays_stress_localnet_single_supplier: test_e2e_env warn_message_local_stress_test ## Run the stress test for E2E relays on LocalNet using exclusively one supplier.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run TestSingleSupplierLoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/loadtest_manifest_localnet_single_supplier.yaml

.PHONY: test_load_relays_stress_localnet_multi_suppliers
test_load_relays_stress_localnet_multi_suppliers: test_e2e_env warn_message_local_stress_test ## Run the stress test for E2E relays on LocalNet using multiple suppliers.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run TestLoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/loadtest_manifest_localnet_multiple_suppliers.yaml

.PHONY test_load_anvil
test_load_anvil: ## Run the stress test for E2E relays on LocalNet using multiple suppliers.
	(cd ./tools/scripts/load_anvil && go run main.go)
