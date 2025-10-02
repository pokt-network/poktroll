# Ethereum JSON-RPC Retrieval MVP <!-- omit in toc -->

Local prototype that turns a natural-language Ethereum question into the right JSON-RPC method and a ready-to-run `curl` command—no external services required.

## Quickstart

```bash
make clean        # optional, ensures a fresh env if you ran an older Python
make quickstart   # generates env.sh, installs deps under Python 3.11, builds FAISS index
source env.sh     # export ETH_RPC_URL=https://eth.rpc.grove.city/v1/6c5de5ff
make ask Q="What is the height of the ethereum blockchain?"
```

> `make quickstart` forces the virtualenv to use CPython 3.11 so the `faiss-cpu` wheel resolves. You can override with `PYTHON_VERSION=3.11` (default) or another compatible release.

- [Quickstart](#quickstart)
- [Architecture](#architecture)
- [OpenAPI vs. OpenRPC](#openapi-vs-openrpc)

## Architecture

Everything runs within one Python process.

```mermaid
flowchart TD
    Q[User question] --> Embed[SentenceTransformer\n(all-MiniLM-L6-v2)]
    Embed --> Search[FAISS index]
    Search --> Method[Best JSON-RPC method]
    Method --> Curl[Render curl payload]
    Curl --> Output[Ready-to-run curl command]
```

- `schema_openrpc_min.json`: minimal OpenRPC schema subset (methods + descriptions).
- `build_index.py`: parses the schema, generates embeddings, persists FAISS index + metadata.
- `query_to_curl.py`: loads the index, embeds the question, retrieves the best method, and prints the curl command using `ETH_RPC_URL`.
- `Makefile`: wraps the workflow with `uv`, prepares `env.sh`, and exposes the `quickstart`, `ask`, and `clean` flows.

## OpenAPI vs. OpenRPC

| Aspect             | OpenAPI                                                       | OpenRPC                                             |
| ------------------ | ------------------------------------------------------------- | --------------------------------------------------- |
| Primary use case   | REST/HTTP APIs                                                | JSON-RPC APIs                                       |
| Spec focus         | HTTP verbs, paths, request/response schemas                   | JSON-RPC methods, params, result schemas            |
| Ethereum alignment | Requires custom adaptation                                    | Mirrors EIP-1474 and Ethereum JSON-RPC semantics    |
| Docs               | [OpenAPI Specification](https://spec.openapis.org/oas/v3.1.0) | [OpenRPC Specification](https://spec.open-rpc.org/) |

Ethereum JSON-RPC is natively described via OpenRPC, so this MVP keeps a local OpenRPC snippet that later can be swapped for the full schema.
