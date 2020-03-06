package recurring

import (
	"time"

	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
)

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

// When a given Task was last "loaded" by a scheduling
// process.
//
// Note that "loaded" as a term is ... loaded here; it basically
// means that a Recurring Task's latest change has been "seen".
// For example, if it was (soft) deleted, we actually remove it
// from the Scheduler; but we update LoadedAt.
//
// This means that this field also acts as a "is_dirty" marker
type LoadedAt time.Time

// Soft-delete
type IsDeleted bool

// A Task that is yet to be persisted
// We assume that the ScheduleExpression is valid
// *before* we persist it
type NewTask struct {
	ID                 task.RecurringTaskId
	ScheduleExpression ScheduleExpression
	TaskDefinition     TaskDefinition
}

// The way this is structured is somewhat odd; why soft deletes?
// Why "loadedAt"?
//
// There is some form of leaky abstraction going on here: saving it this
// way allows us to easily atomically store data and query it.
//
// Given good ES integration is an explicit goal, I *think* at this point
// that it's ok to leak a bit of that into the domain ..
type Task struct {
	ID                 task.RecurringTaskId
	ScheduleExpression ScheduleExpression
	TaskDefinition     TaskDefinition
	IsDeleted          IsDeleted
	LoadedAt           *LoadedAt
	Metadata           metadata.Metadata
}

func (r *Task) UpdateSchedule(expression ScheduleExpression) {
	r.ScheduleExpression = expression
	r.LoadedAt = nil
}

func (r *Task) UpdateTaskDefinition(definition TaskDefinition) {
	r.TaskDefinition = definition
	r.LoadedAt = nil
}

func (r *Task) IntoDeleted() {
	r.IsDeleted = true
	r.LoadedAt = nil
}
