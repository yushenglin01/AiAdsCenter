package bootstrap

import (
	"fmt"
	"time"

	campaigndomain "github.com/example/adnova/internal/campaign/domain"
	"github.com/example/adnova/internal/common/model"
	creativedomain "github.com/example/adnova/internal/creative/domain"
	gamedomain "github.com/example/adnova/internal/game/domain"
	rulesdomain "github.com/example/adnova/internal/rules/domain"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const DemoTenantID = "00000000-0000-4000-8000-000000000001"

func SeedDemo(db *gorm.DB) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Demo@123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}
	tenantRows := []model.Tenant{{ID: DemoTenantID, Slug: "demo-company", Name: "Demo Company"}}
	roleRows := []model.Role{
		{ID: "10000000-0000-4000-8000-000000000001", Code: "ADMIN", Name: "管理员"},
		{ID: "10000000-0000-4000-8000-000000000002", Code: "MANAGER", Name: "投放负责人"},
		{ID: "10000000-0000-4000-8000-000000000003", Code: "OPERATOR", Name: "投放优化师"},
		{ID: "10000000-0000-4000-8000-000000000004", Code: "ANALYST", Name: "数据分析师"},
		{ID: "10000000-0000-4000-8000-000000000005", Code: "VIEWER", Name: "只读用户"},
		{ID: "10000000-0000-4000-8000-000000000006", Code: "SYSTEM_AGENT", Name: "系统 Agent"},
	}
	seedApprovedAt := time.Now().UTC()
	userRows := []model.User{
		{ID: "20000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, Username: "admin", Email: "admin@demo.local", DisplayName: "Demo Admin", Department: "平台管理", JobTitle: "系统管理员", PasswordHash: string(passwordHash), Status: "ACTIVE", EmailVerifiedAt: &seedApprovedAt, ApprovedAt: &seedApprovedAt},
		{ID: "20000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, Username: "manager", Email: "manager@demo.local", DisplayName: "Demo Manager", Department: "广告投放", JobTitle: "投放负责人", PasswordHash: string(passwordHash), Status: "ACTIVE", EmailVerifiedAt: &seedApprovedAt, ApprovedAt: &seedApprovedAt},
		{ID: "20000000-0000-4000-8000-000000000003", TenantID: DemoTenantID, Username: "operator", Email: "operator@demo.local", DisplayName: "Demo Operator", Department: "广告投放", JobTitle: "投放优化师", PasswordHash: string(passwordHash), Status: "ACTIVE", EmailVerifiedAt: &seedApprovedAt, ApprovedAt: &seedApprovedAt},
		{ID: "20000000-0000-4000-8000-000000000006", TenantID: DemoTenantID, Username: "system-agent", Email: "system-agent@demo.local", DisplayName: "System Agent", Department: "系统", JobTitle: "服务账号", PasswordHash: string(passwordHash), Status: "SYSTEM", EmailVerifiedAt: &seedApprovedAt},
	}
	links := []model.UserRole{
		{TenantID: DemoTenantID, UserID: userRows[0].ID, RoleID: roleRows[0].ID},
		{TenantID: DemoTenantID, UserID: userRows[1].ID, RoleID: roleRows[1].ID},
		{TenantID: DemoTenantID, UserID: userRows[2].ID, RoleID: roleRows[2].ID},
		{TenantID: DemoTenantID, UserID: userRows[3].ID, RoleID: roleRows[5].ID},
	}
	gameRows := []gamedomain.Game{{ID: "30000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, Code: "game-1001", Name: "Galaxy Adventure", PackageName: "com.demo.galaxyadventure", Timezone: "UTC", Currency: "USD", Status: "ACTIVE"}}
	channelRows := []campaigndomain.Channel{
		{ID: "40000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, Code: "META", Name: "Meta", Provider: "MOCK", Status: "ACTIVE"},
		{ID: "40000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, Code: "GOOGLE", Name: "Google", Provider: "MOCK", Status: "ACTIVE"},
		{ID: "40000000-0000-4000-8000-000000000003", TenantID: DemoTenantID, Code: "TIKTOK", Name: "TikTok", Provider: "MOCK", Status: "ACTIVE"},
	}
	campaignRows := []campaigndomain.Campaign{
		{ID: "50000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, GameID: gameRows[0].ID, ChannelID: channelRows[0].ID, ExternalID: "meta-us-001", Name: "Meta US Growth", Country: "US", DailyBudget: decimal.RequireFromString("3500"), Currency: "USD", Status: "ACTIVE"},
		{ID: "50000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, GameID: gameRows[0].ID, ChannelID: channelRows[1].ID, ExternalID: "google-jp-001", Name: "Google JP Stable", Country: "JP", DailyBudget: decimal.RequireFromString("2200"), Currency: "USD", Status: "ACTIVE"},
		{ID: "50000000-0000-4000-8000-000000000003", TenantID: DemoTenantID, GameID: gameRows[0].ID, ChannelID: channelRows[2].ID, ExternalID: "tiktok-kr-001", Name: "TikTok KR Scale", Country: "KR", DailyBudget: decimal.RequireFromString("1800"), Currency: "USD", Status: "ACTIVE"},
	}
	creativeRows := []creativedomain.Creative{
		{ID: "60000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, CampaignID: campaignRows[0].ID, ExternalID: "creative-video-302", Name: "Galaxy Launch Video 302", Type: "VIDEO", Status: "ACTIVE"},
		{ID: "60000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, CampaignID: campaignRows[1].ID, ExternalID: "creative-image-101", Name: "JP Stable Image 101", Type: "IMAGE", Status: "ACTIVE"},
	}
	benchmarkStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	benchmarkEnd := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	benchmarkRows := []rulesdomain.BusinessBenchmark{
		{ID: "70000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, GameID: gameRows[0].ID, MetricCode: "TARGET_ROAS_D7", Value: decimal.RequireFromString("1.30"), PeriodStart: benchmarkStart, PeriodEnd: benchmarkEnd},
		{ID: "70000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, GameID: gameRows[0].ID, MetricCode: "BENCHMARK_CPI", Value: decimal.RequireFromString("8.50"), PeriodStart: benchmarkStart, PeriodEnd: benchmarkEnd},
		{ID: "70000000-0000-4000-8000-000000000003", TenantID: DemoTenantID, GameID: gameRows[0].ID, MetricCode: "BENCHMARK_PAYER_RATE", Value: decimal.RequireFromString("0.042"), PeriodStart: benchmarkStart, PeriodEnd: benchmarkEnd},
	}
	ruleRows := []rulesdomain.AnalysisRule{
		{ID: "71000000-0000-4000-8000-000000000001", TenantID: DemoTenantID, Code: "ROAS_BELOW_TARGET", Category: "BUSINESS", Name: "D7 ROAS 低于目标", Severity: "HIGH", Threshold: decimal.RequireFromString("1.30"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000002", TenantID: DemoTenantID, Code: "D1_ROAS_DECLINE", Category: "BUSINESS", Name: "D1 ROAS 连续下降", Severity: "MEDIUM", Threshold: decimal.Zero, ConsecutiveDays: 3, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000003", TenantID: DemoTenantID, Code: "CPI_ABOVE_BENCHMARK", Category: "BUSINESS", Name: "CPI 高于基准", Severity: "MEDIUM", Threshold: decimal.RequireFromString("8.50"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000004", TenantID: DemoTenantID, Code: "PAYER_RATE_BELOW_BENCHMARK", Category: "BUSINESS", Name: "付费率低于基准", Severity: "HIGH", Threshold: decimal.RequireFromString("0.042"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000005", TenantID: DemoTenantID, Code: "BUDGET_OVER_90_ROAS_LOW", Category: "BUSINESS", Name: "预算高消耗低回收", Severity: "HIGH", Threshold: decimal.RequireFromString("0.90"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000006", TenantID: DemoTenantID, Code: "INSTALL_ATTRIBUTION_GAP", Category: "ATTRIBUTION", Name: "安装归因偏差", Severity: "HIGH", Threshold: decimal.RequireFromString("0.10"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000007", TenantID: DemoTenantID, Code: "REVENUE_ATTRIBUTION_GAP", Category: "ATTRIBUTION", Name: "收入归因偏差", Severity: "MEDIUM", Threshold: decimal.RequireFromString("0.10"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000008", TenantID: DemoTenantID, Code: "DATA_MISSING_OR_DELAY", Category: "ATTRIBUTION", Name: "数据缺失或延迟", Severity: "MEDIUM", Threshold: decimal.NewFromInt(1), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000009", TenantID: DemoTenantID, Code: "CREATIVE_CTR_DECLINE", Category: "CREATIVE", Name: "素材 CTR 下降", Severity: "MEDIUM", Threshold: decimal.RequireFromString("0.20"), ConsecutiveDays: 7, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000010", TenantID: DemoTenantID, Code: "CREATIVE_FREQUENCY_HIGH", Category: "CREATIVE", Name: "素材频次过高", Severity: "HIGH", Threshold: decimal.RequireFromString("4.0"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000011", TenantID: DemoTenantID, Code: "CREATIVE_FATIGUE_HIGH", Category: "CREATIVE", Name: "素材疲劳风险高", Severity: "HIGH", Threshold: decimal.RequireFromString("0.80"), ConsecutiveDays: 1, Enabled: true},
		{ID: "71000000-0000-4000-8000-000000000012", TenantID: DemoTenantID, Code: "HIGH_SPEND_LOW_CONVERSION", Category: "CREATIVE", Name: "高消耗低转化", Severity: "MEDIUM", Threshold: decimal.RequireFromString("8.50"), ConsecutiveDays: 1, Enabled: true},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for i := range tenantRows {
			if err := tx.Where("id = ?", tenantRows[i].ID).FirstOrCreate(&tenantRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range roleRows {
			if err := tx.Where("id = ?", roleRows[i].ID).FirstOrCreate(&roleRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range userRows {
			if err := tx.Where("id = ?", userRows[i].ID).FirstOrCreate(&userRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range links {
			if err := tx.Where("tenant_id = ? AND user_id = ? AND role_id = ?", links[i].TenantID, links[i].UserID, links[i].RoleID).FirstOrCreate(&links[i]).Error; err != nil {
				return err
			}
		}
		for i := range gameRows {
			if err := tx.Where("id = ?", gameRows[i].ID).FirstOrCreate(&gameRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range channelRows {
			if err := tx.Where("id = ?", channelRows[i].ID).FirstOrCreate(&channelRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range campaignRows {
			if err := tx.Where("id = ?", campaignRows[i].ID).FirstOrCreate(&campaignRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range creativeRows {
			if err := tx.Where("id = ?", creativeRows[i].ID).FirstOrCreate(&creativeRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range benchmarkRows {
			if err := tx.Where("id = ?", benchmarkRows[i].ID).FirstOrCreate(&benchmarkRows[i]).Error; err != nil {
				return err
			}
		}
		for i := range ruleRows {
			if err := tx.Where("id = ?", ruleRows[i].ID).FirstOrCreate(&ruleRows[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
