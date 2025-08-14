package session

import (
	"github.com/alitto/pond/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	relayMinerProcess = "relayminer"
)

func RegisterPoolMetrics(pool pond.Pool) {
	// Worker pool metrics
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_workers_running",
			Help:      "Current number of running workers",
		},
		func() float64 {
			return float64(pool.RunningWorkers())
		}))

	// Task metrics
	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_submitted_total",
			Help:      "Total number of tasks submitted since the pool was created and before it was stopped. This includes tasks that were dropped because the queue was full",
		},
		func() float64 {
			return float64(pool.SubmittedTasks())
		}))

	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_waiting_total",
			Help:      "Current number of tasks in the queue that are waiting to be executed",
		},
		func() float64 {
			return float64(pool.WaitingTasks())
		}))

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_successful_total",
			Help:      "Total number of tasks that have successfully completed their execution since the pool was created",
		},
		func() float64 {
			return float64(pool.SuccessfulTasks())
		}))

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_failed_total",
			Help:      "Total number of tasks that completed with panic since the pool was created",
		},
		func() float64 {
			return float64(pool.FailedTasks())
		}))

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_completed_total",
			Help:      "Total number of tasks that have completed their execution either successfully or with panic since the pool was created",
		},
		func() float64 {
			return float64(pool.CompletedTasks())
		}))

	prometheus.MustRegister(prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Subsystem: relayMinerProcess,
			Name:      "pool_tasks_dropped_total",
			Help:      "Total number of tasks that were dropped because the queue was full since the pool was created",
		},
		func() float64 {
			return float64(pool.DroppedTasks())
		}))
}
