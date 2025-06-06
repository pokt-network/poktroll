#!/usr/bin/env bash

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> (e.g. v0.1.17)"
    exit 1
fi

NETWORKS=("alpha" "beta" "main")
NETWORK_NAMES=("Alpha TestNet" "Beta TestNet" "MainNet")
HEIGHTS=()
HASHES=()

for i in "${!NETWORKS[@]}"; do
    NET="${NETWORKS[$i]}"
    NAME="${NETWORK_NAMES[$i]}"

    # Step 1: Get upgrade height
    HEIGHT_JSON=$(pocketd query upgrade applied "$VERSION" --network="$NET" -o json 2>/dev/null)

    if ! echo "$HEIGHT_JSON" | jq empty 2>/dev/null; then
        echo "[WARN] No valid upgrade height JSON for $NET"
        HEIGHTS+=("—")
        HASHES+=("—")
        continue
    fi

    HEIGHT=$(echo "$HEIGHT_JSON" | jq -r '.height')

    if [ -z "$HEIGHT" ] || [ "$HEIGHT" == "null" ]; then
        echo "[WARN] No upgrade height found for $NET"
        HEIGHTS+=("—")
        HASHES+=("—")
        continue
    fi

    # Step 2: Get block at upgrade height
    BLOCK_JSON=$(pocketd query block --type=height "$HEIGHT" --network="$NET" -o json 2>/dev/null)

    if ! echo "$BLOCK_JSON" | jq empty 2>/dev/null; then
        echo "[WARN] No valid block JSON at height $HEIGHT for $NET"
        HEIGHTS+=("$HEIGHT")
        HASHES+=("—")
        continue
    fi

    TX_HASH=$(echo "$BLOCK_JSON" | jq -r '.data.txs[0] // "—"')

    HEIGHTS+=("$HEIGHT")
    HASHES+=("$TX_HASH")
done

# Output Markdown table
echo '````markdown'
echo "## Protocol Upgrades"
echo ""
echo "| Category                     | Applicable | Notes                                |"
echo "| ---------------------------- | ---------- | ------------------------------------ |"
echo "| Planned Upgrade              | ❌         | Non-deterministic bug fixes          |"
echo "| Consensus Breaking Change    | ✅         | Yes.                                 |"
echo "| Manual Intervention Required | ⚠️ & ✅     |                                      |"
echo ""
echo "| Network       | Upgrade Height | Upgrade Transaction Hash                        | Notes |"
echo "| ------------- | -------------- | ----------------------------------------------- | ----- |"

for i in "${!NETWORKS[@]}"; do
    echo "| ${NETWORK_NAMES[$i]} | ${HEIGHTS[$i]} | ${HASHES[$i]} |      |"
done

echo '````'
