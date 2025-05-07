package cmd

import (
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCConfig struct {
	HostPort          string        `yaml:"host_port"`
	Insecure          bool          `yaml:"insecure"`
	BackoffBaseDelay  time.Duration `yaml:"backoff_base_delay"`
	BackoffMaxDelay   time.Duration `yaml:"backoff_max_delay"`
	MinConnectTimeout time.Duration `yaml:"min_connect_timeout"`
	KeepAliveTime     time.Duration `yaml:"keep_alive_time"`
	KeepAliveTimeout  time.Duration `yaml:"keep_alive_timeout"`
}

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
