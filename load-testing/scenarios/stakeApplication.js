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

const applications = {};

// stakeApplication is the main function for the stake application scenario.
// It handles staking multiple applications if the batch size is greater than 1
// and adjust the wait time to match the staking duration given the time taken
// by the batch staking operation.
export function stakeApplications() {
  const actor = ENV_CONFIG.application;
  const startTime = new Date().getTime();
  const batchSize = getBatchSize(actor, exec.scenario.iterationInInstance);

  for (let i = 0; i < batchSize; i++) {
    execStakeApplication(operationId(actor, exec.scenario.iterationInInstance, i));
  }

  const operationDuration = Math.floor((new Date().getTime() - startTime) / 1000);
  const iterationDuration = stakingWaitDuration(actor);
  if (operationDuration < iterationDuration) {
    sleep(iterationDuration - operationDuration);
  }
}

// execStakeApplication stakes an application by creating a new account and funding it
// with the required amount of upokt, then it creates a staking config file and
// executes the staking transaction.
function execStakeApplication(id) {
  // The amount to fund the account with.
  const fundAmount = 100000000000000;
  // The key name for the account.
  const keyName = `loadtest_application_${id}`;
  // Create a new account and fund it with the specified amount of upokt
  const address = createAndFundAccount(keyName, fundAmount)
  const configPath = `${ENV_CONFIG.PoktrolldHome}/config/${keyName}_stake_config.yaml`;
  const stakeConfig = {
    stake_amount: `${fundAmount * 0.9}upokt`,
    service_ids: ['anvil'],
  };

  applications[id] = { id, address, stakeConfig, configPath, keyName };

  // Write the staking yaml config file.
  file.writeString(configPath, stringify(stakeConfig));
  // Wait for the minimum amount of time since the k6/file plugin does not write
  // the file immediately.
  sleep(1);

  // Retry the staking transaction as the k6 exec command does not return the error
  // when it fails, so we need to check that the output is not empty before continuing.
  let output = "";
  while (true) {
    command('poktrolld', [
      'tx', 'application', 'stake-application',
      '--yes',
      '--config', configPath,
      '--from', address,
      '--node', ENV_CONFIG.Node,
      '--home', ENV_CONFIG.PoktrolldHome,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // Wait for the transaction to be included in a block.
    sleep(ENV_CONFIG.SecondsPerBlock);

    output = command('poktrolld', [
      'query', 'application', 'show-application', address,
      '--node', ENV_CONFIG.Node,
      '--home', ENV_CONFIG.PoktrolldHome,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // If the output IS empty, this means that the staking transaction failed
    // or failed to be included in a block, so we retry the transaction.
    // break out of the loop if the output is not empty.
    if (output !== "") {
      break;
    }
  }
}