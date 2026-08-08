package middleware

import (
	"strings"

	"github.com/example/adnova/internal/auth/service"
	"github.com/example/adnova/internal/common/apperror"
	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/response"
	"github.com/gin-gonic/gin"
)

func Authenticate(auth *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Fail(c, apperror.Unauthorized)
			c.Abort()
			return
		}
		claims, err := auth.Parse(parts[1], "access")
		if err != nil {
			response.Fail(c, apperror.Unauthorized)
			c.Abort()
			return
		}
		c.Set(identity.UserIDKey, claims.Subject)
		c.Set(identity.TenantIDKey, claims.TenantID)
		c.Set(identity.RolesKey, claims.Roles)
		c.Next()
	}
}

func RequireRoles(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}
	return func(c *gin.Context) {
		value, _ := c.Get(identity.RolesKey)
		roles, _ := value.([]string)
		for _, role := range roles {
			if _, ok := allowedSet[role]; ok {
				c.Next()
				return
			}
		}
		response.Fail(c, apperror.Forbidden)
		c.Abort()
	}
}
