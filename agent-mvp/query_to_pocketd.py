DEFAULT_PAYLOAD = '{"jsonrpc":"2.0","method":"eth_blockNumber","id":4}'
DEFAULT_NETWORK = "main"
DEFAULT_APP = "6b71374e551fe3c25186f14ebe6185abe72fb5666c9606c52001ce961802548c"

POCKETD_RELAY_CMD = [
    "pocketd",
    "relayminer",
    "relay",
    "--app={}",
    "--network={}",
    "--payload={}",
]
