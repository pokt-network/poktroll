import { ENV_CONFIG } from '../config/index.js';
import {
  buildRelayScenarioStages,
  stakingIterationsCount,
  actorStakeTotalDuration,
} from "../modules/scenarioStagesBuilder.js";

// Export the scenario execution functions.
export { stakeSuppliers } from "../scenarios/stakeSupplier.js";
export { stakeGateways } from "../scenarios/stakeGateway.js";
export { stakeApplications } from "../scenarios/stakeApplication.js";
export { sendRelay } from "../scenarios/sendRelay.js";

export const options = {
  scenarios: {
    // Suppliers staking scenario
    suppliers: {
      // Use per-vu-iterations executor to stake the exact number of suppliers.
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: stakingIterationsCount(ENV_CONFIG.supplier),
      maxDuration: `${actorStakeTotalDuration(ENV_CONFIG.supplier)}s`,
      exec: 'stakeSuppliers',
    },
    // Gateways staking scenario
    gateways: {
      // Use per-vu-iterations executor to stake the exact number of gateways.
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: stakingIterationsCount(ENV_CONFIG.gateway),
      maxDuration: `${actorStakeTotalDuration(ENV_CONFIG.gateway)}s`,
      exec: 'stakeGateways',
    },
    // Applications staking scenario
    applications: {
      // Use per-vu-iterations executor to stake the exact number of applications.
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: stakingIterationsCount(ENV_CONFIG.application),
      maxDuration: `${actorStakeTotalDuration(ENV_CONFIG.application)}s`,
      exec: 'stakeApplications',
    },
    // Relays sending scenario
    relays: {
      // Use ramping-arrival-rate executor to send relays and increase their rate
      // as per the load test plan.
      executor: 'ramping-arrival-rate',
      preAllocatedVUs: ENV_CONFIG.rps.initialCount,
      maxVUs: ENV_CONFIG.rps.maxCount,
      startRate: ENV_CONFIG.rps.initialCount,
      timeUnit: '1s',
      stages: buildRelayScenarioStages(),
      exec: 'sendRelay',
    }
  }
};