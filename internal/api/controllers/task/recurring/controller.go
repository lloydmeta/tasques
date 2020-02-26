package recurring

import (
	"context"
	"net/http"

	"github.com/lloydmeta/tasques/internal/api/models/common"

	"github.com/lloydmeta/tasques/internal/api/models/task/recurring"
	"github.com/lloydmeta/tasques/internal/config"
	domainRecurring "github.com/lloydmeta/tasques/internal/domain/task/recurring"
)

// TODO add tests
type Controller interface {

	// Create returns a Task based on the passed in NewTask
	Create(ctx context.Context, task *recurring.NewTask) (*recurring.Task, *common.ApiError)

	// Update updates and returns a Task based on the passed in NewTask
	Update(ctx context.Context, id domainRecurring.Id, task *recurring.TaskUpdate) (*recurring.Task, *common.ApiError)

	// Get returns a Task by Id
	Get(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError)

	// Delete deletes and returns a Task by Id
	Delete(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError)

	// List returns all tasks
	List(ctx context.Context) ([]*recurring.Task, *common.ApiError)
}

type impl struct {
	recurringTasksService domainRecurring.Service

	tasksConfig config.TasksDefaults
}

func (c *impl) Create(ctx context.Context, task *recurring.NewTask) (*recurring.Task, *common.ApiError) {
	domainNewTask := task.ToDomainNewTask(c.tasksConfig.RetryTimes, c.tasksConfig.WorkerProcessingTimeout)
	result, err := c.recurringTasksService.Create(ctx, &domainNewTask)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := recurring.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) Update(ctx context.Context, id domainRecurring.Id, task *recurring.TaskUpdate) (*recurring.Task, *common.ApiError) {
	toUpdate, err := c.recurringTasksService.Get(ctx, id, false)
	if err != nil {
		return nil, handleErr(err)
	} else {
		// attempt update
		if task.ScheduleExpression != nil {
			toUpdate.ScheduleExpression = *task.ScheduleExpression
		}
		if task.TaskDefinition != nil {
			toUpdate.TaskDefinition = task.TaskDefinition.ToDomainTaskDefinition(c.tasksConfig.RetryTimes, c.tasksConfig.WorkerProcessingTimeout)
		}
		updated, err := c.recurringTasksService.Update(ctx, toUpdate)
		if err != nil {
			return nil, handleErr(err)
		} else {
			t := recurring.FromDomainTask(updated)
			return &t, nil
		}
	}
}

func (c *impl) Get(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError) {
	result, err := c.recurringTasksService.Get(ctx, id, false)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := recurring.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) Delete(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError) {
	result, err := c.recurringTasksService.Delete(ctx, id)
	if err != nil {
		return nil, handleErr(err)
	} else {
		t := recurring.FromDomainTask(result)
		return &t, nil
	}
}

func (c *impl) List(ctx context.Context) ([]*recurring.Task, *common.ApiError) {
	panic("implement me")
}

func handleErr(err error) *common.ApiError {
	switch v := err.(type) {
	case domainRecurring.NotFound:
		return notFound(v)
	case domainRecurring.InvalidPersistedData:
		return invalidPersistedData(v)
	case domainRecurring.InvalidVersion:
		return versionConflict(v)
	default:
		return unhandledErr(v)
	}
}

func notFound(notFound domainRecurring.NotFound) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusNotFound,
		Body: common.Body{
			Message: notFound.Error(),
		},
	}
}

func versionConflict(versionConflict domainRecurring.InvalidVersion) *common.ApiError {
	return &common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: versionConflict.Error(),
		},
	}
}

func invalidPersistedData(err domainRecurring.InvalidPersistedData) *common.ApiError {
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
