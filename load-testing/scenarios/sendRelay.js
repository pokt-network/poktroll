import exec from 'k6/execution';
import { sendJsonRPCRequest } from '../modules/sendJsonRPCRequest.js';

import { ENV_CONFIG } from '../config/index.js';

// sendRelay sends a JSON-RPC request to the AppGateServer
export function sendRelay() {
    const method = "eth_blockNumber";
    const params = [];

    sendJsonRPCRequest(ENV_CONFIG.AppGateServerAnvilUrl, method, params);
}