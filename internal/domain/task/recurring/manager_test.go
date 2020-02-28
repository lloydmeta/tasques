package recurring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	unscheduledIds     []Id
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

func (m *mockScheduler) Unschedule(taskId Id) bool {
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
