"""
Ethereum JSON-RPC Method Hints

This module provides semantic hints for Ethereum JSON-RPC methods to improve
semantic search accuracy. Hints are alternative phrases, keywords, and common
use cases that help match user queries to the correct RPC method.

How to extend:
--------------
Add new entries to the METHOD_HINTS dictionary using the method name as the key
and a list of hint strings as the value. Hints should include:

1. Alternative phrasings (e.g., "latest block" for eth_blockNumber)
2. Common use cases (e.g., "check account balance" for eth_getBalance)
3. Related technical terms (e.g., "nonce" for eth_getTransactionCount)
4. User-friendly descriptions of what the method does

Example:
    METHOD_HINTS = {
        "eth_blockNumber": [
            "latest block",
            "block height",
            "chain height",
            "current height"
        ],
        "eth_getBalance": [
            "account balance",
            "wallet balance",
            "check balance",
            "how much eth"
        ]
    }
"""

METHOD_HINTS = {
    "eth_blockNumber": [
        "latest block",
        "block height",
        "chain height",
        "current height",
    ],
    "eth_getBlockByNumber": [
        "block details by number",
        "get block by height",
    ],
    "net_version": [
        "network id",
        "chain id (legacy)",
    ],
}
