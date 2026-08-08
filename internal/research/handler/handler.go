package handler

import (
	"strconv"

	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	researchdomain "github.com/example/adnova/internal/research/domain"
	researchservice "github.com/example/adnova/internal/research/service"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *researchservice.Service }

func New(service *researchservice.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	var input researchservice.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	row, err := h.service.Create(c, actor(c), input)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, row)
}

func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	rows, err := h.service.List(c, identity.TenantID(c), researchdomain.Filter{GameID: c.Query("game_id"), CampaignID: c.Query("campaign_id"), Category: c.Query("category"), Status: c.Query("status"), Limit: limit})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) Verify(c *gin.Context) { h.decide(c, researchdomain.StatusVerified) }
func (h *Handler) Reject(c *gin.Context) { h.decide(c, researchdomain.StatusRejected) }

func (h *Handler) decide(c *gin.Context, status string) {
	var input struct {
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, apperror.Validation("请求参数格式错误"))
		return
	}
	row, err := h.service.Decide(c, actor(c), c.Param("id"), researchservice.DecisionInput{Status: status, Comment: input.Comment})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, row)
}

func actor(c *gin.Context) researchservice.Actor {
	traceID, _ := c.Get("trace_id")
	requestID, _ := c.Get("request_id")
	trace, _ := traceID.(string)
	request, _ := requestID.(string)
	return researchservice.Actor{TenantID: identity.TenantID(c), UserID: identity.UserID(c), TraceID: trace, RequestID: request}
}
