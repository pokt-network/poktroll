import { command } from 'k6/x/exec';
import { sleep } from 'k6';

import { ENV_CONFIG } from '../config/index.js';

// Create a new account and fund it with the specified amount of upokt
export function createAndFundAccount(keyName, amount) {
  // Create a new account
  command("ignite", [
    "account", "create", keyName,
    '--keyring-dir', ENV_CONFIG.PoktrolldHome,
  ]);

  // Get the address of the new account
  let output = command('ignite', [
    'account', 'show', keyName,
    '--address-prefix', 'pokt',
    '--keyring-dir', ENV_CONFIG.PoktrolldHome,
  ]);
  const address = output.substr(output.indexOf('pokt'), 43);

  while(true) {
    // Fund the account with the specified amount of upokt
    command('poktrolld', [
      'tx', 'bank', 'send', 'pnf', address, `${amount}upokt`,
      '--yes',
      '--home', ENV_CONFIG.PoktrolldHome,
      '--node', ENV_CONFIG.Node,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // Wait for the transaction to be included in a block
    sleep(ENV_CONFIG.SecondsPerBlock);

    // Check the account balance
    output = command('poktrolld', [
      'query', 'bank', 'balance', address, 'upokt',
      '--home', ENV_CONFIG.PoktrolldHome,
      '--node', ENV_CONFIG.Node,
      '--chain-id', ENV_CONFIG.ChainID,
    ]);

    // If the account balance is not zero, break out of the loop.
    // k6/exec command does not return the error when it fails, so we need to
    // check that the output is not empty before continuing.
    if (output.indexOf('amount: "0"') < 0) {
      break;
    }
  }

  return address;
}