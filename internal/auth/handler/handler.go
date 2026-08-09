package handler

import (
	"github.com/example/adnova/internal/auth/dto"
	"github.com/example/adnova/internal/auth/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) RegistrationConfig(c *gin.Context) {
	response.OK(c, h.service.RegistrationConfig())
}

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.Register(c.Request.Context(), req, requestMetadata(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Accepted(c, result)
}

func (h *Handler) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.ResendVerification(c.Request.Context(), req.Email)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Accepted(c, result)
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.VerifyEmail(c.Request.Context(), req.Token, requestMetadata(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) Me(c *gin.Context) {
	result, err := h.service.CurrentUser(c.Request.Context(), identity.TenantID(c), identity.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListRegistrationApplications(c *gin.Context) {
	result, err := h.service.ListRegistrationApplications(c.Request.Context(), identity.TenantID(c), c.Query("status"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ListRoles(c *gin.Context) {
	result, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) ApproveRegistration(c *gin.Context) {
	var req dto.ApproveRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.ApproveRegistration(c.Request.Context(), adminActor(c), c.Param("id"), req.Roles)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) RejectRegistration(c *gin.Context) {
	var req dto.RejectRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, apperror.InvalidArgument)
		return
	}
	result, err := h.service.RejectRegistration(c.Request.Context(), adminActor(c), c.Param("id"), req.Reason)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

func requestMetadata(c *gin.Context) service.RequestMetadata {
	return service.RequestMetadata{RequestID: contextValue(c, "request_id"), TraceID: contextValue(c, "trace_id"), IPAddress: c.ClientIP()}
}

func adminActor(c *gin.Context) service.AdminActor {
	return service.AdminActor{TenantID: identity.TenantID(c), UserID: identity.UserID(c), RequestMetadata: requestMetadata(c)}
}

func contextValue(c *gin.Context, key string) string {
	value, _ := c.Get(key)
	result, _ := value.(string)
	return result
}
