// task holds the API Task models. At the moment, the fields here translate one to one
// with the domain Task model, except where we need to use different field types such
// as our custom Duration to allow for friendly serialising and deserialising.
package task

import (
	"time"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	domainQueue "github.com/lloydmeta/tasques/internal/domain/queue"
	"github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

// Swag, the Swagger def parser has a bug that prevents us from directly using the one
// stored in the common package
type Duration time.Duration

// NewTask holds a description of a Task that is yet to be persisted
type NewTask struct {
	// The queue that a Task will be inserted into
	Queue domainQueue.Name `json:"queue" binding:"required,queueName" example:"run-later"`
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
	// If defined, when this Task should run
	// If not passed, falls back to now.
	RunAt *time.Time `json:"run_at,omitempty" swaggertype:"string" format:"date-time"`
	// Arguments for this Task
	Args *task.Args `json:"args,omitempty" swaggertype:"object"`
	// Context for this Task
	Context *task.Context `json:"context,omitempty" swaggertype:"object"`
}

// A Report to be filed on a Job as sent in from the owning Worker
type NewReport struct {
	// Optional data for the report
	Data *task.ReportedData `json:"data,omitempty"`
}

// A Report on a Job as sent in from the owning Worker
type Report struct {
	// When the report was filed
	At time.Time `json:"at" binding:"required" swaggertype:"string" format:"date-time"`
	// Optional report data
	Data *task.ReportedData `json:"data,omitempty" swaggertype:"object"`
}

// The Result of a Task being processed. Only one of Failure or Success will be present
type Result struct {
	// When the Result was produced
	At time.Time `json:"at" binding:"required" swaggertype:"string" format:"date-time"`
	// Failure
	Failure *task.Failure `json:"failure,omitempty" swaggertype:"object"`
	// Success
	Success *task.Success `json:"success,omitempty" swaggertype:"object"`
}

// Represents a request to claim Tasks by single worker
type Claim struct {
	// The Task queues to claim from
	Queues []domainQueue.Name `json:"queues,omitempty" binding:"required,dive,queueName" swaggertype:"array,string" example:"run-later,resize-images"`
	// How many Tasks to try to claim
	Amount *uint `json:"amount,omitempty" example:"1"`
	// How long to block for before retrying, if the specified amount cannot be claimed.
	// If not passed, falls back to a server-side configured default
	BlockFor *Duration `json:"block_for,omitempty" swaggertype:"string" example:"1s"`
}

// Represents successful processing of a Task
type Success struct {
	Data *task.Success `json:"data,omitempty" swaggertype:"object"`
}

// Represents failed processing of a Task
type Failure struct {
	Data *task.Failure `json:"data,omitempty" swaggertype:"object"`
}

// Holds data about the last time a Task was claimed
type LastClaimed struct {
	// Id belonging to a worker that claimed the Task
	WorkerId worker.Id `json:"worker_id" swaggertype:"string" binding:"required"`
	// When the claim was made
	ClaimedAt time.Time `json:"claimed_at" binding:"required" swaggertype:"string" format:"date-time"`
	// When the Task will be timed out if the worker doesn't finish or report back
	TimesOutAt time.Time `json:"times_out_at" binding:"required" swaggertype:"string" format:"date-time"`
	// The LastReport filed by a worker holding a claim on the Task
	LastReport *Report `json:"last_report,omitempty"`
	// The processing Result
	Result *Result `json:"result,omitempty"`
}

// A persisted Task
type Task struct {
	// Unique identifier of a Task
	ID task.Id `json:"id" swaggertype:"string" binding:"required"`
	// The queue the Task is in
	Queue domainQueue.Name `json:"queue" binding:"required,queueName" example:"run-later"`
	// The number of times that a Task will be retried if it fails
	RetryTimes task.RetryTimes `json:"retry_times" binding:"required" example:"10"`
	// The number of times a Task has been attempted
	Attempted task.AttemptedTimes `json:"attempted" binding:"required"`
	// The kind of Task; corresponds roughly with a function name
	Kind task.Kind `json:"kind" binding:"required" example:"sayHello"`
	// The state of a Task
	State task.State `json:"state" binding:"required" swaggertype:"string" example:"queued"`
	// The priority of this Task (higher means higher priority)
	Priority task.Priority `json:"priority" binding:"required"`
	// How long a Worker has upon claiming this Task to finish or report back before it gets timed out by the Tasques server
	ProcessingTimeout Duration `json:"processing_timeout" binding:"required" swaggertype:"string" example:"30m"`
	// When this Task should run
	RunAt time.Time `json:"run_at" binding:"required" swaggertype:"string" format:"date-time"`
	// Arguments for this Task
	Args *task.Args `json:"args,omitempty" swaggertype:"object"`
	// Context for this Task
	Context *task.Context `json:"context,omitempty" swaggertype:"object"`
	// Information on when this Task was last claimed by a worker
	LastClaimed *LastClaimed `json:"last_claimed,omitempty"`
	// When this Task was last enqueued
	LastEnqueuedAt time.Time `json:"last_enqueued_at" binding:"required" swaggertype:"string" format:"date-time"`
	// Metadata (data about data)
	Metadata common.Metadata `json:"metadata" binding:"required"`
	// Only populated if this is a Task that was spawned/enqueued by a Recurring Task definition
	RecurringTaskId *task.RecurringTaskId `json:"recurring_task_id,omitempty" swaggertype:"string"`
}

var TimeZero time.Time

// Converts an API model to the domain model
func (t *NewTask) ToDomainNewTask(defaultRetryTimes uint, defaultRunAt time.Time, defaultProcessingTimeout time.Duration) task.NewTask {
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
	var domainRunAt task.RunAt
	if t.RunAt != nil && *t.RunAt != TimeZero {
		domainRunAt = task.RunAt(*t.RunAt)
	} else {
		domainRunAt = task.RunAt(defaultRunAt)
	}
	var processingTimeout task.ProcessingTimeout
	if t.ProcessingTimeout != nil {
		processingTimeout = task.ProcessingTimeout(*t.ProcessingTimeout)
	} else {
		processingTimeout = task.ProcessingTimeout(defaultProcessingTimeout)
	}

	return task.NewTask{
		Queue:             t.Queue,
		RetryTimes:        domainRetryTimes,
		Kind:              t.Kind,
		Priority:          domainPriority,
		RunAt:             domainRunAt,
		ProcessingTimeout: processingTimeout,
		Args:              t.Args,
		Context:           t.Context,
	}
}

// Creates an API model from the domain model
func FromDomainTask(dTask *task.Task) Task {
	var lastClaimed *LastClaimed
	if dTask.LastClaimed != nil {
		var report *Report
		if dTask.LastClaimed.LastReport != nil {
			report = &Report{
				At:   time.Time(dTask.LastClaimed.LastReport.At),
				Data: dTask.LastClaimed.LastReport.Data,
			}
		}
		var result *Result
		if dTask.LastClaimed.Result != nil {
			result = &Result{
				At:      time.Time(dTask.LastClaimed.Result.At),
				Failure: dTask.LastClaimed.Result.Failure,
				Success: dTask.LastClaimed.Result.Success,
			}
		}
		lastClaimed = &LastClaimed{
			WorkerId:   dTask.LastClaimed.WorkerId,
			ClaimedAt:  time.Time(dTask.LastClaimed.ClaimedAt),
			TimesOutAt: time.Time(dTask.LastClaimed.TimesOutAt),
			LastReport: report,
			Result:     result,
		}
	}
	return Task{
		ID:                dTask.ID,
		Queue:             dTask.Queue,
		RetryTimes:        dTask.RetryTimes,
		Attempted:         dTask.Attempted,
		Kind:              dTask.Kind,
		State:             dTask.State,
		Priority:          dTask.Priority,
		RunAt:             time.Time(dTask.RunAt),
		Args:              dTask.Args,
		ProcessingTimeout: Duration(dTask.ProcessingTimeout),
		Context:           dTask.Context,
		LastClaimed:       lastClaimed,
		LastEnqueuedAt:    time.Time(dTask.LastEnqueuedAt),
		Metadata:          common.FromDomainMetadata(&dTask.Metadata),
		RecurringTaskId:   dTask.RecurringTaskId,
	}
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	return (*common.Duration)(d).UnmarshalJSON(b)
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return (common.Duration)(d).MarshalJSON()
}
