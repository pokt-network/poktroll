// Our custom handleSummary produces HTML report with CLI output and some default configuration.
export { handleSummary, options } from '../config/index.js';
import { requestBlockNumberEthereumScenario } from '../scenarios/requestBlockNumberEthereum.js';
import { ENV_CONFIG } from '../config/index.js';
import { sleep } from 'k6';


// The function that defines VU logic.
//
// See https://grafana.com/docs/k6/latest/examples/get-started-with-k6/ to learn more
// about authoring k6 scripts.
export default function () {
    requestBlockNumberEthereumScenario(ENV_CONFIG.anvilBaseUrl);

    // Simulate think time
    sleep(1);
}