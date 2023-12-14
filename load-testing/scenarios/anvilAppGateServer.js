// Import necessary modules
import { sleep, check } from 'k6';
import { sendEthereumRequest } from '../modules/etheriumRequests.js';
import { ENV_CONFIG } from '../config/env.js';

// Anvil through App Gate Server test scenario
export function anvilAppGateServerScenario() {
    // Example method and parameters for the Ethereum JSON-RPC request
    const method = "eth_blockNumber";
    const params = [];
    const baseUrl = ENV_CONFIG.AppGateServerAnvilUrl;

    const tags = {
        method: method,
        baseUrl: baseUrl,
    }

    // Send request and receive response
    let response = sendEthereumRequest(baseUrl, method, params);

    // Additional checks specific to the scenario
    check(response, {
        "is the response format correct": (r) => {
            let jsonResponse = JSON.parse(r.body);
            return jsonResponse.hasOwnProperty("result") && jsonResponse.hasOwnProperty("id");
        },
    }, tags);

    // Simulate think time
    sleep(1);
}
