package leader

import (
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// InternalRecurringFunction defines a recurring task to run if we have the leader lock
type InternalRecurringFunction struct {
	name string
	// How often to run it
	interval time.Duration
	// What to run
	f func(isLeader Checker) error
}

// NewInternalRecurringFunction returns a new recurring task (but doesn't run it)
// Also assumes f is non-nil.
func NewInternalRecurringFunction(name string, interval time.Duration, f func(isLeader Checker) error) InternalRecurringFunction {
	return InternalRecurringFunction{name: name, interval: interval, f: f}
}

// InternalRecurringFunctionRunner is a runner of InternalRecurringFunction
type InternalRecurringFunctionRunner struct {
	functions  []InternalRecurringFunction
	stopped    uint32
	leaderLock Lock
}

// NewInternalRecurringFunctionRunner creates a new InternalRecurringFunctionRunner
func NewInternalRecurringFunctionRunner(tasks []InternalRecurringFunction, leaderLock Lock) InternalRecurringFunctionRunner {
	return InternalRecurringFunctionRunner{
		functions:  tasks,
		stopped:    1,
		leaderLock: leaderLock,
	}
}

// Start begins the InternalRecurringFunctionRunner loop
func (r *InternalRecurringFunctionRunner) Start() {
	atomic.StoreUint32(&r.stopped, 0)
	for _, t := range r.functions {
		go func(task InternalRecurringFunction, shouldRun func() bool, isLeader Checker) {
			for shouldRun() {
				startIterationTime := time.Now().UTC()
				err := task.f(isLeader)
				if err != nil {
					log.Error().Err(err).Msgf("Failed when running task [%s]", task.name)
				}
				waitTime := task.interval - time.Since(startIterationTime)
				if waitTime > 0 {
					time.Sleep(waitTime)
				}
			}
			log.Info().Msgf("Recurring task ended [%s]", task.name)
		}(t, r.shouldRun, r.leaderLock)
	}
}

// Stop stops the InternalRecurringFunctionRunner loop
func (r *InternalRecurringFunctionRunner) Stop() {
	log.Info().Msg("Stopping recurring functions")
	atomic.StoreUint32(&r.stopped, 1)
}

func (r *InternalRecurringFunctionRunner) shouldRun() bool {
	return atomic.LoadUint32(&r.stopped) == 0
}
