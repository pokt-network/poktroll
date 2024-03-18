import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

// Environment configuration
export const ENV_CONFIG = {
    anvilBaseUrl: __ENV.ANVIL_BASE_URL || 'http://localhost:8547',
    nginxBaseUrl: __ENV.NGINX_BASE_URL || 'http://localhost:8548',
    AppGateServerAnvilUrl: __ENV.APP_GATE_SERVER_ANVIL_URL || 'http://localhost:42069/anvil',

    PoktrolldHome: __ENV.POKTROLLD_HOME || './localnet/poktrolld',
    Node: __ENV.NODE || 'tcp://127.0.0.1:36657',
    ChainID: __ENV.CHAIN_ID || 'poktroll',

    // Time in seconds for a block to be produced.
    // Make sure this matches the block time of the tested network.
    SecondsPerBlock: __ENV.SECONDS_PER_BLOCK || 1, // 10

    // Supplier load test configuration
    supplier: {
      // Maximum number of suppliers to be staked during the test.
      maxCount: __ENV.MAX_SUPPLIERS || 10, // 100
      // Initial number of suppliers to be staked as the test starts.
      initialCount: __ENV.INITIAL_SUPPLIERS_COUNT || 2, // 5
      // Number of suppliers to be staked in each iteration (incrementBlockCount).
      countIncrement: __ENV.SUPPLIER_COUNT_INCREMENT || 1, // 1
      // Number of blocks to wait before staking the next batch of suppliers (countIncrement).
      incrementBlockCount: __ENV.SUPPLIER_INCREMENT_BLOCK_COUNT || 10, // 100
    },
    gateway: {
      // Maximum number of gateways to be staked during the test.
      maxCount: __ENV.MAX_GATEWAYS || 10, // 10
      // Initial number of gateways to be staked as the test starts.
      initialCount: __ENV.INITIAL_GATEWAYS_COUNT || 1, // 1
      // Number of gateways to be staked in each iteration (incrementBlockCount).
      countIncrement: __ENV.GATEWAY_COUNT_INCREMENT || 1, // 1
      // Number of blocks to wait before staking the next batch of gateways (countIncrement).
      incrementBlockCount: __ENV.GATEWAY_INCREMENT_BLOCK_COUNT || 10, // 10
    },
    application: {
      // Maximum number of applications to be staked during the test.
      maxCount: __ENV.MAX_APPLICATIONS || 100, // 1000
      // Initial number of applications to be staked as the test starts.
      initialCount: __ENV.INITIAL_APPLICATIONS_COUNT || 5, // 5
      // Number of applications to be staked in each iteration (incrementBlockCount).
      countIncrement: __ENV.APPLICATION_COUNT_INCREMENT || 10, // 10
      // Number of blocks to wait before staking the next batch of applications (countIncrement).
      incrementBlockCount: __ENV.APPLICATION_INCREMENT_BLOCK_COUNT || 10, // 10
    },
    rps: {
      // Maximum requests rate (RPS) to be sent during the test.
      maxCount: __ENV.MAX_RPS || 1000, // 10000
      // Initial requests rate (RPS) to be sent as the test starts.
      initialCount: __ENV.INITIAL_RPS || 1, // 1
      // Number of requests rate (RPS) to be incremented in each iteration (incrementBlockCount).
      countIncrement: __ENV.RPS_COUNT_INCREMENT || 10, // 100
      // Number of blocks to wait before incrementing the requests rate (RPS) (countIncrement).
      incrementBlockCount: __ENV.RPS_INCREMENT_BLOCK_COUNT || 10, // 10
    }
};

// We can export this function in our tests to generate HTML summary.
export function handleSummary(data) {
  return {
    "summary.html": htmlReport(data),
    stdout: textSummary(data, { indent: " ", enableColors: true }),
  };
}