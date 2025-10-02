import json
import pickle

import faiss
import numpy as np
from sentence_transformers import SentenceTransformer

# SCHEMA_PATH = "schema_openrpc_min.json"
SCHEMA_PATH = "openrpc_eth.json"
INDEX_PATH = "methods.index"
META_PATH = "methods.meta.pkl"
EMBEDDINGS_MODEL_NAME = "sentence-transformers/all-MiniLM-L6-v2"


def doc_for_method(method):
    """Compose a compact retrieval doc for a single JSON-RPC method."""
    name = method["name"]
    summary = method.get("summary", "")
    description = method.get("description", "")
    params = method.get("params", [])

    hints = []
    if name == "eth_blockNumber":
        hints += ["latest block", "block height", "chain height", "current height"]
    if name == "eth_getBlockByNumber":
        hints += ["block details by number", "get block by height"]
    if name == "net_version":
        hints += ["network id", "chain id (legacy)"]

    param_names = ", ".join([param["name"] for param in params]) if params else "no params"
    text = f"""
Method: {name}
Summary: {summary}
Description: {description}
Parameters: {param_names}
Keywords: {', '.join(hints)}
    """.strip()
    return text


def main():
    with open(SCHEMA_PATH, "r", encoding="utf-8") as handle:
        schema = json.load(handle)

    methods = schema["methods"]
    docs = [doc_for_method(method) for method in methods]

    model = SentenceTransformer(EMBEDDINGS_MODEL_NAME)
    embeddings = model.encode(docs, normalize_embeddings=True)
    dim = embeddings.shape[1]

    index = faiss.IndexFlatIP(dim)
    index.add(embeddings.astype(np.float32))

    with open(META_PATH, "wb") as handle:
        pickle.dump({"methods": methods, "docs": docs}, handle)

    faiss.write_index(index, INDEX_PATH)
    print(f"Indexed {len(docs)} methods → {INDEX_PATH} + {META_PATH}")


if __name__ == "__main__":
    main()
