import { ENV_CONFIG, } from '../config/index.js';

// buildRelayScenarioStages returns the stages for the ramping-arrival-rate executor
// that is responsible of sending relays and increasing them as per the load test plan.
export function buildRelayScenarioStages() {
  // Numbmer of seconds for each stage.
  const stageDuration = ENV_CONFIG.SecondsPerBlock * ENV_CONFIG.rps.incrementBlockCount;

  // The first stage with the initial requests rate (RPS).
  const stages = [
    {
      duration: `${stageDuration}s`,
      target: ENV_CONFIG.rps.initialCount
    },
  ];

  // Create as many stages as needed to reach the maximum requests rate (RPS).
  for (
    let i = ENV_CONFIG.rps.countIncrement;
    i <= ENV_CONFIG.rps.maxCount;
    i = i + ENV_CONFIG.rps.countIncrement
  ) {
    // Add a stage with a duration of 0 seconds to immediately increase the
    // requests rate and avoid linearly increasing it.
    stages.push({ duration: 0, target: i });
    stages.push({
      duration: `${stageDuration}s`,
      target: i,
    });
  }

  return stages;
}

// stakingWaitDuration returns the number of seconds that corresponds to
// the number of blocks to wait before staking the next batch of suppliers.
export function stakingWaitDuration(actorConfig) {
  return actorConfig.incrementBlockCount * ENV_CONFIG.SecondsPerBlock
}

// stakingIterationsCount returns the number of iterations needed to reach
// the maximum number of staked actor it is ceiled and added +1 to account for
// staking the initial batch of actors and the last batch that may not increment
// by actorConfig.countIncrement if the division is not an integer.
export function stakingIterationsCount(actorConfig) {
  return Math.ceil((actorConfig.maxCount - actorConfig.initialCount) / actorConfig.countIncrement) + 1;
}

// actorStakeTotalDuration returns the total duration in seconds that corresponds
// to the staking process of the actor.
// k6 needs a duration of at least 1 second, so if the total duration is less than 1
// it returns 1.
export function actorStakeTotalDuration(actorConfig) {
  const totalDuration = (stakingWaitDuration(actorConfig) + 1) * stakingIterationsCount(actorConfig);
  return totalDuration < 1 ? 1 : totalDuration;
}

// getBatchSize returns the number of actors to be staked in the current iteration.
// If it is the first iteration, it returns the initialCount, if it is the last iteration
// return the remaining actors to stake that may not correspond to actorConfig.countIncrement
// it returns actorConfig.countIncrement otherwise.
export function getBatchSize(actorConfig, scenarioIteration) {
  // If it is the first iteration, return the initialCount.
  if (scenarioIteration === 0) {
    return actorConfig.initialCount;
  }

  // If it is the last iteration, return the remaining actors to stake if it does not
  // correspond to actorConfig.countIncrement.
  if (
    stakingIterationsCount(actorConfig) > ((actorConfig.maxCount - actorConfig.initialCount) / actorConfig.countIncrement) + 1 &&
    scenarioIteration + 1 === stakingIterationsCount(actorConfig)
  ) {
    return actorConfig.countIncrement - actorConfig.initialCount;
  }

  return actorConfig.countIncrement;
}

// operationId returns the id of the current operation to be executed.
// It takes into account iterations that stake multiple actors at a time.
// This id is used to identify the actor so it has a unique keyName in the keyring
// and a unique staking config file name.
export function operationId(actor, scenarioIteration, i) {
  if (scenarioIteration === 0) {
    return i;
  }

  return i + (scenarioIteration * actor.countIncrement) - actor.initialCount + 1;
}