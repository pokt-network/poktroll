import json
from dataclasses import dataclass
from typing import List

import chromadb
from sentence_transformers import SentenceTransformer

from eth_hints import METHOD_HINTS

SCHEMA_PATH = "openrpc_eth.json"
CHROMA_PATH = "./chroma_data"
METHODS_COLLECTION = "eth-rpc-methods"
SCHEMAS_COLLECTION = "eth-rpc-schemas"
DESCRIPTORS_COLLECTION = "eth-rpc-descriptors"
EMBEDDINGS_MODEL_NAME = "sentence-transformers/all-MiniLM-L6-v2"


@dataclass
class MethodJSONRPC:
    """Represents a JSON-RPC method with semantic search metadata."""

    name: str
    summary: str
    description: str
    params: List[dict]
    hints: List[str]
    item_type: str = "method"

    @property
    def param_names(self) -> str:
        """Get comma-separated parameter names or 'no params'."""
        if not self.params:
            return "no params"
        return ", ".join([param["name"] for param in self.params])

    def to_document(self) -> str:
        """Convert to search document text."""
        return f"""
Method: {self.name}
Summary: {self.summary}
Description: {self.description}
Parameters: {self.param_names}
Keywords: {', '.join(self.hints)}
        """.strip()

    @classmethod
    def from_schema(cls, method: dict) -> "MethodJSONRPC":
        """Create from OpenRPC schema method definition."""
        name = method["name"]
        return cls(
            name=name,
            summary=method.get("summary", ""),
            description=method.get("description", ""),
            params=method.get("params", []),
            hints=METHOD_HINTS.get(name, []),
        )


@dataclass
class SchemaComponent:
    """Represents a schema component from OpenRPC."""

    name: str
    title: str
    description: str
    schema_type: str
    item_type: str = "schema"

    def to_document(self) -> str:
        """Convert to search document text."""
        return f"""
Schema: {self.name}
Title: {self.title}
Type: {self.schema_type}
Description: {self.description}
        """.strip()

    @classmethod
    def from_schema(cls, name: str, schema: dict) -> "SchemaComponent":
        """Create from OpenRPC schema definition."""
        return cls(
            name=name,
            title=schema.get("title", name),
            description=schema.get("description", ""),
            schema_type=schema.get("type", "object"),
        )


@dataclass
class ContentDescriptor:
    """Represents a content descriptor from OpenRPC."""

    name: str
    summary: str
    description: str
    item_type: str = "contentDescriptor"

    def to_document(self) -> str:
        """Convert to search document text."""
        return f"""
ContentDescriptor: {self.name}
Summary: {self.summary}
Description: {self.description}
        """.strip()

    @classmethod
    def from_schema(cls, name: str, descriptor: dict) -> "ContentDescriptor":
        """Create from OpenRPC content descriptor definition."""
        return cls(
            name=descriptor.get("name", name),
            summary=descriptor.get("summary", ""),
            description=descriptor.get("description", ""),
        )


def main():
    with open(SCHEMA_PATH, "r", encoding="utf-8") as handle:
        spec = json.load(handle)

    client = chromadb.PersistentClient(path=CHROMA_PATH)
    model = SentenceTransformer(EMBEDDINGS_MODEL_NAME)

    # Index methods with embeddings for similarity search
    rpc_methods = [MethodJSONRPC.from_schema(method) for method in spec["methods"]]
    method_docs = [method.to_document() for method in rpc_methods]
    method_embeddings = model.encode(method_docs, normalize_embeddings=True).tolist()

    methods_col = client.get_or_create_collection(
        name=METHODS_COLLECTION, metadata={"hnsw:space": "cosine"}
    )

    methods_col.upsert(
        ids=[method.name for method in rpc_methods],
        documents=method_docs,
        embeddings=method_embeddings,
        metadatas=[{"params": json.dumps(method.params)} for method in rpc_methods],
    )

    # Index schemas by key (no embeddings needed, just key-value storage)
    schemas = []
    if "components" in spec and "schemas" in spec["components"]:
        schemas = [
            SchemaComponent.from_schema(name, schema)
            for name, schema in spec["components"]["schemas"].items()
        ]

    if schemas:
        schema_docs = [schema.to_document() for schema in schemas]
        # Use dummy embeddings (zeros) since we only need key-value lookup
        dummy_embeddings = [[0.0] * 384 for _ in schemas]  # 384 is the dimension of all-MiniLM-L6-v2

        schemas_col = client.get_or_create_collection(
            name=SCHEMAS_COLLECTION, metadata={"hnsw:space": "cosine"}
        )

        schemas_col.upsert(
            ids=[schema.name for schema in schemas],
            documents=schema_docs,
            embeddings=dummy_embeddings,
            metadatas=[{"schema_type": schema.schema_type, "title": schema.title} for schema in schemas],
        )

    # Index content descriptors by key (no embeddings needed)
    content_descriptors = []
    if "components" in spec and "contentDescriptors" in spec["components"]:
        content_descriptors = [
            ContentDescriptor.from_schema(name, descriptor)
            for name, descriptor in spec["components"]["contentDescriptors"].items()
        ]

    if content_descriptors:
        descriptor_docs = [desc.to_document() for desc in content_descriptors]
        # Use dummy embeddings (zeros)
        dummy_embeddings = [[0.0] * 384 for _ in content_descriptors]

        descriptors_col = client.get_or_create_collection(
            name=DESCRIPTORS_COLLECTION, metadata={"hnsw:space": "cosine"}
        )

        descriptors_col.upsert(
            ids=[desc.name for desc in content_descriptors],
            documents=descriptor_docs,
            embeddings=dummy_embeddings,
            metadatas=[{"summary": desc.summary} for desc in content_descriptors],
        )

    print(
        f"Indexed {len(rpc_methods)} methods, {len(schemas)} schemas, {len(content_descriptors)} content descriptors → {CHROMA_PATH}"
    )


if __name__ == "__main__":
    main()
