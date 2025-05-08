// Package cmd provides gRPC connection utilities for the relayminer CLI.
package cmd

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCConfig holds configuration options for establishing a gRPC connection.
//
// Fields:
// - HostPort: gRPC host:port string
// - Insecure: Use insecure credentials
// - BackoffBaseDelay: Base delay for backoff
// - BackoffMaxDelay: Max delay for backoff
// - MinConnectTimeout: Minimum connection timeout
// - KeepAliveTime: Keepalive interval
// - KeepAliveTimeout: Keepalive timeout
//
type GRPCConfig struct {
	HostPort          string        `yaml:"host_port"`
	Insecure          bool          `yaml:"insecure"`
	BackoffBaseDelay  time.Duration `yaml:"backoff_base_delay"`
	BackoffMaxDelay   time.Duration `yaml:"backoff_max_delay"`
	MinConnectTimeout time.Duration `yaml:"min_connect_timeout"`
	KeepAliveTime     time.Duration `yaml:"keep_alive_time"`
	KeepAliveTimeout  time.Duration `yaml:"keep_alive_timeout"`
}

// connectGRPC establishes a gRPC client connection using the provided GRPCConfig.
//
// - Returns a grpc.ClientConn or error
// - Uses insecure credentials if config.Insecure is true
func connectGRPC(config GRPCConfig) (*grpc.ClientConn, error) {
	if config.Insecure {
		transport := grpc.WithTransportCredentials(insecure.NewCredentials())
		dialOptions := []grpc.DialOption{transport}
		return grpc.NewClient(
			config.HostPort,
			dialOptions...,
		)
	}

	return grpc.NewClient(
		config.HostPort,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	)
}
