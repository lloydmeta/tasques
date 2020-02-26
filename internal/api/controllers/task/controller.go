package task

import (
	"context"
	"net/http"
	"time"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	domainTask "github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"

	"github.com/lloydmeta/tasques/internal/api/models/task"
)

// Controller is an interface that defines the methods that are available to the routing
// layer. It is framework-agnostic
type Controller interface {

	// Create returns s a Task based on the passed-in NewTask
	//
	// Never pass a nil here; it's a pointer because the struct isn't small
	Create(ctx context.Context, newTask *task.NewTask) (*task.Task, *common.ApiError)

	// Get returns a Task based on the passed in queue and taskId
	Get(ctx context.Context, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError)

	// Claim attempts to claim a number of tasks for te given workerId, blocking for a certain amount of time.
	Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]task.Task, *common.ApiError)

	// ReportIn files a report on a running Task owned by a worker identified by the given workerId
	ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, report task.NewReport) (*task.Task, *common.ApiError)

	// MarkDone marks a Task as Done.
	MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Success) (*task.Task, *common.ApiError)

	// MarkFailed marks a Task as failed.
	MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Failure) (*task.Task, *common.ApiError)

	// Unclaim puts a Task that currently belongs to a worker with the given workerId back on the work queue
	UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError)
}

func New(tasksService domainTask.Service, tasksConfig config.TasksDefaults) Controller {
	return &impl{
		tasksService: tasksService,
		tasksConfig:  tasksConfig,
		getNowUtc: func() time.Time {
			return time.Now().UTC()
		},
	}
}

type impl struct {
	tasksService domainTask.Service
	tasksConfig  config.TasksDefaults
	getNowUtc    func() time.Time
}

func (c *impl) Create(ctx context.Context, newTask *task.NewTask) (*task.Task, *common.ApiError) {
	domainNewTask := newTask.ToDomainNewTask(c.tasksConfig.RetryTimes, c.getNowUtc(), c.tasksConfig.WorkerProcessingTimeout)
	result, err := c.tasksService.Create(ctx, &domainNewTask)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) Get(ctx context.Context, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError) {
	result, err := c.tasksService.Get(ctx, queue, taskId)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]task.Task, *common.ApiError) {
	result, err := c.tasksService.Claim(ctx, workerId, queues, number, blockFor)
	if err != nil {
		return nil, handleErr(err)
	} else {
		apiTasks := make([]task.Task, 0, len(result))
		for _, dTask := range result {
			apiTasks = append(apiTasks, task.FromDomainTask(&dTask))
		}
		return apiTasks, nil
	}
}

func (c *impl) ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, report task.NewReport) (*task.Task, *common.ApiError) {
	domainNewReport := domainTask.NewReport(report)
	result, err := c.tasksService.ReportIn(ctx, workerId, queue, taskId, domainNewReport)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Success) (*task.Task, *common.ApiError) {
	result, err := c.tasksService.MarkDone(ctx, workerId, queue, taskId, success)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, failure *domainTask.Failure) (*task.Task, *common.ApiError) {
	result, err := c.tasksService.MarkFailed(ctx, workerId, queue, taskId, failure)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError) {
	result, err := c.tasksService.UnClaim(ctx, workerId, queue, taskId)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := task.FromDomainTask(result)
		return &t, nil
	}
}

func handleErr(err error) *common.ApiError {
	switch v := err.(type) {
	case domainTask.NotFound:
		return notFound(v)
	case domainTask.ReportFromThePast:
		return reportFromthePast(v)
	case domainTask.InvalidPersistedData:
		return invalidPersistedData(v)
	case domainTask.NotClaimed:
		return notClaimed(v)
	case domainTask.NotOwnedByWorker:
		return notOwnedByWorker(v)
	case domainTask.InvalidVersion:
		return versionConflict(v)
	default:
		return unhandledErr(v)
	}
}

func notClaimed(notClaimed domainTask.NotClaimed) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusBadRequest,
		Body: common.Body{
			Message: notClaimed.Error(),
		},
	}
}

func reportFromthePast(past domainTask.ReportFromThePast) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusBadRequest,
		Body: common.Body{
			Message: past.Error(),
		},
	}
}

func notOwnedByWorker(notOwnedByWorker domainTask.NotOwnedByWorker) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusForbidden,
		Body: common.Body{
			Message: notOwnedByWorker.Error(),
		},
	}
}

func notFound(notFound domainTask.NotFound) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusNotFound,
		Body: common.Body{
			Message: notFound.Error(),
		},
	}
}

func versionConflict(versionConflict domainTask.InvalidVersion) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: versionConflict.Error(),
		},
	}
}

func invalidPersistedData(err domainTask.InvalidPersistedData) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusInternalServerError,
		Body: common.Body{
			Message: err.Error(),
		},
	}
}
func unhandledErr(e error) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusInternalServerError,
		Body: common.Body{
			Message: e.Error(),
		},
	}
}
