package recurring

import (
	"time"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

// Swag, the Swagger def parser has a bug that prevents us from directly using the one
// stored in the common package
type Duration time.Duration

// The actual Task that gets enqueued
type TaskDefinition struct {
	// The queue that a Task will be inserted into
	Queue queue.Name `json:"queue" binding:"required,queueName" example:"run-later"`
	// The number of times that a Task will be retried if it fails
	// If not passed, falls back to a server-side configured default
	RetryTimes *task.RetryTimes `json:"retry_times,omitempty" example:"10"`
	// The kind of Task; corresponds roughly with a function name
	Kind task.Kind `json:"kind" binding:"required" example:"sayHello"`
	// The priority of this Task (higher means higher priority)
	// If not passed, defaults to zero (neutral)
	Priority *task.Priority `json:"priority,omitempty"`
	// How long a Worker has upon claiming this Task to finish or report back before it gets timed out by the Tasques server
	// If not passed, falls back to a server-side configured default
	ProcessingTimeout *Duration `json:"processing_timeout,omitempty" swaggertype:"string" example:"30m"`
	// Arguments for this Task
	Args *task.Args `json:"args,omitempty" swaggertype:"object"`
	// Context for this Task
	Context *task.Context `json:"context,omitempty" swaggertype:"object"`
}

// A recurring Task that is yet to be persisted.
//
// Once registered, a Task will be enqueued at intervals as speciried
// by the schedule expression
type NewTask struct {
	// User-definable Id for the recurring Task. Must not collide with other existing ones.
	ID task.RecurringTaskId `json:"id" binding:"required"`
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	ScheduleExpression recurring.ScheduleExpression `json:"schedule_expression" binding:"required,scheduleExpression" example:"@every 1m"`
	// The Task to insert at intervals defined by ScheduleExpression
	TaskDefinition TaskDefinition `json:"task_definition" binding:"required"`
}

// Update definition for an existing Task
type TaskUpdate struct {
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	// If not defined, reuses the existing one on the recurring Task
	ScheduleExpression *recurring.ScheduleExpression `json:"schedule_expression,omitempty" binding:"omitempty,scheduleExpression" example:"@every 1m"`
	// The Task to insert at intervals defined by ScheduleExpression
	// If not defined, reuses the existing one on the recurring Task
	TaskDefinition *TaskDefinition `json:"task_definition,omitempty"`
}

// A persisted recurring TAsk
type Task struct {
	// User-defined Id for the recurring Task. Must not collide with other existing ones.
	ID task.RecurringTaskId `json:"id" binding:"required"`
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	ScheduleExpression recurring.ScheduleExpression `json:"schedule_expression" binding:"required,scheduleExpression" example:"@every 1m"`
	// The Task to insert at intervals defined by ScheduleExpression
	TaskDefinition TaskDefinition `json:"task_definition" binding:"required"`
	// When this recurring Task was last acknoledged and _loaded_ by a Tasques server for later
	// automatic enqueueing
	LoadedAt *time.Time `json:"loaded_at,omitempty"`
	// Metadata (data about data)
	Metadata common.Metadata `json:"metadata" binding:"required"`
}

// Converts an API model to the domain model
func (t *NewTask) ToDomainNewTask(defaultRetryTimes uint, defaultProcessingTimeout time.Duration) recurring.NewTask {
	return recurring.NewTask{
		ID:                 t.ID,
		ScheduleExpression: t.ScheduleExpression,
		TaskDefinition:     t.TaskDefinition.ToDomainTaskDefinition(defaultRetryTimes, defaultProcessingTimeout),
	}
}

func FromDomainTask(task *recurring.Task) Task {
	return Task{
		ID:                 task.ID,
		ScheduleExpression: task.ScheduleExpression,
		TaskDefinition:     fromDomainTaskDefinition(&task.TaskDefinition),
		LoadedAt:           (*time.Time)(task.LoadedAt),
		Metadata:           common.FromDomainMetadata(&task.Metadata),
	}
}

func fromDomainTaskDefinition(def *recurring.TaskDefinition) TaskDefinition {
	processingTimeout := Duration(time.Duration(def.ProcessingTimeout))
	priority := def.Priority
	retryTimes := def.RetryTimes

	return TaskDefinition{
		Queue:             def.Queue,
		RetryTimes:        &retryTimes,
		Kind:              def.Kind,
		Priority:          &priority,
		ProcessingTimeout: &processingTimeout,
		Args:              def.Args,
		Context:           def.Context,
	}
}

func (t *TaskDefinition) ToDomainTaskDefinition(defaultRetryTimes uint, defaultProcessingTimeout time.Duration) recurring.TaskDefinition {
	var domainRetryTimes task.RetryTimes
	if t.RetryTimes != nil {
		domainRetryTimes = *t.RetryTimes
	} else {
		domainRetryTimes = task.RetryTimes(defaultRetryTimes)
	}
	var domainPriority task.Priority
	if t.Priority != nil {
		domainPriority = *t.Priority
	} else {
		domainPriority = task.Priority(0)
	}
	var processingTimeout task.ProcessingTimeout
	if t.ProcessingTimeout != nil {
		processingTimeout = task.ProcessingTimeout(*t.ProcessingTimeout)
	} else {
		processingTimeout = task.ProcessingTimeout(defaultProcessingTimeout)
	}

	return recurring.TaskDefinition{
		Queue:             t.Queue,
		RetryTimes:        domainRetryTimes,
		Kind:              t.Kind,
		Priority:          domainPriority,
		ProcessingTimeout: processingTimeout,
		Args:              t.Args,
		Context:           t.Context,
	}
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	return (*common.Duration)(d).UnmarshalJSON(b)
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return (common.Duration)(d).MarshalJSON()
}
