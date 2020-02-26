package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/api/models/task"
	"github.com/lloydmeta/tasques/internal/domain/metadata"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	domainTask "github.com/lloydmeta/tasques/internal/domain/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
	"github.com/lloydmeta/tasques/internal/infra/server/binding/validation"
	"github.com/lloydmeta/tasques/internal/infra/server/routing"
)

func init() {
	validation.SetUpValidators()
}

func workerHeaders() http.Header {
	h := http.Header{}
	h.Set(WorkerIdHeaderKey, "abc")
	return h
}

func setupRouter() (*gin.Engine, *mockTasksController) {
	engine := gin.Default()
	mockController := mockTasksController{}
	topLevelRouterGroup := routing.NewTopLevelRoutesGroup(nil, engine)
	handler := RoutesHandler{Controller: &mockController}
	handler.RegisterRoutes(topLevelRouterGroup)

	return engine, &mockController
}

func performRequest(r http.Handler, method, url string, body interface{}, header http.Header) *httptest.ResponseRecorder {
	var bodyToSend io.Reader
	if body != nil {
		asBytes, _ := json.Marshal(body)
		bodyToSend = bytes.NewBuffer(asBytes)
	}
	req, _ := http.NewRequest(method, url, bodyToSend)
	if header != nil {
		req.Header = header
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTaskCreate_Ok(t *testing.T) {
	router, mockController := setupRouter()
	runAt := now()
	newTask := task.NewTask{
		Queue: "q",
		Kind:  "run",
		Args: &domainTask.Args{
			"hello": "world",
		},
		RunAt: &runAt,
		Context: &domainTask.Context{
			"reqId": "abcs123",
		},
	}
	resp := performRequest(router, http.MethodPost, "/tasques", newTask, nil)
	assert.EqualValues(t, http.StatusCreated, resp.Code)
	assert.EqualValues(t, 1, mockController.createCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, newTask.Queue, respTask.Queue)
		assert.EqualValues(t, newTask.Kind, respTask.Kind)
		assert.EqualValues(t, newTask.RunAt, &respTask.RunAt)
		assert.EqualValues(t, newTask.Args, respTask.Args)
		assert.EqualValues(t, newTask.Context, respTask.Context)
	}
}

func TestTaskCreate_InvalidQueueName(t *testing.T) {
	router, mockController := setupRouter()
	runAt := now()
	newTask := task.NewTask{
		Queue: "+q",
		Kind:  "run",
		Args: &domainTask.Args{
			"hello": "world",
		},
		RunAt: &runAt,
		Context: &domainTask.Context{
			"reqId": "abcs123",
		},
	}
	resp := performRequest(router, http.MethodPost, "/tasques", newTask, nil)
	assert.EqualValues(t, 0, mockController.createCalled)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestTaskGet_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodGet, "/tasques/q/1", nil, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.getCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, "q", respTask.Queue)
		assert.EqualValues(t, "1", respTask.ID)
	}
}

func TestTaskGet_InvalidQueueName(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodGet, "/tasques/+q/1", nil, nil)
	assert.EqualValues(t, 0, mockController.getCalled)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestTaskGet_NotFound(t *testing.T) {
	router, mockController := setupRouter()
	err := common.ApiError{
		StatusCode: http.StatusNotFound,
	}
	mockController.getOverride = func(ctx context.Context, queue queue.Name, taskId domainTask.Id) (t *task.Task, apiError *common.ApiError) {
		return nil, &err
	}
	resp := performRequest(router, http.MethodGet, "/tasques/q/1", nil, nil)
	assert.Equal(t, err.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.getCalled)
}

func TestTaskClaim_Ok(t *testing.T) {
	router, mockController := setupRouter()
	amt := uint(1)
	claim := task.Claim{
		Queues: []queue.Name{"hello"},
		Amount: &amt,
	}
	resp := performRequest(router, http.MethodPost, "/tasques/claims", claim, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.claimCalled)
	var respTasks []task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTasks); err != nil {
		t.Error(err, resp)
	} else {
		respTask := respTasks[0]
		assert.EqualValues(t, "hello", respTask.Queue)
	}
}

func TestTaskClaim_Ok_ZeroWanted(t *testing.T) {
	router, mockController := setupRouter()
	amt := uint(0)
	claim := task.Claim{
		Queues: []queue.Name{"hello"},
		Amount: &amt,
	}
	resp := performRequest(router, http.MethodPost, "/tasques/claims", claim, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.claimCalled)
	var respTasks []task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTasks); err != nil {
		t.Error(err, resp)
	} else {
		assert.Len(t, respTasks, 0)
	}
}

func TestTaskClaim_InvalidQueue(t *testing.T) {
	router, mockController := setupRouter()
	amt := uint(1)
	claim := task.Claim{
		Queues: []queue.Name{"+hello"},
		Amount: &amt,
	}
	resp := performRequest(router, http.MethodPost, "/tasques/claims", claim, workerHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.claimCalled)
}

func TestTaskUnClaim_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodDelete, "/tasques/claims/q/1", nil, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.unClaimCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err, resp)
	} else {
		assert.EqualValues(t, "q", respTask.Queue)
		assert.EqualValues(t, "1", respTask.ID)
	}
}

func TestTaskUnClaim_NoWorkerId(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodDelete, "/tasques/claims/q/1", nil, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.unClaimCalled)
}

func TestTaskUnClaim_InvalidQueue(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodDelete, "/tasques/claims/+/1", nil, workerHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.unClaimCalled)
}

func TestTaskUnClaim_NotOwned(t *testing.T) {
	router, mockController := setupRouter()
	err := common.ApiError{
		StatusCode: http.StatusForbidden,
	}
	mockController.unClaimOverride = func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id) (t *task.Task, apiError *common.ApiError) {
		return nil, &err
	}
	resp := performRequest(router, http.MethodDelete, "/tasques/claims/q/1", nil, workerHeaders())
	assert.EqualValues(t, 1, mockController.unClaimCalled)
	assert.EqualValues(t, err.StatusCode, resp.Code)
}

func TestTaskReportIn_Ok(t *testing.T) {
	router, mockController := setupRouter()
	report := task.NewReport{}
	resp := performRequest(router, http.MethodPut, "/tasques/reports/q/1", report, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.reportCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err, resp)
	} else {
		assert.EqualValues(t, "q", respTask.Queue)
		assert.EqualValues(t, "1", respTask.ID)
	}
}

func TestTaskReportIn_NoWorkerId(t *testing.T) {
	router, mockController := setupRouter()
	report := task.NewReport{}
	resp := performRequest(router, http.MethodPut, "/tasques/reports/q/1", report, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.reportCalled)
}

func TestTaskReportIn_InvalidQueue(t *testing.T) {
	router, mockController := setupRouter()
	report := task.NewReport{}
	resp := performRequest(router, http.MethodPut, "/tasques/reports/+/1", report, workerHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.reportCalled)
}

func TestTaskReportIn_NotOwned(t *testing.T) {
	router, mockController := setupRouter()
	report := task.NewReport{}
	err := common.ApiError{
		StatusCode: http.StatusForbidden,
	}
	mockController.reportInOverride = func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, report task.NewReport) (t *task.Task, apiError *common.ApiError) {
		return nil, &err
	}
	resp := performRequest(router, http.MethodPut, "/tasques/reports/q/1", report, workerHeaders())
	assert.EqualValues(t, 1, mockController.reportCalled)
	assert.EqualValues(t, err.StatusCode, resp.Code)
}

func TestTaskMarkDone_Ok(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/done/q/1", body, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.markDoneCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err, resp)
	} else {
		assert.EqualValues(t, "q", respTask.Queue)
		assert.EqualValues(t, "1", respTask.ID)
	}
}

func TestTaskMarkDone_NoWorkerId(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/done/q/1", body, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.markDoneCalled)
}

func TestTaskMarkDone_InvalidQueue(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/done/+/1", body, workerHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.markDoneCalled)
}

func TestTaskMarkDone_NotOwned(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	err := common.ApiError{
		StatusCode: http.StatusForbidden,
	}
	mockController.markDoneOverride = func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Success) (t *task.Task, apiError *common.ApiError) {
		return nil, &err
	}
	resp := performRequest(router, http.MethodPut, "/tasques/done/q/1", body, workerHeaders())
	assert.EqualValues(t, 1, mockController.markDoneCalled)
	assert.EqualValues(t, err.StatusCode, resp.Code)
}

func TestTaskMarkFailed_Ok(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/failed/q/1", body, workerHeaders())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.markFailedCalled)
	var respTask task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err, resp)
	} else {
		assert.EqualValues(t, "q", respTask.Queue)
		assert.EqualValues(t, "1", respTask.ID)
	}
}

func TestTaskMarkFailed_NoWorkerId(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/failed/q/1", body, nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.markFailedCalled)
}

func TestTaskMarkFailed_InvalidQueue(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	resp := performRequest(router, http.MethodPut, "/tasques/failed/+/1", body, workerHeaders())
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.EqualValues(t, 0, mockController.markFailedCalled)
}

func TestTaskMarkFailed_NotOwned(t *testing.T) {
	router, mockController := setupRouter()
	body := task.Success{}
	err := common.ApiError{
		StatusCode: http.StatusForbidden,
	}
	mockController.markFailedOverride = func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, failure *domainTask.Failure) (t *task.Task, apiError *common.ApiError) {
		return nil, &err
	}
	resp := performRequest(router, http.MethodPut, "/tasques/failed/q/1", body, workerHeaders())
	assert.EqualValues(t, 1, mockController.markFailedCalled)
	assert.EqualValues(t, err.StatusCode, resp.Code)
}

// Mocks

func now() time.Time { return time.Now().UTC() }
func mockDomainTask() domainTask.Task {
	now := now()
	return domainTask.Task{
		ID:             "id",
		Queue:          "q",
		RetryTimes:     10,
		Attempted:      0,
		Kind:           "sayHi",
		State:          domainTask.QUEUED,
		Priority:       0,
		LastEnqueuedAt: domainTask.EnqueuedAt(now),
		Metadata: metadata.Metadata{
			CreatedAt:  metadata.CreatedAt(now),
			ModifiedAt: metadata.ModifiedAt(now),
			Version: metadata.Version{
				SeqNum:      1,
				PrimaryTerm: 2,
			},
		},
	}
}

func mockApiTask() task.Task {
	dTask := mockDomainTask()
	return task.FromDomainTask(&dTask)
}

type mockTasksController struct {
	createCalled       uint
	createOverride     func(ctx context.Context, newTask *task.NewTask) (*task.Task, *common.ApiError)
	getCalled          uint
	getOverride        func(ctx context.Context, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError)
	claimCalled        uint
	claimOverride      func(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]task.Task, *common.ApiError)
	reportCalled       uint
	reportInOverride   func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, report task.NewReport) (*task.Task, *common.ApiError)
	markDoneCalled     uint
	markDoneOverride   func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Success) (*task.Task, *common.ApiError)
	markFailedCalled   uint
	markFailedOverride func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, failure *domainTask.Failure) (*task.Task, *common.ApiError)
	unClaimCalled      uint
	unClaimOverride    func(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError)
}

func (m *mockTasksController) Create(ctx context.Context, newTask *task.NewTask) (*task.Task, *common.ApiError) {
	m.createCalled++
	if m.createOverride != nil {
		return m.createOverride(ctx, newTask)
	} else {
		apiTask := mockApiTask()
		apiTask.Queue = newTask.Queue
		apiTask.Kind = newTask.Kind
		if newTask.RunAt != nil {
			apiTask.RunAt = *newTask.RunAt
		} else {
			apiTask.RunAt = now()
		}
		apiTask.State = domainTask.QUEUED
		apiTask.Args = newTask.Args
		apiTask.Context = newTask.Context
		return &apiTask, nil
	}
}

func (m *mockTasksController) Get(ctx context.Context, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError) {
	m.getCalled++
	if m.getOverride != nil {
		return m.getOverride(ctx, queue, taskId)
	} else {
		apiTask := mockApiTask()
		apiTask.Queue = queue
		apiTask.ID = taskId
		return &apiTask, nil
	}
}

func (m *mockTasksController) Claim(ctx context.Context, workerId worker.Id, queues []queue.Name, number uint, blockFor time.Duration) ([]task.Task, *common.ApiError) {
	m.claimCalled++
	if m.claimOverride != nil {
		return m.claimOverride(ctx, workerId, queues, number, blockFor)
	} else {
		var tasks []task.Task
		if len(queues) > 0 {
			for i := uint(0); i < number; i++ {
				t := mockApiTask()
				t.Queue = queues[0]
				tasks = append(tasks, t)
			}
		}
		return tasks, nil
	}
}

func (m *mockTasksController) ReportIn(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, report task.NewReport) (*task.Task, *common.ApiError) {
	m.reportCalled++
	if m.reportInOverride != nil {
		return m.reportInOverride(ctx, workerId, queue, taskId, report)
	} else {
		now := now()
		apiTask := mockApiTask()
		apiTask.Queue = queue
		apiTask.ID = taskId
		apiTask.LastClaimed = &task.LastClaimed{
			WorkerId:  workerId,
			ClaimedAt: now,
			LastReport: &task.Report{
				At:   now,
				Data: report.Data,
			},
			Result: nil,
		}
		return &apiTask, nil
	}
}

func (m *mockTasksController) MarkDone(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, success *domainTask.Success) (*task.Task, *common.ApiError) {
	m.markDoneCalled++
	if m.markDoneOverride != nil {
		return m.markDoneOverride(ctx, workerId, queue, taskId, success)
	} else {
		now := now()
		apiTask := mockApiTask()
		apiTask.Queue = queue
		apiTask.ID = taskId
		apiTask.LastClaimed = &task.LastClaimed{
			WorkerId:  workerId,
			ClaimedAt: now,
			Result: &task.Result{
				At:      now,
				Failure: nil,
				Success: success,
			},
		}
		return &apiTask, nil
	}
}

func (m *mockTasksController) MarkFailed(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id, failed *domainTask.Failure) (*task.Task, *common.ApiError) {
	m.markFailedCalled++
	if m.markFailedOverride != nil {
		return m.markFailedOverride(ctx, workerId, queue, taskId, failed)
	} else {
		now := now()
		apiTask := mockApiTask()
		apiTask.Queue = queue
		apiTask.ID = taskId
		apiTask.LastClaimed = &task.LastClaimed{
			WorkerId:  workerId,
			ClaimedAt: now,
			Result: &task.Result{
				At:      now,
				Failure: failed,
				Success: nil,
			},
		}
		return &apiTask, nil
	}
}

func (m *mockTasksController) UnClaim(ctx context.Context, workerId worker.Id, queue queue.Name, taskId domainTask.Id) (*task.Task, *common.ApiError) {
	m.unClaimCalled++
	if m.unClaimOverride != nil {
		return m.unClaimOverride(ctx, workerId, queue, taskId)
	} else {
		apiTask := mockApiTask()
		apiTask.Queue = queue
		apiTask.ID = taskId
		apiTask.State = domainTask.QUEUED
		return &apiTask, nil
	}
}
