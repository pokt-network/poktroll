// Our custom handleSummary produces HTML report with CLI output and some default configuration.
export { handleSummary, options } from '../config/index.js';
import { requestBlockNumberEtheriumScenario } from '../scenarios/requestBlockNumberEtherium.js';
import { ENV_CONFIG } from '../config/index.js';
import { sleep } from 'k6';

// TODO(@okdas): expand options to allow multiple stages:
// export const options = {
//     stages: [
//       { duration: '30s', target: 20 },
//       { duration: '1m30s', target: 10 },
//       { duration: '20s', target: 0 },
//     ],
//   };

// The function that defines VU logic.
//
// See https://grafana.com/docs/k6/latest/examples/get-started-with-k6/ to learn more
// about authoring k6 scripts.
export default function () {
    requestBlockNumberEtheriumScenario(ENV_CONFIG.AppGateServerAnvilUrl);

    // Simulate think time
    sleep(1);
}