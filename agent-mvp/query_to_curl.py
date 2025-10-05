import json
import os
import sys

import chromadb
import ollama
from dotenv import load_dotenv
from sentence_transformers import SentenceTransformer

# Load environment variables from .env
load_dotenv()

CHROMA_PATH = "./chroma_data"
METHODS_COLLECTION = "eth-rpc-methods"
SCHEMAS_COLLECTION = "eth-rpc-schemas"
DESCRIPTORS_COLLECTION = "eth-rpc-descriptors"
MODEL_NAME = "sentence-transformers/all-MiniLM-L6-v2"
OLLAMA_MODEL = "qwen2.5-coder:1.5b"
PROMPT_TEMPLATE_PATH = "llm_prompt_template.txt"

# Build Grove URL from GROVE_APP_ID
GROVE_APP_ID = os.environ.get("GROVE_APP_ID")
if GROVE_APP_ID:
    SUPPLIER_URL = f"https://eth.rpc.grove.city/v1/{GROVE_APP_ID}"
else:
    SUPPLIER_URL = "https://YOUR_ETHEREUM_RPC_ENDPOINT"


def load_collections():
    client = chromadb.PersistentClient(path=CHROMA_PATH)
    methods_col = client.get_collection(name=METHODS_COLLECTION)
    schemas_col = client.get_collection(name=SCHEMAS_COLLECTION)
    descriptors_col = client.get_collection(name=DESCRIPTORS_COLLECTION)
    return methods_col, schemas_col, descriptors_col


def find_best_method(question, methods_col, model):
    """Find best matching method using cosine similarity search."""
    query_embedding = model.encode([question], normalize_embeddings=True).tolist()
    results = methods_col.query(query_embeddings=query_embedding, n_results=1)

    method_name = results["ids"][0][0]
    doc = results["documents"][0][0]
    metadata = results["metadatas"][0][0]
    distance = results["distances"][0][0]
    score = 1 - distance

    return {
        "name": method_name,
        "doc": doc,
        "metadata": metadata,
        "score": score,
    }


def lookup_schemas_and_descriptors(schema_keys, descriptor_keys, schemas_col, descriptors_col):
    """Lookup schemas and descriptors by key."""
    schemas = []
    descriptors = []

    # Lookup schemas
    if schema_keys:
        try:
            schema_results = schemas_col.get(ids=schema_keys)
            for i, key in enumerate(schema_results["ids"]):
                schemas.append({
                    "key": key,
                    "doc": schema_results["documents"][i],
                    "metadata": schema_results["metadatas"][i],
                })
        except Exception as e:
            print(f"# Warning: Could not fetch schemas: {e}", file=sys.stderr)

    # Lookup descriptors
    if descriptor_keys:
        try:
            desc_results = descriptors_col.get(ids=descriptor_keys)
            for i, key in enumerate(desc_results["ids"]):
                descriptors.append({
                    "key": key,
                    "doc": desc_results["documents"][i],
                    "metadata": desc_results["metadatas"][i],
                })
        except Exception as e:
            print(f"# Warning: Could not fetch descriptors: {e}", file=sys.stderr)

    return schemas, descriptors


def check_ollama_available():
    """Check if Ollama is available and has the required model."""
    try:
        # Check if Ollama is running
        models = ollama.list()
        model_names = [model["name"] for model in models.get("models", [])]

        if OLLAMA_MODEL in model_names:
            return True
        else:
            print(f"# Ollama is running but {OLLAMA_MODEL} is not available", file=sys.stderr)
            print(f"# Please run: ollama run {OLLAMA_MODEL}", file=sys.stderr)
            return False
    except Exception as e:
        print(f"# Ollama is not available: {e}", file=sys.stderr)
        print(f"# Please run: ollama run {OLLAMA_MODEL}", file=sys.stderr)
        return False


def generate_params_with_llm(question, method_name, method_doc, schemas, descriptors):
    """Use local LLM to generate JSON-RPC params based on question and context."""
    # Load prompt template
    try:
        with open(PROMPT_TEMPLATE_PATH, "r", encoding="utf-8") as f:
            template = f.read()
    except FileNotFoundError:
        print(f"# Warning: Prompt template not found at {PROMPT_TEMPLATE_PATH}", file=sys.stderr)
        return []

    # Build schemas section
    schemas_section = ""
    if schemas:
        schemas_section = "## Related Schemas\n"
        for schema in schemas:
            schemas_section += f"\n{schema['doc']}\n"

    # Build descriptors section
    descriptors_section = ""
    if descriptors:
        descriptors_section = "## Related Content Descriptors\n"
        for desc in descriptors:
            descriptors_section += f"\n{desc['doc']}\n"

    # Fill in template
    prompt = template.format(
        question=question,
        method_name=method_name,
        method_doc=method_doc,
        schemas_section=schemas_section,
        descriptors_section=descriptors_section,
    )

    try:
        response = ollama.generate(model=OLLAMA_MODEL, prompt=prompt)
        params_str = response["response"].strip()

        # Try to parse as JSON
        params = json.loads(params_str)
        return params
    except Exception as e:
        print(f"# Warning: LLM generation failed: {e}", file=sys.stderr)
        return []


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
    methods_col, schemas_col, descriptors_col = load_collections()
    model = SentenceTransformer(MODEL_NAME)

    # Step 1: Find best method using cosine similarity
    method_match = find_best_method(question, methods_col, model)
    method_name = method_match["name"]
    score = method_match["score"]

    print("# Q:", question)
    print(f"# Best method: {method_name} (score={score:.3f})")

    # Step 2: Extract schema/descriptor keys from method metadata (if available)
    # TODO: Parse method params/result to extract schema references
    # For now, hardcode some examples for demonstration
    schema_keys = []
    descriptor_keys = []

    if method_name == "eth_getBlockByNumber":
        schema_keys = ["Block"]
        descriptor_keys = ["Block", "BlockNumber"]

    # Step 3: Lookup schemas and descriptors by key
    schemas, descriptors = lookup_schemas_and_descriptors(
        schema_keys, descriptor_keys, schemas_col, descriptors_col
    )

    if schemas:
        print("# Related schemas:")
        for schema in schemas:
            print(f"#   - {schema['key']}")

    if descriptors:
        print("# Related descriptors:")
        for desc in descriptors:
            print(f"#   - {desc['key']}")

    print()

    # Step 4: Feed method + schemas + descriptors to local LLM to formulate request
    params = []
    if check_ollama_available():
        print("# Using LLM to generate params...")
        params = generate_params_with_llm(
            question, method_name, method_match["doc"], schemas, descriptors
        )
        print(f"# Generated params: {json.dumps(params)}")
    else:
        # Fallback to hardcoded params
        print("# Using fallback hardcoded params")
        if method_name == "eth_getBlockByNumber":
            params = ["latest", False]

    if not GROVE_APP_ID:
        print("# Warning: GROVE_APP_ID not set in .env; using placeholder endpoint")

    print()
    print(method_to_curl(method_name, params=params))


if __name__ == "__main__":
    main()
