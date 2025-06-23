#####################
### Documentation ###
#####################

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/poktroll/"
	godoc -http=:6060

.PHONY: docs_update_gov_params_page
docs_update_gov_params_page: ## Update the page in Docusaurus documenting all the governance parameters
	go run tools/scripts/docusaurus/generate_docs_params.go


.PHONY: docusaurus_start
docusaurus_start: check_yarn check_node ## Start the Docusaurus server
	(cd docusaurus && yarn install && yarn start --port 4000)

.PHONY: docusaurus_start_update_plugin
docusaurus_start_update_plugin: ## Update the docusaurus-plugin-chat-page to the latest commit
	@echo "🔄 Fetching latest commit SHA for docusaurus-plugin-chat-page..."
	@SHA=$$(git ls-remote https://github.com/olshansk/docusaurus-plugin-chat-page.git main | cut -f1); \
	echo "📌 Pinning to commit $$SHA"; \
	sed -i.bak -E 's|("docusaurus-plugin-chat-page":\s*)"github:[^"]+"|\1"github:olshansk/docusaurus-plugin-chat-page#'"$$SHA"'"|' docusaurus/package.json; \
	rm -f docusaurus/package.json.bak; \
	cd docusaurus && yarn cache clean docusaurus-plugin-chat-page

.PHONY: docusaurus_chat_yarn_link
docusaurus_chat_yarn_link: ## Yarn link docusaurus-plugin-chat-page
	(cd docusaurus && yarn link docusaurus-plugin-chat-page)

.PHONY: docusaurus_update_dependencies
docusaurus_update_dependencies: ## Force re-fetch of docusaurus-plugin-chat-page from main branch
	@echo "🧼 Cleaning yarn cache for docusaurus-plugin-chat-page..."
	(cd docusaurus && yarn cache clean docusaurus-plugin-chat-page)

	@echo "⬇️  Reinstalling dependencies from latest main..."
	(cd docusaurus && yarn install --force)


.PHONY: clean_embeddings
clean_embeddings: ## Remove embedding cache files from all Docusaurus sites
	@echo "🧹 Cleaning embedding cache files..."
	@find . -name ".docusaurus" -type d -exec find {} -name "*embeddings.json" -delete \; 2>/dev/null || true
	@find . -name ".docusaurus" -type d -exec find {} -name "docusaurus-plugin-chat-page-embeddings.json" -delete \; 2>/dev/null || true
	@echo "✅ Embedding cache files cleaned"

.PHONY: clean_docusaurus_cache
clean_docusaurus_cache: ## Remove all Docusaurus cache directories
	@echo "🧹 Cleaning all Docusaurus cache directories..."
	@find . -name ".docusaurus" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "✅ Docusaurus cache directories cleaned"