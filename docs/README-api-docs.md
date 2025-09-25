API Docs Generation (OpenAPI + Docusaurus)

Overview

- Source of truth: `.proto` files with `openapiv2_operation` annotations for clean `operation_id` and `summary`.
- OpenAPI generator: `protoc-gen-openapiv2` via Ignite/Docker, configured by `proto/buf.gen.swagger.yaml`.
- Docs site generator: `docusaurus-plugin-openapi-docs` (outputs MDX under `docusaurus/docs/5_api`).

What changed

- Deleted manual post-processing scripts under `scripts/` to remove brittle steps.
- Added and refined operation titles/summaries in proto services (Msg and Query RPCs).
- Updated `proto/buf.gen.swagger.yaml` options to improve grouping and naming.
- Added `docs_build` Make target to chain OpenAPI generation and Docusaurus generation.

Prerequisites

- Docker running (used by `openapi_ignite_gen_docker`).
- Node.js and Yarn for Docusaurus (see `docusaurus/package.json`).

Main commands

- Regenerate OpenAPI + MDX docs: `make docs_build`
- Preview docs locally: `make docusaurus_start` (serves at http://localhost:4000)

Outputs

- OpenAPI spec: `docs/static/openapi.yml` (and `openapi.json`)
- Generated MDX: `docusaurus/docs/5_api/`

Notes

- `docusaurus_gen_api_docs` runs `clean-api-docs` which clears previously generated files for the `pocket` spec. Avoid placing hand-authored files inside `docusaurus/docs/5_api`.
- If Docker is unavailable, use `make openapi_ignite_gen` to invoke the native Ignite generator, then `make docusaurus_gen_api_docs`.
- If Docusaurus install needs network access, run `yarn install` inside `docusaurus/` first.

