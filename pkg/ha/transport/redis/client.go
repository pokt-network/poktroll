package redis

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ClientConfig contains configuration for creating a Redis client.
type ClientConfig struct {
	// URL is the Redis connection URL.
	// Supports: redis://, rediss:// (TLS), redis-sentinel://, redis-cluster://
	URL string

	// MaxRetries is the maximum number of retries before giving up.
	// Default: 3
	MaxRetries int

	// PoolSize is the maximum number of socket connections.
	// Default: 10 connections per CPU
	PoolSize int

	// MinIdleConns is the minimum number of idle connections.
	// Default: 0
	MinIdleConns int
}

// NewClient creates a new Redis client from the configuration.
// Supports standalone, sentinel, and cluster modes based on URL scheme.
func NewClient(ctx context.Context, config ClientConfig) (redis.UniversalClient, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("redis URL is required")
	}

	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	// Set defaults
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	poolSize := config.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}

	var client redis.UniversalClient

	switch u.Scheme {
	case "redis", "rediss":
		// Standalone Redis
		opts, parseErr := redis.ParseURL(config.URL)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", parseErr)
		}
		opts.MaxRetries = maxRetries
		opts.PoolSize = poolSize
		opts.MinIdleConns = config.MinIdleConns
		client = redis.NewClient(opts)

	case "redis-sentinel":
		// Redis Sentinel
		client, err = newSentinelClient(u, maxRetries, poolSize, config.MinIdleConns)
		if err != nil {
			return nil, err
		}

	case "redis-cluster":
		// Redis Cluster
		client, err = newClusterClient(u, maxRetries, poolSize, config.MinIdleConns)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported redis URL scheme: %s", u.Scheme)
	}

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// newSentinelClient creates a Redis Sentinel client.
// URL format: redis-sentinel://[:password@]host1:port1,host2:port2/master_name[?db=N]
func newSentinelClient(u *url.URL, maxRetries, poolSize, minIdleConns int) (redis.UniversalClient, error) {
	// Parse master name from path
	masterName := strings.TrimPrefix(u.Path, "/")
	if masterName == "" {
		return nil, fmt.Errorf("sentinel URL must include master name in path")
	}

	// Parse sentinel addresses
	addrs := strings.Split(u.Host, ",")
	if len(addrs) == 0 {
		return nil, fmt.Errorf("sentinel URL must include at least one sentinel address")
	}

	// Parse password
	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Parse DB number
	db := 0
	if dbStr := u.Query().Get("db"); dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid db number: %w", err)
		}
	}

	opts := &redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: addrs,
		Password:      password,
		DB:            db,
		MaxRetries:    maxRetries,
		PoolSize:      poolSize,
		MinIdleConns:  minIdleConns,
	}

	return redis.NewFailoverClient(opts), nil
}

// newClusterClient creates a Redis Cluster client.
// URL format: redis-cluster://[:password@]host1:port1,host2:port2[?db=N]
func newClusterClient(u *url.URL, maxRetries, poolSize, minIdleConns int) (redis.UniversalClient, error) {
	// Parse cluster addresses
	addrs := strings.Split(u.Host, ",")
	if len(addrs) == 0 {
		return nil, fmt.Errorf("cluster URL must include at least one node address")
	}

	// Parse password
	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}

	opts := &redis.ClusterOptions{
		Addrs:        addrs,
		Password:     password,
		MaxRetries:   maxRetries,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	}

	return redis.NewClusterClient(opts), nil
}
