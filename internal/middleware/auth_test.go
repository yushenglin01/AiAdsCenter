package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/adnova/internal/common/identity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequireRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		roles []string
		want  int
	}{
		{name: "manager allowed", roles: []string{"MANAGER"}, want: http.StatusOK},
		{name: "operator denied", roles: []string{"OPERATOR"}, want: http.StatusForbidden},
		{name: "missing denied", roles: nil, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/approval", func(c *gin.Context) { c.Set(identity.RolesKey, tt.roles) }, RequireRoles("ADMIN", "MANAGER"), func(c *gin.Context) { c.Status(http.StatusOK) })
			request := httptest.NewRequest(http.MethodGet, "/approval", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tt.want, response.Code)
		})
	}
}
