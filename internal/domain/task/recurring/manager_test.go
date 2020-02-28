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
