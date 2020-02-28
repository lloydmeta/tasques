package routing

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lloydmeta/tasques/internal/api/models/common"
	"github.com/lloydmeta/tasques/internal/config"
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

func NewTopLevelRoutesGroup(auth *config.Auth, ginEngine *gin.Engine) *gin.RouterGroup {

	accounts := make(gin.Accounts)
	if auth != nil {
		for _, bAuthUser := range auth.BasicAuth {
			accounts[bAuthUser.Name] = bAuthUser.Password
		}
	}

	var routerGroup *gin.RouterGroup
	if len(accounts) > 0 {
		routerGroup = ginEngine.Group("", gin.BasicAuth(accounts))
	} else {
		routerGroup = ginEngine.Group("")
	}

	return routerGroup
}

func NoRoute(c *gin.Context) {
	c.JSON(notFoundErr.StatusCode, notFoundErr.Body)
}

func NoMethod(c *gin.Context) {
	c.JSON(notFoundErr.StatusCode, noMethodErr.Body)
}

func HandleApiErr(c *gin.Context, apiError *common.ApiError) {
	c.JSON(apiError.StatusCode, apiError.Body)
}

func HandleJsonSerdesErr(c *gin.Context, err error) {
	errResp := common.ApiError{
		StatusCode: http.StatusBadRequest,
		Body: common.Body{
			Message: err.Error(),
		},
	}
	HandleApiErr(c, &errResp)
}
