package config

import "time"

func (relayMinerConfig *RelayMinerConfig) HydrateMiningSupervisor(
	yamlMiningSupervisorConfig *YAMLMiningSupervisorConfig,
) error {
	relayMinerConfig.MiningSupervisorConfig = &MiningSupervisorConfig{}

	if yamlMiningSupervisorConfig.QueueSize == 0 {
		relayMinerConfig.MiningSupervisorConfig.QueueSize = DefaultMSQueueSize
	} else {
		relayMinerConfig.MiningSupervisorConfig.QueueSize = yamlMiningSupervisorConfig.QueueSize
	}

	if yamlMiningSupervisorConfig.Workers == 0 {
		relayMinerConfig.MiningSupervisorConfig.Workers = DefaultMSWorkers
	} else {
		relayMinerConfig.MiningSupervisorConfig.Workers = yamlMiningSupervisorConfig.Workers
	}

	if yamlMiningSupervisorConfig.EnqueueTimeoutMs == 0 {
		relayMinerConfig.MiningSupervisorConfig.EnqueueTimeout = time.Duration(DefaultMSEnqueueTimeout) * time.Millisecond
	} else {
		relayMinerConfig.MiningSupervisorConfig.EnqueueTimeout = time.Duration(yamlMiningSupervisorConfig.EnqueueTimeoutMs) * time.Millisecond
	}

	if yamlMiningSupervisorConfig.GaugeSampleIntervalMs == 0 {
		relayMinerConfig.MiningSupervisorConfig.GaugeSampleInterval = time.Duration(DefaultMSGaugeSampleInterval) * time.Millisecond
	} else {
		relayMinerConfig.MiningSupervisorConfig.GaugeSampleInterval = time.Duration(yamlMiningSupervisorConfig.GaugeSampleIntervalMs) * time.Millisecond
	}

	if yamlMiningSupervisorConfig.DropLogIntervalMs == 0 {
		relayMinerConfig.MiningSupervisorConfig.DropLogInterval = time.Duration(DefaultMSDropLogInterval) * time.Millisecond
	} else {
		relayMinerConfig.MiningSupervisorConfig.DropLogInterval = time.Duration(yamlMiningSupervisorConfig.DropLogIntervalMs) * time.Millisecond
	}

	if yamlMiningSupervisorConfig.DropPolicy == "" || (yamlMiningSupervisorConfig.DropPolicy != "drop-new" && yamlMiningSupervisorConfig.DropPolicy != "drop-oldest") {
		relayMinerConfig.MiningSupervisorConfig.DropPolicy = DefaultMSDropPolicy
	} else {
		relayMinerConfig.MiningSupervisorConfig.DropPolicy = yamlMiningSupervisorConfig.DropPolicy
	}

	return nil
}
