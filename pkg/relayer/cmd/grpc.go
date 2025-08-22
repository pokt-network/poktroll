// Package cmd provides gRPC connection utilities for the relayminer CLI.
package cmd

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCConfig holds configuration options for establishing a gRPC connection.
//
// Fields:
// - HostPort: gRPC host:port string
// - Insecure: Use insecure credentials
type GRPCConfig struct {
	HostPort string `yaml:"host_port"`
	Insecure bool   `yaml:"insecure"`
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
