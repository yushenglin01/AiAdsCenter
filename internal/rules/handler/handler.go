package handler

import (
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/rules/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	result, err := h.service.List(c, identity.TenantID(c), c.Query("game_id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Update(c *gin.Context) {
	var input service.UpdateRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	result, err := h.service.Update(c, identity.TenantID(c), c.Param("id"), input)
	if err != nil {
		response.Fail(c, apperror.Validation(err.Error()))
		return
	}
	response.OK(c, result)
}
