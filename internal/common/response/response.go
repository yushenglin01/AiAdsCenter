package response

import (
	"errors"
	"net/http"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data, RequestID: requestID(c)})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: 0, Message: "ok", Data: data, RequestID: requestID(c)})
}

func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, Envelope{Code: 0, Message: "accepted", Data: data, RequestID: requestID(c)})
}

func Fail(c *gin.Context, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = apperror.Internal
	}
	c.JSON(appErr.HTTPStatus, Envelope{Code: appErr.Code, Message: appErr.Message, Data: nil, RequestID: requestID(c)})
}

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	id, _ := value.(string)
	return id
}
