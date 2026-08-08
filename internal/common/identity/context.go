package identity

import "github.com/gin-gonic/gin"

const (
	UserIDKey         = "user_id"
	TenantIDKey       = "tenant_id"
	RolesKey          = "roles"
	SystemAgentUserID = "20000000-0000-4000-8000-000000000006"
)

func TenantID(c *gin.Context) string {
	value, _ := c.Get(TenantIDKey)
	id, _ := value.(string)
	return id
}

func UserID(c *gin.Context) string {
	value, _ := c.Get(UserIDKey)
	id, _ := value.(string)
	return id
}

func Roles(c *gin.Context) []string {
	value, _ := c.Get(RolesKey)
	roles, _ := value.([]string)
	return append([]string(nil), roles...)
}
