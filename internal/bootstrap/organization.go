package bootstrap

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/example/adnova/internal/common/identity"
	"github.com/example/adnova/internal/common/model"
	rulesdomain "github.com/example/adnova/internal/rules/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	organizationSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,78}[a-z0-9]$`)
	adminUsernamePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,79}$`)
)

type OrganizationInput struct {
	TenantID        string
	TenantSlug      string
	TenantName      string
	AdminUsername   string
	AdminEmail      string
	AdminName       string
	AdminDepartment string
	AdminJobTitle   string
	AdminPassword   string
}

type OrganizationResult struct {
	TenantID string
	AdminID  string
	Created  bool
}

// BootstrapOrganization creates only the minimum trusted identities needed to
// operate an empty single-tenant installation. It never overwrites an existing
// administrator password and is safe to retry with the same identity values.
func BootstrapOrganization(db *gorm.DB, raw OrganizationInput) (*OrganizationResult, error) {
	input, err := normalizeOrganizationInput(raw)
	if err != nil {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash initial administrator password: %w", err)
	}
	systemPasswordHash, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash system identity password: %w", err)
	}

	result := &OrganizationResult{TenantID: input.TenantID}
	err = db.Transaction(func(tx *gorm.DB) error {
		tenant := model.Tenant{}
		findTenant := tx.Where("id = ?", input.TenantID).First(&tenant)
		if errors.Is(findTenant.Error, gorm.ErrRecordNotFound) {
			tenant = model.Tenant{ID: input.TenantID, Slug: input.TenantSlug, Name: input.TenantName}
			if err := tx.Create(&tenant).Error; err != nil {
				return fmt.Errorf("create tenant: %w", err)
			}
		} else if findTenant.Error != nil {
			return fmt.Errorf("read tenant: %w", findTenant.Error)
		} else if tenant.Slug != input.TenantSlug {
			return fmt.Errorf("tenant %s already exists with slug %q", tenant.ID, tenant.Slug)
		}

		roles, err := ensureStandardRoles(tx)
		if err != nil {
			return err
		}
		adminRole := roles["ADMIN"]
		systemRole := roles["SYSTEM_AGENT"]
		if err := ensureStandardAnalysisRules(tx, input.TenantID); err != nil {
			return err
		}

		systemUser := model.User{}
		findSystem := tx.Where("id = ?", identity.SystemAgentUserID).First(&systemUser)
		if errors.Is(findSystem.Error, gorm.ErrRecordNotFound) {
			systemUser = model.User{
				ID: identity.SystemAgentUserID, TenantID: input.TenantID, Username: "system-agent",
				Email: "system-agent@internal.invalid", DisplayName: "System Agent", Department: "系统",
				JobTitle: "服务账号", PasswordHash: string(systemPasswordHash), Status: "SYSTEM",
			}
			if err := tx.Create(&systemUser).Error; err != nil {
				return fmt.Errorf("create system identity: %w", err)
			}
		} else if findSystem.Error != nil {
			return fmt.Errorf("read system identity: %w", findSystem.Error)
		} else if systemUser.TenantID != input.TenantID {
			return fmt.Errorf("system identity is already bound to tenant %s", systemUser.TenantID)
		}
		if err := ensureUserRole(tx, input.TenantID, systemUser.ID, systemRole.ID); err != nil {
			return err
		}

		admin := model.User{}
		findAdmin := tx.Where("tenant_id = ? AND (username = ? OR email = ?)", input.TenantID, input.AdminUsername, input.AdminEmail).First(&admin)
		if findAdmin.Error == nil {
			if admin.Username != input.AdminUsername || admin.Email != input.AdminEmail || admin.Status != "ACTIVE" {
				return fmt.Errorf("administrator username or email is already in use")
			}
			if err := ensureUserRole(tx, input.TenantID, admin.ID, adminRole.ID); err != nil {
				return err
			}
			result.AdminID = admin.ID
			return nil
		}
		if !errors.Is(findAdmin.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read initial administrator: %w", findAdmin.Error)
		}

		now := time.Now().UTC()
		admin = model.User{
			ID: uuid.NewString(), TenantID: input.TenantID, Username: input.AdminUsername, Email: input.AdminEmail,
			DisplayName: input.AdminName, Department: input.AdminDepartment, JobTitle: input.AdminJobTitle,
			PasswordHash: string(passwordHash), Status: "ACTIVE", EmailVerifiedAt: &now, ApprovedAt: &now,
		}
		admin.ApprovedBy = admin.ID
		if err := tx.Create(&admin).Error; err != nil {
			return fmt.Errorf("create initial administrator: %w", err)
		}
		if err := ensureUserRole(tx, input.TenantID, admin.ID, adminRole.ID); err != nil {
			return err
		}
		result.AdminID = admin.ID
		result.Created = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeOrganizationInput(input OrganizationInput) (OrganizationInput, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.TenantSlug = strings.ToLower(strings.TrimSpace(input.TenantSlug))
	input.TenantName = strings.TrimSpace(input.TenantName)
	input.AdminUsername = strings.ToLower(strings.TrimSpace(input.AdminUsername))
	input.AdminEmail = strings.ToLower(strings.TrimSpace(input.AdminEmail))
	input.AdminName = strings.TrimSpace(input.AdminName)
	input.AdminDepartment = strings.TrimSpace(input.AdminDepartment)
	input.AdminJobTitle = strings.TrimSpace(input.AdminJobTitle)
	if _, err := uuid.Parse(input.TenantID); err != nil {
		return OrganizationInput{}, fmt.Errorf("tenant ID must be a UUID")
	}
	if !organizationSlugPattern.MatchString(input.TenantSlug) {
		return OrganizationInput{}, fmt.Errorf("tenant slug must be 3-80 lowercase letters, numbers or hyphens")
	}
	if length := len([]rune(input.TenantName)); length < 2 || length > 160 {
		return OrganizationInput{}, fmt.Errorf("tenant name must be 2-160 characters")
	}
	if !adminUsernamePattern.MatchString(input.AdminUsername) {
		return OrganizationInput{}, fmt.Errorf("administrator username format is invalid")
	}
	parsedEmail, err := mail.ParseAddress(input.AdminEmail)
	if err != nil || parsedEmail.Address != input.AdminEmail || len(input.AdminEmail) > 254 {
		return OrganizationInput{}, fmt.Errorf("administrator email is invalid")
	}
	if length := len([]rune(input.AdminName)); length < 2 || length > 120 {
		return OrganizationInput{}, fmt.Errorf("administrator name must be 2-120 characters")
	}
	if length := len([]rune(input.AdminDepartment)); length < 2 || length > 120 {
		return OrganizationInput{}, fmt.Errorf("administrator department must be 2-120 characters")
	}
	if len([]rune(input.AdminJobTitle)) > 120 {
		return OrganizationInput{}, fmt.Errorf("administrator job title must not exceed 120 characters")
	}
	if containsControlCharacter(input.TenantName) || containsControlCharacter(input.AdminName) || containsControlCharacter(input.AdminDepartment) || containsControlCharacter(input.AdminJobTitle) {
		return OrganizationInput{}, fmt.Errorf("organization identity fields must not contain control characters")
	}
	if !strongBootstrapPassword(input.AdminPassword) {
		return OrganizationInput{}, fmt.Errorf("administrator password must be 12-128 characters and include uppercase, lowercase, number and symbol")
	}
	return input, nil
}

func ensureStandardRoles(tx *gorm.DB) (map[string]model.Role, error) {
	definitions := []model.Role{
		{ID: "10000000-0000-4000-8000-000000000001", Code: "ADMIN", Name: "管理员"},
		{ID: "10000000-0000-4000-8000-000000000002", Code: "MANAGER", Name: "投放负责人"},
		{ID: "10000000-0000-4000-8000-000000000003", Code: "OPERATOR", Name: "投放优化师"},
		{ID: "10000000-0000-4000-8000-000000000004", Code: "ANALYST", Name: "数据分析师"},
		{ID: "10000000-0000-4000-8000-000000000005", Code: "VIEWER", Name: "只读用户"},
		{ID: "10000000-0000-4000-8000-000000000006", Code: "SYSTEM_AGENT", Name: "系统 Agent"},
	}
	result := make(map[string]model.Role, len(definitions))
	for _, definition := range definitions {
		role := definition
		if err := tx.Where("code = ?", definition.Code).FirstOrCreate(&role).Error; err != nil {
			return nil, fmt.Errorf("ensure role %s: %w", definition.Code, err)
		}
		result[role.Code] = role
	}
	return result, nil
}

func ensureStandardAnalysisRules(tx *gorm.DB, tenantID string) error {
	for _, definition := range standardAnalysisRules(tenantID) {
		rule := definition
		if err := tx.Where("tenant_id = ? AND code = ?", tenantID, definition.Code).FirstOrCreate(&rule).Error; err != nil {
			return fmt.Errorf("ensure analysis rule %s: %w", definition.Code, err)
		}
	}
	return nil
}

func standardAnalysisRules(tenantID string) []rulesdomain.AnalysisRule {
	return []rulesdomain.AnalysisRule{
		{ID: "71000000-0000-4000-8000-000000000001", TenantID: tenantID, Code: "ROAS_BELOW_TARGET", Category: "BUSINESS", Name: "D7 ROAS 低于目标", Severity: "HIGH", Threshold: decimal.RequireFromString("1.30"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000002", TenantID: tenantID, Code: "D1_ROAS_DECLINE", Category: "BUSINESS", Name: "D1 ROAS 连续下降", Severity: "MEDIUM", Threshold: decimal.Zero, ConsecutiveDays: 3, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000003", TenantID: tenantID, Code: "CPI_ABOVE_BENCHMARK", Category: "BUSINESS", Name: "CPI 高于基准", Severity: "MEDIUM", Threshold: decimal.RequireFromString("8.50"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000004", TenantID: tenantID, Code: "PAYER_RATE_BELOW_BENCHMARK", Category: "BUSINESS", Name: "付费率低于基准", Severity: "HIGH", Threshold: decimal.RequireFromString("0.042"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000005", TenantID: tenantID, Code: "BUDGET_OVER_90_ROAS_LOW", Category: "BUSINESS", Name: "预算高消耗低回收", Severity: "HIGH", Threshold: decimal.RequireFromString("0.90"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000006", TenantID: tenantID, Code: "INSTALL_ATTRIBUTION_GAP", Category: "ATTRIBUTION", Name: "安装归因偏差", Severity: "HIGH", Threshold: decimal.RequireFromString("0.10"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000007", TenantID: tenantID, Code: "REVENUE_ATTRIBUTION_GAP", Category: "ATTRIBUTION", Name: "收入归因偏差", Severity: "MEDIUM", Threshold: decimal.RequireFromString("0.10"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000008", TenantID: tenantID, Code: "DATA_MISSING_OR_DELAY", Category: "ATTRIBUTION", Name: "数据缺失或延迟", Severity: "MEDIUM", Threshold: decimal.NewFromInt(1), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000009", TenantID: tenantID, Code: "CREATIVE_CTR_DECLINE", Category: "CREATIVE", Name: "素材 CTR 下降", Severity: "MEDIUM", Threshold: decimal.RequireFromString("0.20"), ConsecutiveDays: 7, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000010", TenantID: tenantID, Code: "CREATIVE_FREQUENCY_HIGH", Category: "CREATIVE", Name: "素材频次过高", Severity: "HIGH", Threshold: decimal.RequireFromString("4.0"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000011", TenantID: tenantID, Code: "CREATIVE_FATIGUE_HIGH", Category: "CREATIVE", Name: "素材疲劳风险高", Severity: "HIGH", Threshold: decimal.RequireFromString("0.80"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000012", TenantID: tenantID, Code: "HIGH_SPEND_LOW_CONVERSION", Category: "CREATIVE", Name: "高消耗低转化", Severity: "MEDIUM", Threshold: decimal.RequireFromString("8.50"), ConsecutiveDays: 1, Enabled: true},
	}
}

func ensureUserRole(tx *gorm.DB, tenantID, userID, roleID string) error {
	link := model.UserRole{TenantID: tenantID, UserID: userID, RoleID: roleID}
	if err := tx.Where("tenant_id = ? AND user_id = ? AND role_id = ?", tenantID, userID, roleID).FirstOrCreate(&link).Error; err != nil {
		return fmt.Errorf("assign identity role: %w", err)
	}
	return nil
}

func strongBootstrapPassword(password string) bool {
	length := len([]rune(password))
	if length < 12 || length > 128 {
		return false
	}
	var upper, lower, digit, symbol bool
	for _, character := range password {
		switch {
		case unicode.IsUpper(character):
			upper = true
		case unicode.IsLower(character):
			lower = true
		case unicode.IsDigit(character):
			digit = true
		default:
			symbol = true
		}
	}
	return upper && lower && digit && symbol
}

func containsControlCharacter(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}
