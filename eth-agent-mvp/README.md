# Ethereum JSON-RPC Retrieval MVP

Local prototype that turns a natural-language Ethereum question into the right JSON-RPC method and a ready-to-run `curl` command—no external services required.

## Quickstart

```bash
make quickstart    # generates env.sh, installs deps, builds FAISS index
source env.sh      # export ETH_RPC_URL=https://eth.rpc.grove.city/v1/6c5de5ff
make ask Q="What is the height of the ethereum blockchain?"
```

## Architecture

The project stays deliberately small so everything runs inside one Python process.

```mermaid
flowchart TD
    Q[User question] --> Embed[SentenceTransformer\n(all-MiniLM-L6-v2)]
    Embed --> Search[FAISS index]
    Search --> Method[Best JSON-RPC method]
    Method --> Curl[Render curl payload]
    Curl --> Output[Ready-to-run curl command]
```

- `schema_openrpc_min.json`: Minimal OpenRPC schema subset (methods + descriptions).
- `build_index.py`: Parses the schema, generates embeddings, persists FAISS index + metadata.
- `query_to_curl.py`: Loads index, embeds the question, retrieves the best method, and prints the curl command using `ETH_RPC_URL`.
- `Makefile`: Wraps the workflow with `uv`, prepares `env.sh`, and exposes the `quickstart`, `ask`, and `clean` flows.

## OpenAPI vs. OpenRPC

| Aspect | OpenAPI | OpenRPC |
| --- | --- | --- |
| Primary use case | REST/HTTP APIs | JSON-RPC APIs |
| Spec focus | HTTP verbs, paths, request/response schemas | JSON-RPC methods, params, result schemas |
| Ethereum alignment | Requires custom adaptation | Mirrors EIP-1474 and Ethereum JSON-RPC semantics |
| Docs | [OpenAPI Specification](https://spec.openapis.org/oas/v3.1.0) | [OpenRPC Specification](https://spec.open-rpc.org/) |

Ethereum JSON-RPC is natively described via OpenRPC, so this MVP keeps a local OpenRPC snippet that can later be swapped for the full schema.
