import json
import os
import pickle
import sys

import faiss
import numpy as np
from sentence_transformers import SentenceTransformer

INDEX_PATH = "methods.index"
META_PATH = "methods.meta.pkl"
MODEL_NAME = "sentence-transformers/all-MiniLM-L6-v2"

SUPPLIER_URL = os.environ.get("ETH_RPC_URL", "https://YOUR_ETHEREUM_RPC_ENDPOINT")


def load_index():
    index = faiss.read_index(INDEX_PATH)
    with open(META_PATH, "rb") as handle:
        meta = pickle.load(handle)
    return index, meta


def best_match(question, index, docs, model):
    query_embedding = model.encode([question], normalize_embeddings=True).astype(np.float32)
    distances, indices = index.search(query_embedding, 3)
    hits = [(int(idx), float(dist)) for idx, dist in zip(indices[0], distances[0])]
    return hits[0]


def method_to_curl(method_name, params=None, request_id=1):
    if params is None:
        params = []
    payload = {
        "jsonrpc": "2.0",
        "method": method_name,
        "params": params,
        "id": request_id,
    }
    curl = (
        "curl -s -X POST {url} "
        "-H 'Content-Type: application/json' "
        "-d '{payload}'"
    ).format(url=SUPPLIER_URL, payload=json.dumps(payload))
    return curl


def main():
    if len(sys.argv) < 2:
        print("Usage: python query_to_curl.py \"What is the height of the ethereum blockchain?\"")
        sys.exit(1)

    question = sys.argv[1]
    index, meta = load_index()
    methods = meta["methods"]
    docs = meta["docs"]

    model = SentenceTransformer(MODEL_NAME)
    top_idx, score = best_match(question, index, docs, model)
    method = methods[top_idx]
    name = method["name"]

    params = []
    if name == "eth_getBlockByNumber":
        params = ["latest", False]

    if "ETH_RPC_URL" not in os.environ:
        print("# Warning: ETH_RPC_URL not set; using placeholder endpoint")

    print("# Q:", question)
    print("# Picked method:", name, f"(score={score:.3f})")
    print(method_to_curl(name, params=params))


if __name__ == "__main__":
    main()
