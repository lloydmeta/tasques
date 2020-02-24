package recurring

import (
	"time"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

// A user-specifiable Id
type Id string

// The actual recurring task that gets inserted
type TaskDefinition struct {
	Queue             queue.Name
	RetryTimes        task.RetryTimes
	Kind              task.Kind
	Priority          task.Priority
	ProcessingTimeout task.ProcessingTimeout
	Args              *task.Args
	Context           *task.Context
}

// A cron-like statement. Just a typed version of whatever
// can be parsed by our actual scheduling infra lib...
type ScheduleExpression string

// When a given RecurringTask was last "loaded" by a scheduling
// process.
type LoadedAt time.Time

// Soft-delete
type IsDeleted bool

// A Task that is yet to be persisted
// We assume that the ScheduleExpression is valid
// *before* we persist it
type NewRecurringTask struct {
	Id                 Id
	ScheduleExpression ScheduleExpression
	TaskDefinition     TaskDefinition
}

// The way this is structured is somewhat odd; why soft deletes?
// Why "loadedAt"?
//
// There is some form of leaky abstraction going on here: saving it this
// way allows us to easily atomically store data and query it.
// Given good ES integration is an explicit goal, I *think* at this point
// that it's ok to leak a bit of that into the domain ..
type RecurringTask struct {
	Id                 Id
	ScheduleExpression ScheduleExpression
	TaskDefinition     TaskDefinition
	IsDeleted          IsDeleted
	LoadedAt           *LoadedAt
	Metadata           metadata.Metadata
}

func (r *RecurringTask) UpdateSchedule(expression ScheduleExpression) {
	r.ScheduleExpression = expression
	r.LoadedAt = nil
}

func (r *RecurringTask) UpdateTaskDefinition(definition TaskDefinition) {
	r.TaskDefinition = definition
	r.LoadedAt = nil
}

func (r *RecurringTask) IntoDeleted() {
	r.IsDeleted = true
	r.LoadedAt = nil
}
