// Import necessary modules
import { sleep, check } from 'k6';
import { sendEthereumRequest } from '../modules/etheriumRequests.js';
import { ENV_CONFIG } from '../config/env.js';

// Anvil direct test scenario
export function anvilDirectScenario() {
    // Example method and parameters for the Ethereum JSON-RPC request
    const method = "eth_blockNumber";
    const params = [];
    const baseUrl = ENV_CONFIG.anvilBaseUrl;

    const tags = {
        method: method,
        baseUrl: baseUrl,
    }

    // Send request and receive response
    let response = sendEthereumRequest(baseUrl, method, params, tags);

    // Additional checks specific to the scenario can be written like that:
    // check(response, {
    //     "is the response format correct": (r) => {
    //         let jsonResponse = JSON.parse(r.body);
    //         return jsonResponse.hasOwnProperty("result") && jsonResponse.hasOwnProperty("id");
    //     },
    // }, tags);

    // Simulate think time
    sleep(1);
}
