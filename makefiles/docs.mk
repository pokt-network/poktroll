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

# Uses https://github.com/PaloAltoNetworks/docusaurus-openapi-docs to generate OpenAPI docs.
# This is a custom plugin for Docusaurus that allows us to embed the OpenAPI spec into the docs.
# Outputs docs files to docusaurus/docs/5_api/*.mdx
.PHONY: docusaurus_gen_api_docs
docusaurus_gen_api_docs: ## Generate docusaurus OpenAPI docs
	(cd docusaurus && yarn install && yarn docusaurus clean-api-docs pocket && yarn docusaurus gen-api-docs pocket)

# High-level entrypoint to (re)generate OpenAPI spec and Docusaurus pages
.PHONY: docs_build
docs_build: ## Generate OpenAPI spec (Docker) and Docusaurus API docs
	$(MAKE) openapi_ignite_gen_docker
	$(MAKE) docusaurus_gen_api_docs

.PHONY: openapi_ignite_gen
openapi_ignite_gen: ignite_check_version ## Generate the OpenAPI spec natively and process the output
	# Ensure Buf deps (incl. grpc-gateway) are available for imports like openapiv2 annotations
	buf dep update
	ignite generate openapi --yes
	$(MAKE) openapi_process

.PHONY: openapi_ignite_gen_docker
openapi_ignite_gen_docker: ## Generate the OpenAPI spec using Docker and process the output; workaround due to https://github.com/ignite/cli/issues/4495
	docker build -f ./proto/Dockerfile.ignite -t ignite-openapi .
	docker run --rm -v "$(PWD):/workspace" ignite-openapi
	$(MAKE) openapi_process


.PHONY: openapi_process
# Internal helper target - Ensure OpenAPI JSON and YAML files are properly formatted
openapi_process: check_yq
# Ignite currently writes JSON content to docs/static/openapi.yml. Create canonical JSON, then YAML from it.
	yq -o=json '.' docs/static/openapi.yml -I=4 > docs/static/openapi.json
	yq -P -o=yaml '.' docs/static/openapi.json > docs/static/openapi.yml
