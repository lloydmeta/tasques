package recurring

import (
	"context"
	"fmt"
	"sync"

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

// Manager is in charge of reading recurring Tasks and scheduling them
// to be run, keeping things synced and updated on calls to its methods.
//
// It exposes methods that are called by the InternalRecurringFunctionRunner
// at configured intervals.
type Manager interface {
	// Returns a function that conditionally syncs Recurring Task changes from the data store,
	// ensuring tasks that are deleted are stopped, tasks that are created or updated are scheduled
	// properly
	RecurringSyncFunc() func(ctx context.Context, isLeader leader.Checker) error

	// Returns a function that conditionally runs a full sync of Recurring Task changes from the data store
	RecurringSyncEnforceFunc() func(ctx context.Context, isLeader leader.Checker) error
}

// TODO tests
type impl struct {
	scheduler Scheduler
	service   Service

	// Internal state
	scheduledTasks map[Id]Task
	leaderState    state

	// Used to ensure sanity
	mu sync.Mutex
}

// Returns a new recurring Tasks Manager
func NewManager(scheduler Scheduler, service Service) Manager {
	return &impl{
		scheduler:      scheduler,
		service:        service,
		scheduledTasks: make(map[Id]Task),
		leaderState:    NOT_LEADER,
		mu:             sync.Mutex{},
	}
}

func (m *impl) RecurringSyncFunc() func(ctx context.Context, isLeader leader.Checker) error {
	return m.recurringFunc("sync of unseen changes", m.syncNotLoaded)
}

func (m *impl) RecurringSyncEnforceFunc() func(ctx context.Context, isLeader leader.Checker) error {
	return m.recurringFunc("full sync check", m.enforceSync)
}

// Returns a sync-related function
func (m *impl) recurringFunc(description string, ifStillLeader func(ctx context.Context) error) func(ctx context.Context, leaderChecker leader.Checker) error {
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
func (m *impl) stopAll() {
	for _, t := range m.scheduledTasks {
		m.unscheduleAndRemoveFromScheduledTasksState(t)
	}
}

// Stops all recurring Tasks and removes them from the in-memory
// map, then reads them from the backing store and schedules them
// and loads them in memory
func (m *impl) completeReload(ctx context.Context) error {
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
func (m *impl) syncNotLoaded(ctx context.Context) error {
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

	scheduled := make([]Task, 0, len(newOrUpdatedTasks))
	// handle new and updateds by unscheduling if present, then scheduling
	for _, updated := range newOrUpdatedTasks {
		m.unscheduleAndRemoveFromScheduledTasksState(updated)
		m.scheduleAndAddToList(updated, &scheduled)
	}

	// "acked" just means we've taken it into account, hence why we add deleteds in there as well.
	acked := append(deletedTasks, scheduled...)
	return m.markAsLoadedAndUpdateScheduledTasksState(ctx, acked)
}

func (m *impl) enforceSync(ctx context.Context) error {
	log.Info().Msg("Checking if scheduled Recurring Tasks are in sync with data store.")

	checkResults, err := m.checkSync(ctx)
	if err != nil {
		return err
	}

	if needsResync := checkResults.needsResync(); needsResync {
		log.Warn().Bool("needsResync", needsResync).Msg("Initiating optimistic reload of just the things that are out of sync.")

		// unschedule things that are not in the data store
		for _, deleted := range checkResults.notInDataStore {
			m.unscheduleAndRemoveFromScheduledTasksState(deleted)
		}

		acked := make([]Task, 0, len(checkResults.notInMemory)+len(checkResults.versionMismatch))
		for _, t := range append(checkResults.notInMemory, checkResults.versionMismatch...) {
			m.unscheduleAndRemoveFromScheduledTasksState(t)
			m.scheduleAndAddToList(t, &acked)
		}
		if err := m.markAsLoadedAndUpdateScheduledTasksState(ctx, acked); err != nil {
			return err
		}

		// do another check
		postOptimisticCheckSyncResults, err := m.checkSync(ctx)
		if err != nil {
			return err
		}
		if needsResync := postOptimisticCheckSyncResults.needsResync(); needsResync {
			log.Warn().Bool("needsResync", needsResync).Msg("Initiating *FULL* reload because optimistic reload was not enough...")
			return m.completeReload(ctx)
		} else {
			log.Info().Bool("needsResync", needsResync).Msg("Sweet. Optimistic reload was enough.")
			return nil
		}
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
func (m *impl) scheduleAndAddToList(t Task, loaded *[]Task) {
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
func (m *impl) unscheduleAndRemoveFromScheduledTasksState(t Task) {
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

func (m *impl) markAsLoadedAndUpdateScheduledTasksState(ctx context.Context, loaded []Task) error {
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

		// Stuff non-deleted Tasks that failed to update with anything but "not found" into the internal
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

func (m *impl) checkSync(ctx context.Context) (*syncCheckResults, error) {
	all, err := m.service.All(ctx)
	if err != nil {
		return nil, err
	}
	results := syncCheckResults{}
	// check if what we have in the data store is different from in memory
	idsFromDataStore := make(map[Id]void, len(all))
	for _, fromDataStore := range all {
		idsFromDataStore[fromDataStore.ID] = member
		if inMemory, present := m.scheduledTasks[fromDataStore.ID]; !present {
			results.addToNotInMemory(fromDataStore)
		} else {
			if inMemory.Metadata.Version != fromDataStore.Metadata.Version {
				results.addToVersionMismatch(fromDataStore)
			}
		}
	}

	// check if we have things in memory that are not in the data store
	for fromInMemoryId, fromInMemoryTask := range m.scheduledTasks {
		if _, present := idsFromDataStore[fromInMemoryId]; !present {
			results.addToNotInDataStore(fromInMemoryTask)
		}
	}

	return &results, nil
}

type syncCheckResults struct {
	// Tasks that are in memory but absent from the data store
	notInDataStore []Task
	// Tasks that are from and present in the data store but not in memory
	notInMemory []Task
	// Tasks that are from and present in the data store but have a different version than the one in memory
	versionMismatch []Task
}

func (s *syncCheckResults) addToNotInDataStore(task Task) {
	s.notInDataStore = append(s.notInDataStore, task)
}

func (s *syncCheckResults) addToNotInMemory(task Task) {
	s.notInMemory = append(s.notInMemory, task)
}

func (s *syncCheckResults) addToVersionMismatch(task Task) {
	s.versionMismatch = append(s.versionMismatch, task)
}

// Returns true if things are out of sync
//
// Yes, we're printing in a function that is used as a boolean, but in both usages of this, we want it to and
// this is an internal method so 🤷🏻‍♂️
func (s *syncCheckResults) needsResync() bool {
	needsResync := false
	if len(s.notInDataStore) != 0 {
		log.Warn().
			Int("count", len(s.notInDataStore)).
			Msg("Detected Tasks that are in memory, but not persisted")
		needsResync = true
	}
	if len(s.notInMemory) != 0 {
		log.Warn().
			Int("count", len(s.notInMemory)).
			Msg("Detected Tasks that are persisted, but not loaded in memory")
		needsResync = true
	}
	if len(s.versionMismatch) != 0 {
		log.Warn().
			Int("count", len(s.versionMismatch)).
			Msg("Detected Tasks that are in memory, but have version mismatches vs what is persisted")
		needsResync = true
	}
	return needsResync
}
