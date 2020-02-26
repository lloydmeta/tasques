package recurring

import (
	"net/http"

	"github.com/gin-gonic/gin"

	recurringController "github.com/lloydmeta/tasques/internal/api/controllers/task/recurring"
	"github.com/lloydmeta/tasques/internal/api/models/task/recurring"
	domainRecurring "github.com/lloydmeta/tasques/internal/domain/task/recurring"
	"github.com/lloydmeta/tasques/internal/infra/server/routing"
)

var subPath = "recurring_tasques"

var recurringTaskKey = "recurring_task_id"

type RoutesHandler struct {
	Controller recurringController.Controller
}

func (h *RoutesHandler) RegisterRoutes(routerGroup *gin.RouterGroup) {
	subGroup := routerGroup.Group(subPath)
	subGroup.POST("", h.create)
	subGroup.PUT("/:"+recurringTaskKey, h.update)
	subGroup.GET("/:"+recurringTaskKey, h.get)
	subGroup.DELETE("/:"+recurringTaskKey, h.delete)
	subGroup.GET("", h.list)
}

// @Summary Add a new Recurring Task
// @ID create-recurring-task
// @Tags recurring-tasks
// @Description Creates a new Recurring Task
// @Accept  json
// @Produce  json
// @Param   newRecurringTask body recurring.NewTask true "The request body"
// @Success 201 {object} recurring.Task
// @Failure 400 {object} common.Body "Invalid JSON"
// @Failure 409 {object} common.Body "Id in use"
// @Router /recurring_tasques [post]
func (h *RoutesHandler) create(c *gin.Context) {
	var newTask recurring.NewTask
	if err := c.ShouldBindJSON(&newTask); err != nil {
		routing.HandleJsonSerdesErr(c, err)
	} else {
		if t, err := recurringController.Controller.Create(h.Controller, c.Request.Context(), &newTask); err == nil {
			c.JSON(http.StatusCreated, t)
		} else {
			c.JSON(err.StatusCode, err.Body)
		}
	}
}

// @Summary Update a Recurring Task
// @ID update-existing-recurring-task
// @Tags recurring-tasks
// @Description Updates a persisted Recurring Task
// @Accept  json
// @Produce  json
// @Param   id path string true "The id of the Recurring Task"
// @Param   recurringTaskUpdate body recurring.TaskUpdate true "The request body"
// @Success 200 {object} recurring.Task
// @Failure 404 {object} common.Body "Recurring Task does not exist"
// @Router /recurring_tasques/{id} [put]
func (h *RoutesHandler) update(c *gin.Context) {
	var id = domainRecurring.Id(c.Param(recurringTaskKey))
	var taskUpdate recurring.TaskUpdate
	if err := c.ShouldBindJSON(&taskUpdate); err != nil {
		routing.HandleJsonSerdesErr(c, err)
	} else {
		if t, err := h.Controller.Update(c.Request.Context(), id, &taskUpdate); err == nil {
			c.JSON(http.StatusOK, t)
		} else {
			c.JSON(err.StatusCode, err.Body)
		}
	}
}

// @Summary Get a Recurring Task
// @ID get-existing-recurring-task
// @Tags recurring-tasks
// @Description Retrieves a persisted Recurring Task
// @Accept  json
// @Produce  json
// @Param   id path string true "The id of the Recurring Task"
// @Success 200 {object} recurring.Task
// @Failure 404 {object} common.Body "Recurring Task does not exist"
// @Router /recurring_tasques/{id} [get]
func (h *RoutesHandler) get(c *gin.Context) {
	var id = domainRecurring.Id(c.Param(recurringTaskKey))
	if t, err := h.Controller.Get(c.Request.Context(), id); err == nil {
		c.JSON(http.StatusOK, t)
	} else {
		c.JSON(err.StatusCode, err.Body)
	}
}

// @Summary Delete a Recurring Task
// @ID delete-existing-recurring-task
// @Tags recurring-tasks
// @Description Deletes a persisted Recurring Task
// @Accept  json
// @Produce  json
// @Param   id path string true "The id of the Recurring Task"
// @Success 200 {object} recurring.Task
// @Failure 404 {object} common.Body "Recurring Task does not exist"
// @Router /recurring_tasques/{id} [delete]
func (h *RoutesHandler) delete(c *gin.Context) {
	var id = domainRecurring.Id(c.Param(recurringTaskKey))
	if t, err := h.Controller.Delete(c.Request.Context(), id); err == nil {
		c.JSON(http.StatusOK, t)
	} else {
		c.JSON(err.StatusCode, err.Body)
	}
}

// @Summary List Recurring Tasks
// @ID list-existing-recurring-tasks
// @Tags recurring-tasks
// @Description Lists persisted Recurring Tasks
// @Accept  json
// @Produce  json
// @Success 200 {array} recurring.Task
// @Router /recurring_tasques [get]
func (h *RoutesHandler) list(c *gin.Context) {
	if t, err := h.Controller.List(c.Request.Context()); err == nil {
		c.JSON(http.StatusOK, t)
	} else {
		c.JSON(err.StatusCode, err.Body)
	}
}
