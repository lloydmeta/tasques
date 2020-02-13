package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lloydmeta/tasques/internal/api/models/common"
)

var notFoundErr = common.ApiError{
	StatusCode: http.StatusNotFound,
	Body: common.Body{
		Message: "No such route.",
	},
}

var noMethodErr = common.ApiError{
	StatusCode: http.StatusMethodNotAllowed,
	Body: common.Body{
		Message: "No such route.",
	},
}

func NoRoute(c *gin.Context) {
	c.JSON(notFoundErr.StatusCode, notFoundErr.Body)
}

func NoMethod(c *gin.Context) {
	c.JSON(notFoundErr.StatusCode, noMethodErr.Body)
}
