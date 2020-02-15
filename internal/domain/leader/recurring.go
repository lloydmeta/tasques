package leader

import (
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// RecurringTask defines a recurring task to run if we have the leader lock
type RecurringTask struct {
	name string
	// How often to run it
	interval time.Duration
	// What to run
	f func(isLeader Checker) error
}

// NewRecurringTask returns a new recurring task (but doesn't run it)
// Also assumes f is non-nil.
func NewRecurringTask(name string, interval time.Duration, f func(isLeader Checker) error) RecurringTask {
	return RecurringTask{name: name, interval: interval, f: f}
}

// RecurringTaskRunner is a runner of RecurringTasks
type RecurringTaskRunner struct {
	tasks      []RecurringTask
	stopped    uint32
	leaderLock Lock
}

// NewRecurringTaskRunner creates a new RecurringTaskRunner
func NewRecurringTaskRunner(tasks []RecurringTask, leaderLock Lock) RecurringTaskRunner {
	return RecurringTaskRunner{
		tasks:      tasks,
		stopped:    1,
		leaderLock: leaderLock,
	}
}

// Start begins the RecurringTaskRunner loop
func (r *RecurringTaskRunner) Start() {
	atomic.StoreUint32(&r.stopped, 0)
	for _, t := range r.tasks {
		go func(task RecurringTask, isStopped func() bool, hasLock func() bool) {
			for isStopped() {
				startIterationTime := time.Now().UTC()
				err := task.f(r.leaderLock)
				if err != nil {
					log.Error().Err(err).Msgf("Failed when running task [%s]", task.name)
				}
				waitTime := task.interval - time.Since(startIterationTime)
				if waitTime > 0 {
					time.Sleep(waitTime)
				}
			}
			log.Info().Msgf("Recurring task ended [%s]", task.name)
		}(t, r.shouldRun, r.leaderLock.IsLeader)
	}
}

// Stop stops the RecurringTaskRunner loop
func (r *RecurringTaskRunner) Stop() {
	log.Info().Msg("Stopping recurring tasks")
	atomic.StoreUint32(&r.stopped, 1)
}

func (r *RecurringTaskRunner) shouldRun() bool {
	return atomic.LoadUint32(&r.stopped) == 0
}
