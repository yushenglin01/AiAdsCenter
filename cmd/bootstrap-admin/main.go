package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/example/adnova/internal/bootstrap"
	"github.com/example/adnova/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	tenantID := flag.String("tenant-id", cfg.Tenant.DefaultID, "tenant UUID; defaults to GAI_TENANT_DEFAULT_ID")
	tenantSlug := flag.String("tenant-slug", "", "stable lowercase tenant slug")
	tenantName := flag.String("tenant-name", "", "company or studio name")
	adminUsername := flag.String("admin-username", "", "initial administrator username")
	adminEmail := flag.String("admin-email", "", "initial administrator company email")
	adminName := flag.String("admin-name", "", "initial administrator display name")
	adminDepartment := flag.String("admin-department", "平台管理", "initial administrator department")
	adminJobTitle := flag.String("admin-job-title", "系统管理员", "initial administrator job title")
	flag.Parse()

	passwordBytes, err := io.ReadAll(io.LimitReader(os.Stdin, 1024))
	if err != nil {
		fatal(fmt.Errorf("read administrator password from stdin: %w", err))
	}
	password := strings.TrimRight(string(passwordBytes), "\r\n")
	if password == "" {
		fatal(fmt.Errorf("administrator password must be supplied on stdin"))
	}

	db, err := bootstrap.NewDatabase(cfg.Database)
	if err != nil {
		fatal(err)
	}
	result, err := bootstrap.BootstrapOrganization(db, bootstrap.OrganizationInput{
		TenantID: *tenantID, TenantSlug: *tenantSlug, TenantName: *tenantName,
		AdminUsername: *adminUsername, AdminEmail: *adminEmail, AdminName: *adminName,
		AdminDepartment: *adminDepartment, AdminJobTitle: *adminJobTitle, AdminPassword: password,
	})
	if err != nil {
		fatal(err)
	}
	status := "already initialized"
	if result.Created {
		status = "created"
	}
	fmt.Printf("organization bootstrap %s: tenant_id=%s admin_id=%s\n", status, result.TenantID, result.AdminID)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "bootstrap-admin:", err)
	os.Exit(1)
}
