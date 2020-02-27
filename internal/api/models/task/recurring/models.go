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

// The actual recurring task that gets inserted
type TaskDefinition struct {
	Queue             queue.Name       `json:"queue" binding:"required,queueName" example:"run-later"`
	RetryTimes        *task.RetryTimes `json:"retry_times,omitempty" example:"10"`
	Kind              task.Kind        `json:"kind" binding:"required" example:"sayHello"`
	Priority          *task.Priority   `json:"priority,omitempty"`
	ProcessingTimeout *Duration        `json:"processing_timeout,omitempty" swaggertype:"string" example:"30m"`
	Args              *task.Args       `json:"args,omitempty" swaggertype:"object"`
	Context           *task.Context    `json:"context,omitempty" swaggertype:"object"`
}

// A Task that is yet to be persisted
// We assume that the ScheduleExpression is valid
// *before* we persist it
type NewTask struct {
	ID recurring.Id `json:"id" binding:"required"`
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	ScheduleExpression recurring.ScheduleExpression `json:"schedule_expression" binding:"required,scheduleExpression" example:"@every 1m"`
	TaskDefinition     TaskDefinition               `json:"task_definition" binding:"required"`
}

// Update definition for an existing Task
type TaskUpdate struct {
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	ScheduleExpression *recurring.ScheduleExpression `json:"schedule_expression,omitempty" binding:"scheduleExpression" example:"@every 1m"`
	TaskDefinition     *TaskDefinition               `json:"task_definition,omitempty"`
}

type Task struct {
	ID recurring.Id `json:"id" binding:"required"`
	// A schedule expression; can be any valid cron expression, with some support for simple macros
	ScheduleExpression recurring.ScheduleExpression `json:"schedule_expression" binding:"required,scheduleExpression" example:"@every 1m"`
	TaskDefinition     TaskDefinition               `json:"task_definition" binding:"required"`
	LoadedAt           *time.Time                   `json:"loaded_at,omitempty"`
	Metadata           common.Metadata              `json:"metadata" binding:"required"`
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
