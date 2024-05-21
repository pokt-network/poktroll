// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information

package sync2_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pokt-network/poktroll/pkg/sync2"
)

func TestLimiterLimiting(t *testing.T) {
	t.Parallel()

	const N, Limit = 1000, 10
	ctx := context.Background()
	limiter := sync2.NewLimiter(Limit)
	counter := int32(0)
	for i := 0; i < N; i++ {
		limiter.Go(ctx, func() {
			if atomic.AddInt32(&counter, 1) > Limit {
				panic("limit exceeded")
			}
			time.Sleep(time.Millisecond)
			atomic.AddInt32(&counter, -1)
		})
	}
	limiter.Close()
}

func TestLimiterCanceling(t *testing.T) {
	t.Parallel()

	const N, Limit = 1000, 10
	limiter := sync2.NewLimiter(Limit)

	ctx, cancel := context.WithCancel(context.Background())

	counter := int32(0)

	waitForCancel := make(chan struct{}, N)
	block := make(chan struct{})
	allreturned := make(chan struct{})

	go func() {
		for i := 0; i < N; i++ {
			limiter.Go(ctx, func() {
				if atomic.AddInt32(&counter, 1) > Limit {
					panic("limit exceeded")
				}

				waitForCancel <- struct{}{}
				<-block
			})
		}
		close(allreturned)
	}()

	for i := 0; i < Limit; i++ {
		<-waitForCancel
	}
	cancel()
	<-allreturned
	close(block)

	limiter.Close()
	if counter > Limit {
		t.Fatal("too many times run")
	}

	started := limiter.Go(context.Background(), func() {
		panic("should not start")
	})
	if started {
		t.Fatal("should not start")
	}
}
