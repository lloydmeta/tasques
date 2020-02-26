package routing

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	taskController "github.com/lloydmeta/tasques/internal/api/controllers/task"
	"github.com/lloydmeta/tasques/internal/config"
	"github.com/lloydmeta/tasques/internal/domain/queue"
	domainTask "github.com/lloydmeta/tasques/internal/domain/task"

	"github.com/gin-gonic/gin"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/api/models/task"
	"github.com/lloydmeta/tasques/internal/domain/worker"
)

var rootPath = "/tasques"
var WorkerIdHeaderKey = "X-TASQUES-WORKER-ID"
var taskIdPathKey = "task_id"
var queuePathKey = "queue"

type TasksRoutesHandler struct {
	TasksDefaultsSettings config.TasksDefaults
	AuthSettings          *config.Auth
	Controller            taskController.Controller
}

func (h *TasksRoutesHandler) RegisterRoutes(ginEngine *gin.Engine) {
	accounts := make(gin.Accounts)
	if h.AuthSettings != nil {
		for _, bAuthUser := range h.AuthSettings.BasicAuth {
			accounts[bAuthUser.Name] = bAuthUser.Password
		}
	}

	var routerGroup *gin.RouterGroup
	if len(accounts) > 0 {
		routerGroup = ginEngine.Group(rootPath, gin.BasicAuth(accounts))
	} else {
		routerGroup = ginEngine.Group(rootPath)
	}

	routerGroup.POST("", h.create)
	routerGroup.GET("/:"+queuePathKey+"/:"+taskIdPathKey, h.get)
	routerGroup.POST("/claims", h.claim)
	routerGroup.DELETE("/claims/:"+queuePathKey+"/:"+taskIdPathKey, h.unClaim)
	routerGroup.PUT("/reports/:"+queuePathKey+"/:"+taskIdPathKey, h.reportIn)
	routerGroup.PUT("/done/:"+queuePathKey+"/:"+taskIdPathKey, h.markDone)
	routerGroup.PUT("/failed/:"+queuePathKey+"/:"+taskIdPathKey, h.markFailed)
}

// @Summary Add a new Task
// @ID create-task
// @Tags tasks
// @Description Creates a new Task
// @Accept  json
// @Produce  json
// @Param   newTask body task.NewTask true "The request body"
// @Success 201 {object} task.Task
// @Failure 400 {object} common.Body "Invalid JSON"
// @Router /tasques [post]
func (h *TasksRoutesHandler) create(c *gin.Context) {
	var newTask task.NewTask
	if err := c.ShouldBindJSON(&newTask); err != nil {
		handleJsonSerdesErr(c, err)
	} else {
		if t, err := h.Controller.Create(c.Request.Context(), &newTask); err == nil {
			c.JSON(http.StatusCreated, t)
		} else {
			c.JSON(err.StatusCode, err.Body)
		}
	}
}

// @Summary Get a Task
// @ID get-existing-task
// @Tags tasks
// @Description Retrieves a persisted Task
// @Accept  json
// @Produce  json
// @Param   queue path string true "The Queue of the Task"
// @Param   id path string true "The id of the Task"
// @Success 200 {object} task.Task
// @Failure 404 {object} common.Body "Task does not exist"
// @Router /tasques/{queue}/{id} [get]
func (h *TasksRoutesHandler) get(c *gin.Context) {
	var taskId = domainTask.Id(c.Param(taskIdPathKey))
	var queueStr = c.Param(queuePathKey)
	queueName, err := queue.NameFromString(queueStr)
	if err != nil {
		badQueueNames(c, []error{err})
	} else {
		if t, err := h.Controller.Get(c.Request.Context(), *queueName, taskId); err == nil {
			c.JSON(http.StatusOK, t)
		} else {
			c.JSON(err.StatusCode, err.Body)
		}
	}
}

// @Summary Claims a number of Tasks
// @ID claim-tasks
// @Tags tasks
// @Description Claims a number of existing Tasks.
// @Accept  json
// @Produce  json
// @Param X-TASQUES-WORKER-ID header string true "Worker ID"
// @Param   claim body task.Claim true "The request body"
// @Success 200 {array} task.Task
// @Router /tasques/claims [post]
func (h *TasksRoutesHandler) claim(c *gin.Context) {
	if workerId, err := getWorkerIdOrErr(c); err != nil {
		handleApiErr(c, err)
	} else {
		var claim task.Claim
		if err := c.ShouldBindJSON(&claim); err != nil {
			handleJsonSerdesErr(c, err)
		} else {
			var blockFor time.Duration
			if claim.BlockFor == nil {
				blockFor = h.TasksDefaultsSettings.BlockFor
			} else {
				blockFor = time.Duration(*claim.BlockFor)
			}
			var amount uint
			if claim.Amount == nil {
				amount = h.TasksDefaultsSettings.ClaimAmount
			} else {
				amount = *claim.Amount
			}
			if claimed, err := h.Controller.Claim(c.Request.Context(), *workerId, claim.Queues, amount, blockFor); err != nil {
				c.JSON(err.StatusCode, err.Body)
			} else {
				c.JSON(http.StatusOK, claimed)
			}
		}
	}
}

// @Summary Unclaims a Task
// @ID unclaim-existing-task
// @Tags tasks
// @Description Unclaims a claimed Task.
// @Accept  json
// @Produce  json
// @Param   queue path string true "The Queue of the Task"
// @Param   id path string true "The id of the Task"
// @Param X-TASQUES-WORKER-ID header string true "Worker ID"
// @Success 200 {object} task.Task
// @Failure 400 {object} common.Body "The Task is not currently claimed"
// @Failure 403 {object} common.Body "Worker currently has not claimed the Task"
// @Failure 404 {object} common.Body "Task does not exist"
// @Router /tasques/claims/{queue}/{id} [delete]
func (h *TasksRoutesHandler) unClaim(c *gin.Context) {
	var taskId = domainTask.Id(c.Param(taskIdPathKey))
	var queueStr = c.Param(queuePathKey)
	queueName, err := queue.NameFromString(queueStr)
	if err != nil {
		badQueueNames(c, []error{err})
	} else {
		if workerId, err := getWorkerIdOrErr(c); err != nil {
			handleApiErr(c, err)
		} else {
			if t, err := h.Controller.UnClaim(c.Request.Context(), *workerId, *queueName, taskId); err == nil {
				c.JSON(http.StatusOK, t)
			} else {
				c.JSON(err.StatusCode, err.Body)
			}
		}
	}
}

// @Summary Reports on a Task
// @ID report-on-claimed-task
// @Tags tasks
// @Description Reports in on a claimed Task.
// @Accept  json
// @Produce  json
// @Param   newReport body task.NewReport true "The request body"
// @Param   queue path string true "The Queue of the Task"
// @Param   id path string true "The id of the Task"
// @Param X-TASQUES-WORKER-ID header string true "Worker ID"
// @Success 200 {object} task.Task
// @Failure 400 {object} common.Body "The Task is not currently claimed"
// @Failure 403 {object} common.Body "Worker currently has not claimed the Task"
// @Failure 404 {object} common.Body "Task does not exist"
// @Router /tasques/reports/{queue}/{id} [put]
func (h *TasksRoutesHandler) reportIn(c *gin.Context) {
	var taskId = domainTask.Id(c.Param(taskIdPathKey))
	var queueStr = c.Param(queuePathKey)
	queueName, err := queue.NameFromString(queueStr)
	if err != nil {
		badQueueNames(c, []error{err})
	} else {
		var newReport task.NewReport
		if err := c.ShouldBindJSON(&newReport); err != nil {
			handleJsonSerdesErr(c, err)
		} else {
			if workerId, err := getWorkerIdOrErr(c); err != nil {
				handleApiErr(c, err)
			} else {
				if t, err := h.Controller.ReportIn(c.Request.Context(), *workerId, *queueName, taskId, newReport); err == nil {
					c.JSON(http.StatusOK, t)
				} else {
					c.JSON(err.StatusCode, err.Body)
				}
			}
		}
	}
}

// @Summary Mark Task as Done
// @ID mark-claimed-task-done
// @Tags tasks
// @Description Marks a claimed Task as done.
// @Accept  json
// @Produce  json
// @Param   success body task.Success true "The request body"
// @Param   queue path string true "The Queue of the Task"
// @Param   id path string true "The id of the Task"
// @Param X-TASQUES-WORKER-ID header string true "Worker ID"
// @Success 200 {object} task.Task
// @Failure 400 {object} common.Body "The Task is not currently claimed"
// @Failure 403 {object} common.Body "Worker currently has not claimed the Task"
// @Failure 404 {object} common.Body "Task does not exist"
// @Router /tasques/done/{queue}/{id} [put]
func (h *TasksRoutesHandler) markDone(c *gin.Context) {
	var taskId = domainTask.Id(c.Param(taskIdPathKey))
	var queueStr = c.Param(queuePathKey)
	queueName, err := queue.NameFromString(queueStr)
	if err != nil {
		badQueueNames(c, []error{err})
	} else {
		var newResult task.Success
		if err := c.ShouldBindJSON(&newResult); err != nil {
			handleJsonSerdesErr(c, err)
		} else {
			if workerId, err := getWorkerIdOrErr(c); err != nil {
				handleApiErr(c, err)
			} else {
				if t, err := h.Controller.MarkDone(c.Request.Context(), *workerId, *queueName, taskId, newResult.Data); err == nil {
					c.JSON(http.StatusOK, t)
				} else {
					c.JSON(err.StatusCode, err.Body)
				}
			}
		}
	}
}

// @Summary Mark Task as Failed
// @ID mark-claimed-task-failed
// @Tags tasks
// @Description Marks a claimed Task as failed.
// @Accept  json
// @Produce  json
// @Param   failure body task.Failure true "The request body"
// @Param   queue path string true "The Queue of the Task"
// @Param   id path string true "The id of the Task"
// @Param X-TASQUES-WORKER-ID header string true "Worker ID"
// @Success 200 {object} task.Task
// @Failure 400 {object} common.Body "The Task is not currently claimed"
// @Failure 403 {object} common.Body "Worker currently has not claimed the Task"
// @Failure 404 {object} common.Body "Task does not exist"
// @Router /tasques/failed/{queue}/{id} [put]
func (h *TasksRoutesHandler) markFailed(c *gin.Context) {
	var taskId = domainTask.Id(c.Param(taskIdPathKey))
	var queueStr = c.Param(queuePathKey)
	queueName, err := queue.NameFromString(queueStr)
	if err != nil {
		badQueueNames(c, []error{err})
	} else {
		var newResult task.Failure
		if err := c.ShouldBindJSON(&newResult); err != nil {
			handleJsonSerdesErr(c, err)
		} else {
			if workerId, err := getWorkerIdOrErr(c); err != nil {
				handleApiErr(c, err)
			} else {
				if t, err := h.Controller.MarkFailed(c.Request.Context(), *workerId, *queueName, taskId, newResult.Data); err == nil {
					c.JSON(http.StatusOK, t)
				} else {
					c.JSON(err.StatusCode, err.Body)
				}
			}
		}
	}
}

var noWorkerIdApiErr = common.ApiError{
	StatusCode: http.StatusBadRequest,
	Body: common.Body{
		Message: fmt.Sprintf("Worker Id header [%s] not sent", WorkerIdHeaderKey),
	},
}

func getWorkerIdOrErr(c *gin.Context) (*worker.Id, *common.ApiError) {
	log.Info().Msgf("%v", c.Request.Header)
	workerIdStr := strings.TrimSpace(c.Request.Header.Get(WorkerIdHeaderKey))
	if len(workerIdStr) == 0 {
		return nil, &noWorkerIdApiErr
	} else {
		workerId := worker.Id(workerIdStr)
		return &workerId, nil
	}
}

func handleApiErr(c *gin.Context, apiError *common.ApiError) {
	c.JSON(apiError.StatusCode, apiError.Body)
}

func handleJsonSerdesErr(c *gin.Context, err error) {
	errResp := common.ApiError{
		StatusCode: http.StatusBadRequest,
		Body: common.Body{
			Message: err.Error(),
		},
	}
	handleApiErr(c, &errResp)
}

func badQueueNames(c *gin.Context, queueNameErrors []error) {
	errorMsgs := make([]string, 0, len(queueNameErrors))
	for _, err := range queueNameErrors {
		errorMsgs = append(errorMsgs, err.Error())
	}
	errResp := common.ApiError{
		StatusCode: http.StatusBadRequest,
		Body: common.Body{
			Message: strings.Join(errorMsgs, ", "),
		},
	}
	handleApiErr(c, &errResp)
}
