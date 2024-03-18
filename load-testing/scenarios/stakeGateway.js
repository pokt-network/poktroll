import { sleep } from 'k6';
import exec from 'k6/execution';
import file from 'k6/x/file';
import { command } from 'k6/x/exec';
import { stringify } from 'k6/x/yaml';

import { ENV_CONFIG } from '../config/index.js';
import { createAndFundAccount } from '../modules/accountSetup.js';
import {
  operationId,
  getBatchSize,
  stakingWaitDuration,
} from '../modules/scenarioStagesBuilder.js';

const gateways = {};

// stakeGateways is the main function for the stake gateway scenario.
// It handles staking multiple gateways if the batch size is greater than 1
// and adjust the wait time to match the staking duration given the time taken
// by the batch staking operation.
export function stakeGateways() {
  const actor = ENV_CONFIG.gateway;
  const startTime = new Date().getTime();
  const batchSize = getBatchSize(actor, exec.scenario.iterationInInstance);

  for (let i = 0; i < batchSize; i++) {
    execStakeGateway(operationId(actor, exec.scenario.iterationInInstance, i));
  }

  const operationDuration = Math.floor((new Date().getTime() - startTime) / 1000);
  const iterationDuration = stakingWaitDuration(actor);
  if (operationDuration < iterationDuration) {
    sleep(iterationDuration - operationDuration);
  }
}

// execStakeGateway stakes a gateway by creating a new account and funding it
// with the required amount of upokt, then it creates a staking config file and
// executes the staking transaction.
function execStakeGateway(id) {
  // The amount to fund the account with
  const fundAmount = 1000000000;
  // The key name for the account
  const keyName = `loadtest_gateway_${id}`;
  // Create a new account and fund it with the specified amount of upokt
  const address = createAndFundAccount(keyName, fundAmount)
  const configPath = `${ENV_CONFIG.PoktrolldHome}/config/${keyName}_stake_config.yaml`;
  const stakeConfig = {
    stake_amount: `${fundAmount * 0.9}upokt`,
  };

  gateways[id] = { id, address, stakeConfig, configPath, keyName };

  // Write the staking yaml config file
  file.writeString(configPath, stringify(stakeConfig));
  // Wait for the minimum amount of time since the k6/file plugin does not write
  // the file immediately.
  sleep(1);

  // Retry the staking transaction as the k6 exec command does not return the error
  // when it fails, so we need to check the output to see if it was successful.
  let output = "";
  while (true) {
    command('poktrolld', [
      'tx', 'gateway', 'stake-gateway',
      '--yes',
      '--config', configPath,
      '--from', address,
      '--node', ENV_CONFIG.Node,
      '--home', ENV_CONFIG.PoktrolldHome,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // Wait for the transaction to be included in a block
    sleep(ENV_CONFIG.SecondsPerBlock);

    output = command('poktrolld', [
      'query', 'gateway', 'show-gateway', address,
      '--node', ENV_CONFIG.Node,
      '--home', ENV_CONFIG.PoktrolldHome,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // If the output IS empty, this means that the staking transaction failed.
    // break out of the loop if the staking transaction was successful.
    if (output !== "") {
      break;
    }
  }
}