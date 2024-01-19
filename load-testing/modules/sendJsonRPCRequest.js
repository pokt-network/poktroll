// Import necessary modules
import http from 'k6/http';
import { check } from 'k6';

// Function to send a generic JSON-RPC request to Anvil
export function sendJsonRPCRequest(baseUrl, method, params = [], tags = {}) {
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

    // Basic check for HTTP 200 OK and a valid JSON-RPC response
    let passed = check(response, {
        "is status 200": (r) => r.status === 200,
        "is successful JSON-RPC response": (r) => {
            let jsonResponse = JSON.parse(r.body);
            // Check for 'result' and 'id', and ensure 'error' is not present
            return jsonResponse.hasOwnProperty("result") && jsonResponse.hasOwnProperty("id") && !jsonResponse.hasOwnProperty("error");
        }
    }, tags);

    if (!passed) {
        // Logging output includes vital information for troubleshooting: request/response body, status code, etc.
        console.log(`Request to ${response.request.url} failed:`, JSON.stringify(response, null, 2));
    }

    return response;
}
