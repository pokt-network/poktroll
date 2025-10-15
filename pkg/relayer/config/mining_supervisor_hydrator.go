package config

import "time"

const (
	MSDropPolicyNew    = "drop-new"
	MSDropPolicyOldest = "drop-oldest"
)

func (relayMinerConfig *RelayMinerConfig) HydrateMiningSupervisor(
	yamlMiningSupervisorConfig *YAMLMiningSupervisorConfig,
) error {
	config := &MiningSupervisorConfig{}

	// Relay Queue Size
	if yamlMiningSupervisorConfig.QueueSize == 0 {
		config.QueueSize = DefaultMSQueueSize
	} else {
		config.QueueSize = yamlMiningSupervisorConfig.QueueSize
	}

	// Relay Workers
	if yamlMiningSupervisorConfig.Workers == 0 {
		config.Workers = DefaultMSWorkers
	} else {
		config.Workers = yamlMiningSupervisorConfig.Workers
	}

	// Enqueue Timeout
	if yamlMiningSupervisorConfig.EnqueueTimeoutMs == 0 {
		config.EnqueueTimeout = time.Duration(DefaultMSEnqueueTimeout) * time.Millisecond
	} else {
		config.EnqueueTimeout = time.Duration(yamlMiningSupervisorConfig.EnqueueTimeoutMs) * time.Millisecond
	}

	// Gauge Sample Interval
	if yamlMiningSupervisorConfig.GaugeSampleIntervalMs == 0 {
		config.GaugeSampleInterval = time.Duration(DefaultMSGaugeSampleInterval) * time.Millisecond
	} else {
		config.GaugeSampleInterval = time.Duration(yamlMiningSupervisorConfig.GaugeSampleIntervalMs) * time.Millisecond
	}

	// Drop Log Interval
	if yamlMiningSupervisorConfig.DropLogIntervalMs == 0 {
		config.DropLogInterval = time.Duration(DefaultMSDropLogInterval) * time.Millisecond
	} else {
		config.DropLogInterval = time.Duration(yamlMiningSupervisorConfig.DropLogIntervalMs) * time.Millisecond
	}

	// Drop Policy
	isDropPolicyConfigured := (yamlMiningSupervisorConfig.DropPolicy == "") ||
		(yamlMiningSupervisorConfig.DropPolicy != MSDropPolicyNew && yamlMiningSupervisorConfig.DropPolicy != MSDropPolicyOldest)
	if isDropPolicyConfigured {
		config.DropPolicy = DefaultMSDropPolicy
	} else {
		config.DropPolicy = yamlMiningSupervisorConfig.DropPolicy
	}

	relayMinerConfig.MiningSupervisorConfig = config
	return nil
}
