#!/bin/bash
# Fix the http-api-console.info.mdx file by replacing the problematic useCurrentSidebarCategory usage
# with a functional API endpoints list.

set -euo pipefail

API_CONSOLE_FILE="docusaurus/docs/5_api/http-api-console.info.mdx"

if [ ! -f "$API_CONSOLE_FILE" ]; then
    echo "API console file not found: $API_CONSOLE_FILE"
    exit 1
fi

echo "Fixing API console page: $API_CONSOLE_FILE"

# Create a backup
cp "$API_CONSOLE_FILE" "$API_CONSOLE_FILE.backup"

# Replace the problematic DocCardList code with a functional API endpoints listing
sed -i.tmp '
/```mdx-code-block/,/```/ {
    /```mdx-code-block/c\
## API Endpoints\
\
The following API endpoints are available:\
\
### Message Endpoints\
- [MsgAddService](./msg-add-service.api.mdx) - Add a new service to the network\
- [MsgStakeApplication](./msg-stake-application.api.mdx) - Stake an application\
- [MsgStakeGateway](./msg-stake-gateway.api.mdx) - Stake a gateway\
- [MsgStakeSupplier](./msg-stake-supplier.api.mdx) - Stake a supplier\
- [MsgDelegateToGateway](./msg-delegate-to-gateway.api.mdx) - Delegate to a gateway\
- [MsgUndelegateFromGateway](./msg-undelegate-from-gateway.api.mdx) - Undelegate from a gateway\
- [MsgUnstakeApplication](./msg-unstake-application.api.mdx) - Unstake an application\
- [MsgUnstakeGateway](./msg-unstake-gateway.api.mdx) - Unstake a gateway\
- [MsgUnstakeSupplier](./msg-unstake-supplier.api.mdx) - Unstake a supplier\
- [MsgTransferApplication](./msg-transfer-application.api.mdx) - Transfer an application\
- [MsgCreateClaim](./msg-create-claim.api.mdx) - Create a claim\
- [MsgSubmitProof](./msg-submit-proof.api.mdx) - Submit a proof\
\
### Query Endpoints\
- [QueryAllApplications](./query-all-applications.api.mdx) - Get all applications\
- [QueryApplication](./query-application.api.mdx) - Get a specific application\
- [QueryAllGateways](./query-all-gateways.api.mdx) - Get all gateways\
- [QueryGateway](./query-gateway.api.mdx) - Get a specific gateway\
- [QueryAllSuppliers](./query-all-suppliers.api.mdx) - Get all suppliers\
- [QuerySupplier](./query-supplier.api.mdx) - Get a specific supplier\
- [QueryAllServices](./query-all-services.api.mdx) - Get all services\
- [QueryService](./query-service.api.mdx) - Get a specific service\
- [QueryAllClaims](./query-all-claims.api.mdx) - Get all claims\
- [QueryClaim](./query-claim.api.mdx) - Get a specific claim\
- [QueryAllProofs](./query-all-proofs.api.mdx) - Get all proofs\
- [QueryProof](./query-proof.api.mdx) - Get a specific proof\
- [QueryGetSession](./query-get-session.api.mdx) - Get session information\
\
> **Note**: All endpoints support interactive testing through the API explorer on each individual page.
    /import DocCardList/d
    /import {useCurrentSidebarCategory}/d
    /^$/d
    /<DocCardList/d
    /```$/d
}' "$API_CONSOLE_FILE"

# Clean up temporary files
rm -f "$API_CONSOLE_FILE.tmp" "$API_CONSOLE_FILE.backup"

echo "API console page fixed with functional endpoint list"