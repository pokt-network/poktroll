// Import necessary modules
import http from 'k6/http';
import { check } from 'k6';
import { Trend } from 'k6/metrics';

// let myRequestTrend = new Trend('sendEthereumRequest')

// Function to send a generic Ethereum JSON-RPC request to Anvil
export function sendEthereumRequest(baseUrl, method, params = [], tags = {}) {
     const payload = JSON.stringify({
        jsonrpc: "2.0",
        method: method,
        params: params,
        id: 1
    });

    const requestOptions = {
        headers: {
            "Content-Type": "application/json",
        },
        tags: tags
    };

    let response = http.post(baseUrl, payload, requestOptions);

    // myRequestTrend.add(response.timings.duration, tags);

    // Basic check for HTTP 200 response
    check(response, {
        "is status 200": (r) => r.status === 200,
        "is successful JSON-RPC response": (r) => {
            let jsonResponse = JSON.parse(r.body);
            // Check for 'result' and 'id', and ensure 'error' is not present
            return jsonResponse.hasOwnProperty("result") && jsonResponse.hasOwnProperty("id") && !jsonResponse.hasOwnProperty("error");
        }
    }, tags);

    return response;
}
