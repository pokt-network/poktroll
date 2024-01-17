// Import necessary modules
import { check } from 'k6';
import { sendJsonRPCRequest } from '../modules/jsonRpc.js';

export function requestBlockNumberEtheriumScenario(baseUrl) {
    // Example method and parameters for the Ethereum JSON-RPC request
    const method = "eth_blockNumber";
    const params = [];

    // We can populate tags in addition to the defaults.
    // const tags = {
    //     method: method,
    //     baseUrl: baseUrl,
    // }

    // Send request and receive response
    let response = sendJsonRPCRequest(baseUrl, method, params);

    // Additional checks specific to the scenario can be written like that:
    // check(response, {
    //     "is the response format correct": (r) => {
    //         let jsonResponse = JSON.parse(r.body);
    //         return jsonResponse.hasOwnProperty("result") && jsonResponse.hasOwnProperty("id");
    //     },
    // }, tags);
}
