package recurring

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

	"github.com/lloydmeta/tasques/internal/api/models/task/recurring"
	"github.com/lloydmeta/tasques/internal/domain/metadata"

	domainRecurring "github.com/lloydmeta/tasques/internal/domain/task/recurring"
	"github.com/lloydmeta/tasques/internal/infra/server/binding/validation"
	"github.com/lloydmeta/tasques/internal/infra/server/routing"
)

func init() {
	validation.SetUpValidators(validation.TestStandardParser{})
}

func Test_Create_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodPost, "/recurring_tasques", mockApiNewTask, nil)
	assert.EqualValues(t, http.StatusCreated, resp.Code)
	assert.EqualValues(t, 1, mockController.createCalled)
	var respTask recurring.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, mockApiRecurringTask, respTask)
	}
}

func Test_Create_Err(t *testing.T) {
	router, mockController := setupRouter()
	apiErr := common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: "nope",
		},
	}
	mockController.createOverride = func() (task *recurring.Task, apiError *common.ApiError) {
		return nil, &apiErr
	}
	resp := performRequest(router, http.MethodPost, "/recurring_tasques", mockApiNewTask, nil)
	assert.EqualValues(t, apiErr.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.createCalled)
	var respTask common.Body
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, apiErr.Body, respTask)
	}
}

func Test_Create_InvalidSchedule(t *testing.T) {
	router, _ := setupRouter()
	invalidScheduleApiNewTask := recurring.NewTask{
		ID:                 "abc",
		ScheduleExpression: "lolz",
		TaskDefinition: recurring.TaskDefinition{
			Queue: "q",
			Kind:  "k",
		},
	}
	resp := performRequest(router, http.MethodPost, "/recurring_tasques", invalidScheduleApiNewTask, nil)
	assert.EqualValues(t, http.StatusBadRequest, resp.Code)
}

func Test_Update_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodPut, "/recurring_tasques/hello", mockApiTaskUpdate, nil)
	assert.EqualValues(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.updateCalled)
	var respTask recurring.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, mockApiRecurringTask, respTask)
	}
}

func Test_Update_Err(t *testing.T) {
	router, mockController := setupRouter()
	apiErr := common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: "nope",
		},
	}
	mockController.updateOverride = func() (task *recurring.Task, apiError *common.ApiError) {
		return nil, &apiErr
	}
	resp := performRequest(router, http.MethodPut, "/recurring_tasques/hello", mockApiTaskUpdate, nil)
	assert.EqualValues(t, apiErr.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.updateCalled)
	var respTask common.Body
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, apiErr.Body, respTask)
	}
}

func Test_Update_InvalidSchedule(t *testing.T) {
	router, _ := setupRouter()
	invalidSchedule := domainRecurring.ScheduleExpression("lol")
	invalidScheduleUpdate := recurring.TaskUpdate{
		ScheduleExpression: &invalidSchedule,
		TaskDefinition: &recurring.TaskDefinition{
			Queue: "q",
			Kind:  "k",
		},
	}
	resp := performRequest(router, http.MethodPut, "/recurring_tasques/hello", invalidScheduleUpdate, nil)
	assert.EqualValues(t, http.StatusBadRequest, resp.Code)
}

func Test_Update_NoSchedule(t *testing.T) {
	router, _ := setupRouter()
	noScheduleUpdate := recurring.TaskUpdate{
		TaskDefinition: &recurring.TaskDefinition{
			Queue: "q",
			Kind:  "k",
		},
	}
	resp := performRequest(router, http.MethodPut, "/recurring_tasques/hello", noScheduleUpdate, nil)
	assert.EqualValues(t, http.StatusOK, resp.Code)
}

func Test_Get_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodGet, "/recurring_tasques/hello", nil, nil)
	assert.EqualValues(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.getCalled)
	var respTask recurring.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, mockApiRecurringTask, respTask)
	}
}

func Test_Get_Err(t *testing.T) {
	router, mockController := setupRouter()
	apiErr := common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: "nope",
		},
	}
	mockController.getOverride = func() (task *recurring.Task, apiError *common.ApiError) {
		return nil, &apiErr
	}
	resp := performRequest(router, http.MethodGet, "/recurring_tasques/hello", nil, nil)
	assert.EqualValues(t, apiErr.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.getCalled)
	var respTask common.Body
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, apiErr.Body, respTask)
	}
}

func Test_Delete_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodDelete, "/recurring_tasques/hello", nil, nil)
	assert.EqualValues(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.deleteCalled)
	var respTask recurring.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, mockApiRecurringTask, respTask)
	}
}

func Test_Delete_Err(t *testing.T) {
	router, mockController := setupRouter()
	apiErr := common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: "nope",
		},
	}
	mockController.deleteOverride = func() (task *recurring.Task, apiError *common.ApiError) {
		return nil, &apiErr
	}
	resp := performRequest(router, http.MethodDelete, "/recurring_tasques/hello", nil, nil)
	assert.EqualValues(t, apiErr.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.deleteCalled)
	var respTask common.Body
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, apiErr.Body, respTask)
	}
}

func Test_List_Ok(t *testing.T) {
	router, mockController := setupRouter()
	resp := performRequest(router, http.MethodGet, "/recurring_tasques", nil, nil)
	assert.EqualValues(t, http.StatusOK, resp.Code)
	assert.EqualValues(t, 1, mockController.listCalled)
	var respTask []recurring.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, []recurring.Task{mockApiRecurringTask}, respTask)
	}
}

func Test_List_Err(t *testing.T) {
	router, mockController := setupRouter()
	apiErr := common.ApiError{
		StatusCode: http.StatusConflict,
		Body: common.Body{
			Message: "nope",
		},
	}
	mockController.listOverride = func() (task []recurring.Task, apiError *common.ApiError) {
		return nil, &apiErr
	}
	resp := performRequest(router, http.MethodGet, "/recurring_tasques", nil, nil)
	assert.EqualValues(t, apiErr.StatusCode, resp.Code)
	assert.EqualValues(t, 1, mockController.listCalled)
	var respTask common.Body
	if err := json.Unmarshal(resp.Body.Bytes(), &respTask); err != nil {
		t.Error(err)
	} else {
		assert.EqualValues(t, apiErr.Body, respTask)
	}
}

func setupRouter() (*gin.Engine, *mockRecurringTasksController) {
	engine := gin.Default()
	mockController := mockRecurringTasksController{}
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

var now = time.Now().UTC()

var mockApiNewTask = recurring.NewTask{
	ID:                 "abc",
	ScheduleExpression: "* * * * * ",
	TaskDefinition: recurring.TaskDefinition{
		Queue:             "q",
		RetryTimes:        nil,
		Kind:              "k",
		Priority:          nil,
		ProcessingTimeout: nil,
		Args:              nil,
		Context:           nil,
	},
}

var mockApiTaskUpdate = recurring.TaskUpdate{

	ScheduleExpression: &mockApiNewTask.ScheduleExpression,
	TaskDefinition: &recurring.TaskDefinition{
		Queue:             "q",
		RetryTimes:        nil,
		Kind:              "k",
		Priority:          nil,
		ProcessingTimeout: nil,
		Args:              nil,
		Context:           nil,
	},
}

var mockDomainRecurringTask = domainRecurring.Task{
	ID:                 "t",
	ScheduleExpression: "* * * * *",
	TaskDefinition:     domainRecurring.TaskDefinition{},
	IsDeleted:          false,
	LoadedAt:           nil,
	Metadata: metadata.Metadata{
		CreatedAt:  metadata.CreatedAt(now),
		ModifiedAt: metadata.ModifiedAt(now),
		Version: metadata.Version{
			SeqNum:      0,
			PrimaryTerm: 1,
		},
	},
}

var mockApiRecurringTask = recurring.FromDomainTask(&mockDomainRecurringTask)

type mockRecurringTasksController struct {
	createCalled   uint
	createOverride func() (*recurring.Task, *common.ApiError)
	updateCalled   uint
	updateOverride func() (*recurring.Task, *common.ApiError)
	getCalled      uint
	getOverride    func() (*recurring.Task, *common.ApiError)
	deleteCalled   uint
	deleteOverride func() (*recurring.Task, *common.ApiError)
	listCalled     uint
	listOverride   func() ([]recurring.Task, *common.ApiError)
}

func (m *mockRecurringTasksController) Create(ctx context.Context, task *recurring.NewTask) (*recurring.Task, *common.ApiError) {
	m.createCalled++
	if m.createOverride != nil {
		return m.createOverride()
	} else {
		return &mockApiRecurringTask, nil
	}
}

func (m *mockRecurringTasksController) Update(ctx context.Context, id domainRecurring.Id, task *recurring.TaskUpdate) (*recurring.Task, *common.ApiError) {
	m.updateCalled++
	if m.updateOverride != nil {
		return m.updateOverride()
	} else {
		return &mockApiRecurringTask, nil
	}
}

func (m *mockRecurringTasksController) Get(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError) {
	m.getCalled++
	if m.getOverride != nil {
		return m.getOverride()
	} else {
		return &mockApiRecurringTask, nil
	}
}

func (m *mockRecurringTasksController) Delete(ctx context.Context, id domainRecurring.Id) (*recurring.Task, *common.ApiError) {
	m.deleteCalled++
	if m.deleteOverride != nil {
		return m.deleteOverride()
	} else {
		return &mockApiRecurringTask, nil
	}
}

func (m *mockRecurringTasksController) List(ctx context.Context) ([]recurring.Task, *common.ApiError) {
	m.listCalled++
	if m.listOverride != nil {
		return m.listOverride()
	} else {
		return []recurring.Task{mockApiRecurringTask}, nil
	}
}
