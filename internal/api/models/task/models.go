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

type NewTask struct {
	Queue             domainQueue.Name `json:"queue" binding:"required,queueName" example:"run-later"`
	RetryTimes        *task.RetryTimes `json:"retry_times,omitempty" example:"10"`
	Kind              task.Kind        `json:"kind" binding:"required" example:"sayHello"`
	Priority          *task.Priority   `json:"priority,omitempty"`
	ProcessingTimeout *Duration        `json:"processing_timeout,omitempty" swaggertype:"string" example:"30m"`
	RunAt             *time.Time       `json:"run_at,omitempty" swaggertype:"string" format:"date-time"`
	Args              *task.Args       `json:"args,omitempty" swaggertype:"object"`
	Context           *task.Context    `json:"context,omitempty" swaggertype:"object"`
}

// A Report to be filed on a Job as sent in from the owning Worker
type NewReport struct {
	Data *task.ReportedData `json:"data,omitempty"`
}

// A Report on a Job as sent in from the owning Worker
type Report struct {
	At   time.Time          `json:"at" binding:"required" swaggertype:"string" format:"date-time"`
	Data *task.ReportedData `json:"data,omitempty" swaggertype:"object"`
}

type Result struct {
	At time.Time `json:"at" binding:"required" swaggertype:"string" format:"date-time"`
	// Results. Only one of the following will be filled in at a given time
	Failure *task.Failure `json:"failure,omitempty" swaggertype:"object"`
	Success *task.Success `json:"success,omitempty" swaggertype:"object"`
}

type Claim struct {
	Queues   []domainQueue.Name `json:"queues,omitempty" binding:"required,dive,queueName" swaggertype:"array,string" example:"run-later,resize-images"`
	Amount   *uint              `json:"amount,omitempty" example:"1"`
	BlockFor *Duration          `json:"block_for,omitempty" swaggertype:"string" example:"1s"`
}

type Success struct {
	Data *task.Success `json:"data,omitempty" swaggertype:"object"`
}

type Failure struct {
	Data *task.Failure `json:"data,omitempty" swaggertype:"object"`
}

type LastClaimed struct {
	WorkerId   worker.Id `json:"worker_id" swaggertype:"string" binding:"required"`
	ClaimedAt  time.Time `json:"claimed_at" binding:"required" swaggertype:"string" format:"date-time"`
	TimesOutAt time.Time `json:"times_out_at" binding:"required" swaggertype:"string" format:"date-time"`
	LastReport *Report   `json:"last_report,omitempty"`
	Result     *Result   `json:"result,omitempty"`
}

type Task struct {
	ID                task.Id             `json:"id" swaggertype:"string" binding:"required"`
	Queue             domainQueue.Name    `json:"queue" binding:"required,queueName" example:"run-later"`
	RetryTimes        task.RetryTimes     `json:"retry_times" binding:"required" example:"10"`
	Attempted         task.AttemptedTimes `json:"attempted" binding:"required"`
	Kind              task.Kind           `json:"kind" binding:"required" example:"sayHello"`
	State             task.State          `json:"state" binding:"required" swaggertype:"string" example:"queued"`
	Priority          task.Priority       `json:"priority" binding:"required"`
	ProcessingTimeout Duration            `json:"processing_timeout" binding:"required" swaggertype:"string" example:"30m"`
	RunAt             time.Time           `json:"run_at" binding:"required" swaggertype:"string" format:"date-time"`
	Args              *task.Args          `json:"args,omitempty" swaggertype:"object"`
	Context           *task.Context       `json:"context,omitempty" swaggertype:"object"`
	LastClaimed       *LastClaimed        `json:"last_claimed,omitempty"`
	LastEnqueuedAt    time.Time           `json:"last_enqueued_at" binding:"required" swaggertype:"string" format:"date-time"`
	Metadata          common.Metadata     `json:"metadata" binding:"required"`
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
	}
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	return (*common.Duration)(d).UnmarshalJSON(b)
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return (common.Duration)(d).MarshalJSON()
}
