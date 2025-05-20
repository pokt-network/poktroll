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
