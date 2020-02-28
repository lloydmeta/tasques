package recurring

import (
	"context"
	"testing"
	"time"

	"github.com/jinzhu/copier"

	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/domain/leader"
	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

func TestNewManager(t *testing.T) {
	assert.NotPanics(t, func() {
		r := NewManager(&mockScheduler{}, &MockRecurringTasksService{})
		assert.NotNil(t, r)
	})
}

func Test_syncCheckResults(t *testing.T) {
	result := syncCheckResults{}
	assert.False(t, result.needsResync())
	task := Task{}
	result.addToNotInDataStore(task)
	assert.Len(t, result.notInDataStore, 1)
	assert.True(t, result.needsResync())
	result.addToVersionMismatch(task)
	assert.Len(t, result.versionMismatch, 1)
	assert.True(t, result.needsResync())
	result.addToNotInMemory(task)
	assert.Len(t, result.notInMemory, 1)
	assert.True(t, result.needsResync())
}

// <-- common basic sanity tests for RecurringSyncFunc and RecurringSyncEnforceFunc

func Test_CommonRecurringFunc_NewLeader(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(manager Manager) func(context.Context, leader.Checker) error
	}{
		{
			name: "RecurringSyncFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncFunc()
			},
		},
		{
			name: "RecurringSyncEnforceFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncEnforceFunc()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := mockScheduler{}
			service := MockRecurringTasksService{}
			m := impl{scheduler: &scheduler, service: &service}
			f := tt.getFunc(&m)
			leaderChecker := mockLeaderCheck{isLeader: true}
			err := f(ctx, &leaderChecker)
			if err != nil {
				t.Error(err)
			}
			assert.EqualValues(t, 1, service.AllCalled)
			assert.EqualValues(t, []Task{MockDomainRecurringTask}, scheduler.scheduledTasks)
			assert.EqualValues(t, 1, service.MarkLoadedCalled)
		})
	}
}

func Test_CommonRecurringFunc_NewlyNotLeader(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(manager Manager) func(context.Context, leader.Checker) error
	}{
		{
			name: "RecurringSyncFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncFunc()
			},
		},
		{
			name: "RecurringSyncEnforceFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncEnforceFunc()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := mockScheduler{}
			service := MockRecurringTasksService{}
			m := impl{
				scheduler:   &scheduler,
				service:     &service,
				leaderState: LEADER,
				scheduledTasks: map[task.RecurringTaskId]Task{
					MockDomainRecurringTask.ID: MockDomainRecurringTask,
				}}
			f := tt.getFunc(&m)
			leaderChecker := mockLeaderCheck{isLeader: false}
			err := f(ctx, &leaderChecker)
			if err != nil {
				t.Error(err)
			}
			assert.EqualValues(t, []task.RecurringTaskId{MockDomainRecurringTask.ID}, scheduler.unscheduledIds)
			assert.Empty(t, scheduler.scheduledTasks)
		})
	}
}

func Test_CommonRecurringFunc_StillNotLeader(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func(manager Manager) func(context.Context, leader.Checker) error
	}{
		{
			name: "RecurringSyncFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncFunc()
			},
		},
		{
			name: "RecurringSyncEnforceFunc",
			getFunc: func(manager Manager) func(context.Context, leader.Checker) error {
				return manager.RecurringSyncEnforceFunc()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := mockScheduler{}
			service := MockRecurringTasksService{}
			m := impl{
				scheduler:   &scheduler,
				service:     &service,
				leaderState: NOT_LEADER,
				scheduledTasks: map[task.RecurringTaskId]Task{
					MockDomainRecurringTask.ID: MockDomainRecurringTask,
				}}
			f := tt.getFunc(&m)
			leaderChecker := mockLeaderCheck{isLeader: false}
			err := f(ctx, &leaderChecker)
			if err != nil {
				t.Error(err)
			}
			// Nothing should have been called, but just check a few methods
			assert.EqualValues(t, 0, service.AllCalled)
			assert.EqualValues(t, 0, service.NotLoadedCalled)
			assert.Empty(t, scheduler.unscheduledIds)
		})
	}
}

//     common basic sanity tests for RecurringSyncFunc and RecurringSyncEnforceFunc -->

// <-- basic tests for the different parts of RecurringSyncFunc and RecurringSyncEnforceFunc

func Test_RecurringSyncFunc_StillLeader(t *testing.T) {
	scheduler := mockScheduler{}
	service := MockRecurringTasksService{}
	m := impl{scheduler: &scheduler, service: &service, leaderState: LEADER}
	f := m.RecurringSyncFunc()
	leaderChecker := mockLeaderCheck{isLeader: true}
	err := f(ctx, &leaderChecker)
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, 1, service.NotLoadedCalled)
	assert.EqualValues(t, []Task{MockDomainRecurringTask}, scheduler.scheduledTasks)
	assert.EqualValues(t, 1, service.MarkLoadedCalled)
}

func Test_RecurringSyncEnforceFunc_StillLeader(t *testing.T) {
	scheduler := mockScheduler{}
	service := MockRecurringTasksService{}
	m := impl{scheduler: &scheduler, service: &service, leaderState: LEADER}
	f := m.RecurringSyncEnforceFunc()
	leaderChecker := mockLeaderCheck{isLeader: true}
	err := f(ctx, &leaderChecker)
	if err != nil {
		t.Error(err)
	}
	// The following is the worst case scenario
	// 1st for initial load, 2nd for check, 3rd for full reload
	assert.EqualValues(t, 3, service.AllCalled)
	// 1st from initial load, 2nd from full reload
	assert.EqualValues(t, []Task{MockDomainRecurringTask, MockDomainRecurringTask}, scheduler.scheduledTasks)
	// 1st from initial load, 2nd from full reload
	assert.EqualValues(t, 2, service.MarkLoadedCalled)
}

//    basic tests for the different parts of RecurringSyncFunc and RecurringSyncEnforceFunc -->

func Test_checkSync(t *testing.T) {

	inMemNotInStore := Task{}
	if err := copier.Copy(&inMemNotInStore, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	inMemNotInStore.ID = "in-mem-not-in-store"

	inMemWrongVersion := Task{}
	if err := copier.Copy(&inMemWrongVersion, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	inMemWrongVersion.ID = "wrong-version"

	inStoreWrongVersion := Task{}
	if err := copier.Copy(&inStoreWrongVersion, &inMemWrongVersion); err != nil {
		t.Error(err)
	}
	inStoreWrongVersion.Metadata.Version = metadata.Version{
		SeqNum:      10101010101,
		PrimaryTerm: 90909090909,
	}

	inStoreNotInMem := Task{}
	if err := copier.Copy(&inStoreNotInMem, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	inStoreNotInMem.ID = "in-store-not-in-mem"

	scheduler := mockScheduler{}
	service := MockRecurringTasksService{}

	service.AllOverride = func() (tasks []Task, err error) {
		return []Task{inStoreWrongVersion, inStoreNotInMem}, nil
	}

	m := impl{
		scheduler:   &scheduler,
		service:     &service,
		leaderState: LEADER,
		scheduledTasks: map[task.RecurringTaskId]Task{
			inMemWrongVersion.ID: inMemWrongVersion,
			inMemNotInStore.ID:   inMemNotInStore,
		},
	}

	r, err := m.checkSync(ctx)
	if err != nil {
		t.Error(err)
	}

	assert.EqualValues(t, 1, service.AllCalled)
	assert.EqualValues(t, []Task{inStoreNotInMem}, r.notInMemory)
	assert.EqualValues(t, []Task{inStoreWrongVersion}, r.versionMismatch)
	assert.EqualValues(t, []Task{inMemNotInStore}, r.notInDataStore)
}

func Test_markAsLoadedAndUpdateScheduledTasksState_empty(t *testing.T) {
	scheduler := mockScheduler{}
	service := MockRecurringTasksService{}
	m := impl{
		scheduler: &scheduler,
		service:   &service,
	}
	err := m.markAsLoadedAndUpdateScheduledTasksState(ctx, nil)
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, 0, service.MarkLoadedCalled)
}

func Test_markAsLoadedAndUpdateScheduledTasksState_nonEmpty_mixed_MultiUpdateResult(t *testing.T) {
	scheduler := mockScheduler{}
	service := MockRecurringTasksService{}

	successNotDeleted := Task{} // Same as MockDomainRecurringTask except with a different version
	if err := copier.Copy(&successNotDeleted, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	successNotDeleted.Metadata.Version = metadata.Version{
		SeqNum:      11111,
		PrimaryTerm: 22222,
	}

	successDeleted := Task{} // Same as MockDomainRecurringTask except with a different version
	if err := copier.Copy(&successDeleted, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	successDeleted.IsDeleted = true
	successDeleted.Metadata.Version = metadata.Version{
		SeqNum:      11111,
		PrimaryTerm: 22222,
	}
	successDeleted.ID = "success-deleted"

	notFoundErrTask := Task{}
	if err := copier.Copy(&notFoundErrTask, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	notFoundErrTask.ID = "not-in-store"

	versionConflictErrTask := Task{}
	if err := copier.Copy(&versionConflictErrTask, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	versionConflictErrTask.ID = "version-conflict"

	otherErrTaskNotDeleted := Task{}
	if err := copier.Copy(&otherErrTaskNotDeleted, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	otherErrTaskNotDeleted.ID = "other-err-not-deleted"

	otherErrTaskDeleted := Task{}
	if err := copier.Copy(&otherErrTaskDeleted, &MockDomainRecurringTask); err != nil {
		t.Error(err)
	}
	otherErrTaskDeleted.IsDeleted = true
	otherErrTaskDeleted.ID = "other-err-deleted"

	service.MarkLoadedOverride = func() (result *MultiUpdateResult, err error) {
		return &MultiUpdateResult{
			Successes:        []Task{successNotDeleted, successDeleted},
			VersionConflicts: []Task{versionConflictErrTask},
			NotFounds:        []Task{notFoundErrTask},
			Others: []BulkUpdateOtherError{{
				RecurringTask: otherErrTaskNotDeleted,
				Result:        "no clue",
			}},
		}, nil
	}

	m := impl{
		scheduler:      &scheduler,
		service:        &service,
		scheduledTasks: make(map[task.RecurringTaskId]Task),
	}
	err := m.markAsLoadedAndUpdateScheduledTasksState(ctx, []Task{MockDomainRecurringTask})
	if err != nil {
		t.Error(err)
	}
	assert.EqualValues(t, 1, service.MarkLoadedCalled)
	assert.EqualValues(t, map[task.RecurringTaskId]Task{
		successNotDeleted.ID:      successNotDeleted,
		versionConflictErrTask.ID: versionConflictErrTask,
		otherErrTaskNotDeleted.ID: otherErrTaskNotDeleted,
	}, m.scheduledTasks)
}

var ctx = context.Background()

type mockLeaderCheck struct {
	isLeader bool
}

func (m *mockLeaderCheck) IsLeader() bool {
	return m.isLeader
}

type mockSchedule struct {
	next time.Time
}

func (m *mockSchedule) Next(t time.Time) time.Time {
	return m.next
}

var now = time.Now().UTC()

var dummySchedule = mockSchedule{
	next: now,
}

type mockScheduler struct {
	parseCalled   uint
	parseOverride func() (Schedule, error)

	scheduledTasks   []Task
	scheduleOverride func() error

	unscheduledIds     []task.RecurringTaskId
	unscheduleOverride func() bool
}

func (m *mockScheduler) Parse(spec string) (Schedule, error) {
	m.parseCalled++
	if m.parseOverride != nil {
		return m.parseOverride()
	} else {
		return &dummySchedule, nil
	}
}

func (m *mockScheduler) Schedule(task Task) error {
	m.scheduledTasks = append(m.scheduledTasks, task)
	if m.scheduleOverride != nil {
		return m.scheduleOverride()
	} else {
		return nil
	}
}

func (m *mockScheduler) Unschedule(taskId task.RecurringTaskId) bool {
	m.unscheduledIds = append(m.unscheduledIds, taskId)
	if m.unscheduleOverride != nil {
		return m.unscheduleOverride()
	} else {
		return true
	}
}

func (m *mockScheduler) Start() {
}

func (m *mockScheduler) Stop() {
}
