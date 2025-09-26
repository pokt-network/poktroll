#####################
### Documentation ###
#####################

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/poktroll/"
	godoc -http=:6060

.PHONY: docusaurus_start
docusaurus_start: check_yarn check_node ## Start the Docusaurus server
	(cd docusaurus && yarn install && yarn start --port 4000)

.PHONY: docs_update_gov_params_page
docs_update_gov_params_page: ## Update the page in Docusaurus documenting all the governance parameters
	go run tools/scripts/docusaurus/generate_docs_params.go

################
### OpenAPI ###
################

.PHONY: openapi_ignite_gen
openapi_ignite_gen: ignite_check_version ## Generate OpenAPI spec natively and process output
	@ignite generate openapi --yes
	@$(MAKE) process_openapi

.PHONY: openapi_ignite_gen_docker
openapi_ignite_gen_docker: ## Generate OpenAPI spec using Docker (workaround for ignite/cli#4495)
	@docker build -f ./proto/Dockerfile.ignite -t ignite-openapi .
	@docker run --rm -v "$(PWD):/workspace" ignite-openapi
	@$(MAKE) process_openapi

.PHONY: process_openapi
# Internal helper: Process OpenAPI output to proper JSON/YAML format
process_openapi:
	@# Fix incorrectly named .yml file that contains JSON
	@mv docs/static/openapi.yml docs/static/openapi.json
	@yq -o=json '.' docs/static/openapi.json -I=4 > docs/static/openapi.json.tmp && mv docs/static/openapi.json.tmp docs/static/openapi.json
	@yq -P -o=yaml '.' docs/static/openapi.json > docs/static/openapi.yml