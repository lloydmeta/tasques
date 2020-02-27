package recurring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lloydmeta/tasques/internal/domain/leader"
)

type void struct{}

var member void

type state uint32

func (s state) String() string {
	return statesToString[s]
}

const (
	NOT_LEADER state = iota
	LEADER
)

var statesToString = map[state]string{
	NOT_LEADER: "NOT_LEADER",
	LEADER:     "LEADER",
}

// TODO tests
// Manager is in charge of reading recurring Tasks and scheduling them
// to be run, keeping things synced and updated on calls to its methods.
type Manager struct {
	scheduler Scheduler
	service   Service

	scheduledTasks map[Id]Task

	getUTC func() time.Time

	leaderState state
	mu          sync.Mutex
}

// Returns a new recurring Tasks Manager
func NewManager(scheduler Scheduler, service Service) Manager {
	return Manager{
		scheduler:      scheduler,
		service:        service,
		scheduledTasks: make(map[Id]Task),
		getUTC: func() time.Time {
			return time.Now().UTC()
		},
		leaderState: NOT_LEADER,
		mu:          sync.Mutex{},
	}
}

// Returns a function that conditionally syncs Recurring Task changes from the data store,
// ensuring tasks that are deleted are stopped, tasks that are created or updated are scheduled
// properly
func (m *Manager) RecurringSyncFunc() func(ctx context.Context, isLeader leader.Checker) error {
	return m.recurringFunc("sync of unseen changes", m.syncNotLoaded)
}

// Returns a function that conditionally runs a full sync of Recurring Task changes from the data store
func (m *Manager) RecurringSyncEnforceFunc() func(ctx context.Context, isLeader leader.Checker) error {
	return m.recurringFunc("full sync check", m.enforceSync)
}

// Returns a sync-related function
func (m *Manager) recurringFunc(description string, ifStillLeader func(ctx context.Context) error) func(ctx context.Context, leaderChecker leader.Checker) error {
	return func(ctx context.Context, leaderChecker leader.Checker) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		previousLeaderState := m.leaderState
		currentLeaderState := leaderState(leaderChecker)
		m.leaderState = currentLeaderState
		switch {
		case previousLeaderState == NOT_LEADER && currentLeaderState == LEADER:
			log.Info().
				Str("previous", previousLeaderState.String()).
				Str("current", currentLeaderState.String()).
				Msg("Newly acquired leader lock, initiating complete refresh of Recurring Tasks")
			return m.completeReload(ctx)
		case previousLeaderState == LEADER && currentLeaderState == LEADER:
			log.Debug().
				Str("previous", previousLeaderState.String()).
				Str("current", currentLeaderState.String()).
				Msgf("Still has leader lock, initiating %s", description)
			return ifStillLeader(ctx)
		case previousLeaderState == LEADER && currentLeaderState == NOT_LEADER:
			log.Info().
				Str("previous", previousLeaderState.String()).
				Str("current", currentLeaderState.String()).
				Msg("Lost leader lock, initiating stop and unloading of all Recurring Tasks.")
			m.stopAll()
			return nil
		case previousLeaderState == NOT_LEADER && currentLeaderState == NOT_LEADER:
			log.Debug().
				Str("previous", previousLeaderState.String()).
				Str("current", currentLeaderState.String()).
				Msg("Still does not have leader lock, ignoring")
			return nil
		default:
			return fmt.Errorf("Impossible leader states. Previous [%v] Current [%v]", previousLeaderState.String(), currentLeaderState.String())
		}
	}
}

func leaderState(leaderChecker leader.Checker) state {
	if leaderChecker.IsLeader() {
		return LEADER
	} else {
		return NOT_LEADER
	}
}

// ALL OF THE FOLLOWING METHODS ARE NOT SYNCED
// because they should only be called from the functions
// returned above, which *are* synced

// Stops all recurring Tasks and removes them from the in-memory
// map
// stops all scheduled tasks and removes them from the in-memory lookup
func (m *Manager) stopAll() {
	for _, t := range m.scheduledTasks {
		m.unscheduleAndRemoveFromScheduledTasksState(t)
	}
}

// Stops all recurring Tasks and removes them from the in-memory
// map, then reads them from the backing store and schedules them
// and loads them in memory
func (m *Manager) completeReload(ctx context.Context) error {
	// Stop everything first
	m.stopAll()

	all, err := m.service.All(ctx)
	if err != nil {
		return err
	}

	var loaded []Task
	for _, t := range all {
		m.scheduleAndAddToList(t, &loaded)
	}
	return m.markAsLoadedAndUpdateScheduledTasksState(ctx, loaded)
}

// Loads un-seen changes from the data store:
// * Newly-deleted recurring Tasks are unscheduled and removed from the scheduled Tasks
//   internal state
// * Newly-updated + created Tasks are unscheduled, scheduled, updating state
func (m *Manager) syncNotLoaded(ctx context.Context) error {
	notLoadeds, err := m.service.NotLoaded(ctx)
	if err != nil {
		log.Error().Err(err).Msg(
			"Failed to grab unloaded recurring task changes; returning the error without updating internal state because subsequent calls should hopefully succeed")
		return err
	}

	var deletedTasks []Task
	var newOrUpdatedTasks []Task
	for _, notLoaded := range notLoadeds {
		if notLoaded.IsDeleted {
			deletedTasks = append(deletedTasks, notLoaded)
		} else {
			newOrUpdatedTasks = append(newOrUpdatedTasks, notLoaded)
		}
	}

	// handle deleteds by unscheduling
	for _, deleted := range deletedTasks {
		m.unscheduleAndRemoveFromScheduledTasksState(deleted)
	}

	var scheduled []Task
	// handle new and updateds by unscheduling if present, then scheduling
	for _, updated := range newOrUpdatedTasks {
		m.unscheduleAndRemoveFromScheduledTasksState(updated)
		m.scheduleAndAddToList(updated, &scheduled)
	}

	// "acked" just means we've taken it into account, hence why we add deleteds in there as well.
	acked := append(deletedTasks, scheduled...)
	return m.markAsLoadedAndUpdateScheduledTasksState(ctx, acked)
}

func (m *Manager) enforceSync(ctx context.Context) error {
	log.Info().Msg("Checking if scheduled Recurring Tasks are in sync with data store.")

	all, err := m.service.All(ctx)
	if err != nil {
		return err
	}

	var notInDataStore []Task
	var notInMemory []Task
	var versionMismatch []Task

	// check if what we have in the data store is different from in memory
	idsFromDataStore := make(map[Id]void, len(all))
	for _, fromDataStore := range all {
		idsFromDataStore[fromDataStore.ID] = member
		if inMemory, present := m.scheduledTasks[fromDataStore.ID]; !present {
			notInMemory = append(notInMemory, fromDataStore)
		} else {
			if inMemory.Metadata.Version != fromDataStore.Metadata.Version {
				versionMismatch = append(versionMismatch, fromDataStore)
			}
		}
	}

	// check if we have things in memory that are not in the data store
	for fromInMemoryId, fromInMemoryTask := range m.scheduledTasks {
		if _, present := idsFromDataStore[fromInMemoryId]; !present {
			notInDataStore = append(notInDataStore, fromInMemoryTask)
		}
	}

	needsResync := false
	if len(notInDataStore) != 0 {
		log.Warn().
			Int("count", len(notInDataStore)).
			Msg("Detected Tasks that are in memory, but not persisted")
		needsResync = true
	}
	if len(notInMemory) != 0 {
		log.Warn().
			Int("count", len(notInMemory)).
			Msg("Detected Tasks that are persisted, but not loaded in memory")
		needsResync = true
	}
	if len(versionMismatch) != 0 {
		log.Warn().
			Int("count", len(versionMismatch)).
			Msg("Detected Tasks that are in memory, but have version mismatches vs what is persisted")
		needsResync = true
	}

	if needsResync {
		log.Warn().Bool("needsResync", needsResync).Msg("Initiating complete reload")
		return m.completeReload(ctx)
	} else {
		log.Info().Msg("Scheduled Recurring Tasks are in sync with data store.")
		return nil
	}

}

// Attempts to schedules a Task to be inserted at its specified interval.
// If successful (majority case), we add it to the list provided as the second
// argument, otherwise, we log a failure and move on without adding it to
// the list.
//
// Note that this *does not* update the internal scheduled Tasks map because we only
// want to update it _after_ we mark it as loaded in ES and have updated metadata version
func (m *Manager) scheduleAndAddToList(t Task, loaded *[]Task) {
	err := m.scheduler.Schedule(t)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", string(t.ID)).
			Str("schedule", string(t.ScheduleExpression)).
			Msg("Failed to schedule")
	} else {
		*loaded = append(*loaded, t)
	}
}

// Schedules a Task to be inserted at its specified interval
// _and_ removes it from the internal scheduled Tasks map
func (m *Manager) unscheduleAndRemoveFromScheduledTasksState(t Task) {
	success := m.scheduler.Unschedule(t.ID)
	if !success {
		if log.Debug().Enabled() {
			log.Debug().
				Str("id", string(t.ID)).
				Str("schedule", string(t.ScheduleExpression)).
				Msg("Failed to unschedule Task; likely means it was never scheduled locally in the first place.")
		}
	}
	delete(m.scheduledTasks, t.ID)
}

func (m *Manager) markAsLoadedAndUpdateScheduledTasksState(ctx context.Context, loaded []Task) error {
	if len(loaded) > 0 {
		result, err := m.service.MarkLoaded(ctx, loaded)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to mark scheduledTasks recurring tasks as scheduledTasks; not updating internal state and simply returning the error because subsequent refreshes will eventually fix this")
			return err
		}

		// unschedule failures; we'll grab them next time
		if len(result.NotFounds) != 0 {
			log.Error().
				Interface("tasks", result.NotFounds).
				Msg("Unscheduling recurring tasks that could not be found when marking as loaded")
			for _, t := range result.NotFounds {
				m.unscheduleAndRemoveFromScheduledTasksState(t)
			}
		}
		if len(result.VersionConflicts) != 0 {
			log.Error().
				Interface("tasks", result.VersionConflicts).
				Msg("Tasks that resulted in version conflicts when marking as loaded, not updating internal state instead of unscheduling because subsequent refreshes will eventually fix this")
		}
		if len(result.Others) != 0 {
			log.Error().
				Interface("tasks", result.Others).
				Msg("Tasks that resulted in other errors when marking as loaded, not updating internal state instead of unscheduling because subsequent refreshes will eventually fix this")
		}

		// Stuff non-deleted Tasks that failed to update with "not found" into the internal
		// scheduled Tasks map.
		//
		// Successful updates will get updated metadata versions, while the failures won't, but
		// it doesn't matter much, we'll get them in subsequent calls
		for _, t := range result.Successes {
			if !t.IsDeleted {
				m.scheduledTasks[t.ID] = t
			}
		}
		for _, t := range result.VersionConflicts {
			if !t.IsDeleted {
				m.scheduledTasks[t.ID] = t
			}
		}
		for _, r := range result.Others {
			t := r.RecurringTask
			if !t.IsDeleted {
				m.scheduledTasks[t.ID] = t
			}
		}
	}
	return nil
}
