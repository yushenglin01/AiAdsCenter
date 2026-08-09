package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validOrganizationInput() OrganizationInput {
	return OrganizationInput{
		TenantID: "00000000-0000-4000-8000-000000000002", TenantSlug: "acme-games", TenantName: "Acme Games",
		AdminUsername: "platform.admin", AdminEmail: "admin@acme.example", AdminName: "平台管理员",
		AdminDepartment: "平台管理", AdminJobTitle: "系统管理员", AdminPassword: "SecurePass!2026",
	}
}

func TestNormalizeOrganizationInput(t *testing.T) {
	input := validOrganizationInput()
	input.TenantSlug = " ACME-GAMES "
	input.AdminEmail = " ADMIN@ACME.EXAMPLE "
	normalized, err := normalizeOrganizationInput(input)
	require.NoError(t, err)
	require.Equal(t, "acme-games", normalized.TenantSlug)
	require.Equal(t, "admin@acme.example", normalized.AdminEmail)
}

func TestNormalizeOrganizationInputRejectsWeakPassword(t *testing.T) {
	input := validOrganizationInput()
	input.AdminPassword = "only-lowercase"
	_, err := normalizeOrganizationInput(input)
	require.ErrorContains(t, err, "password")
}

func TestNormalizeOrganizationInputRejectsInvalidTenantID(t *testing.T) {
	input := validOrganizationInput()
	input.TenantID = "demo"
	_, err := normalizeOrganizationInput(input)
	require.ErrorContains(t, err, "UUID")
}
