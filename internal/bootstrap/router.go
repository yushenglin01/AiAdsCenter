package bootstrap

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	agenthandler "github.com/example/adnova/internal/agent/handler"
	analysisservice "github.com/example/adnova/internal/analysis/service"
	approvalhandler "github.com/example/adnova/internal/approval/handler"
	approvalrepo "github.com/example/adnova/internal/approval/repository"
	approvalservice "github.com/example/adnova/internal/approval/service"
	attributionhandler "github.com/example/adnova/internal/attribution/handler"
	attributionrepo "github.com/example/adnova/internal/attribution/repository"
	attributionservice "github.com/example/adnova/internal/attribution/service"
	audithandler "github.com/example/adnova/internal/audit/handler"
	auditrepo "github.com/example/adnova/internal/audit/repository"
	auditservice "github.com/example/adnova/internal/audit/service"
	authemail "github.com/example/adnova/internal/auth/email"
	authhandler "github.com/example/adnova/internal/auth/handler"
	authrepo "github.com/example/adnova/internal/auth/repository"
	authservice "github.com/example/adnova/internal/auth/service"
	businesshandler "github.com/example/adnova/internal/business/handler"
	campaignhandler "github.com/example/adnova/internal/campaign/handler"
	campaignrepo "github.com/example/adnova/internal/campaign/repository"
	campaignservice "github.com/example/adnova/internal/campaign/service"
	"github.com/example/adnova/internal/common/response"
	"github.com/example/adnova/internal/config"
	creativeanalysishandler "github.com/example/adnova/internal/creative/analysis/handler"
	creativeanalysisrepo "github.com/example/adnova/internal/creative/analysis/repository"
	creativeanalysisservice "github.com/example/adnova/internal/creative/analysis/service"
	creativehandler "github.com/example/adnova/internal/creative/handler"
	creativerepo "github.com/example/adnova/internal/creative/repository"
	creativeservice "github.com/example/adnova/internal/creative/service"
	dashboardhandler "github.com/example/adnova/internal/dashboard/handler"
	dashboardrepo "github.com/example/adnova/internal/dashboard/repository"
	dashboardservice "github.com/example/adnova/internal/dashboard/service"
	dataqualityrepo "github.com/example/adnova/internal/dataquality/repository"
	dataqualityservice "github.com/example/adnova/internal/dataquality/service"
	gamehandler "github.com/example/adnova/internal/game/handler"
	gamerepo "github.com/example/adnova/internal/game/repository"
	gameservice "github.com/example/adnova/internal/game/service"
	ingestionhandler "github.com/example/adnova/internal/ingestion/handler"
	ingestionrepo "github.com/example/adnova/internal/ingestion/repository"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	metricshandler "github.com/example/adnova/internal/metrics/handler"
	metricsrepo "github.com/example/adnova/internal/metrics/repository"
	metricsservice "github.com/example/adnova/internal/metrics/service"
	appmiddleware "github.com/example/adnova/internal/middleware"
	mmpappsflyer "github.com/example/adnova/internal/mmp/appsflyer"
	mmphandler "github.com/example/adnova/internal/mmp/handler"
	mmprepo "github.com/example/adnova/internal/mmp/repository"
	mmpservice "github.com/example/adnova/internal/mmp/service"
	modelusagehandler "github.com/example/adnova/internal/modelusage/handler"
	modelusagerepo "github.com/example/adnova/internal/modelusage/repository"
	modelusageservice "github.com/example/adnova/internal/modelusage/service"
	notificationhandler "github.com/example/adnova/internal/notification/handler"
	notificationrepo "github.com/example/adnova/internal/notification/repository"
	notificationservice "github.com/example/adnova/internal/notification/service"
	openclawservice "github.com/example/adnova/internal/openclaw/service"
	recommendationhandler "github.com/example/adnova/internal/recommendation/handler"
	recommendationrepo "github.com/example/adnova/internal/recommendation/repository"
	recommendationservice "github.com/example/adnova/internal/recommendation/service"
	researchhandler "github.com/example/adnova/internal/research/handler"
	researchrepo "github.com/example/adnova/internal/research/repository"
	researchservice "github.com/example/adnova/internal/research/service"
	ruleshandler "github.com/example/adnova/internal/rules/handler"
	rulesrepo "github.com/example/adnova/internal/rules/repository"
	rulesservice "github.com/example/adnova/internal/rules/service"
	"github.com/example/adnova/internal/taskqueue"
	tenanthandler "github.com/example/adnova/internal/tenant/handler"
	tenantrepo "github.com/example/adnova/internal/tenant/repository"
	tenantservice "github.com/example/adnova/internal/tenant/service"
	workflowhandler "github.com/example/adnova/internal/workflow/handler"
	workflowrepo "github.com/example/adnova/internal/workflow/repository"
	workflowservice "github.com/example/adnova/internal/workflow/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewRouter(cfg config.Config, logger *zap.Logger, db *gorm.DB, redisClient *redis.Client, enqueuer taskqueue.Enqueuer) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery(), appmiddleware.CORS(cfg.HTTP.AllowedOrigin), appmiddleware.RequestContext(logger))

	sqlDB, _ := db.DB()
	router.GET("/health", healthHandler(sqlDB, redisClient))

	userRepo := authrepo.New(db)
	auditService := auditservice.New(auditrepo.New(db))
	var registrationSender authemail.Sender = authemail.NewLogSender(logger)
	if cfg.Registration.Mail.Provider == "smtp" {
		registrationSender = authemail.NewSMTPSender(cfg.Registration.Mail)
	}
	authService := authservice.New(userRepo, cfg.JWT, cfg.Tenant.DefaultID, authservice.WithRegistration(userRepo, registrationSender, cfg.Registration, auditService))
	authHandler := authhandler.New(authService)
	tenantHandler := tenanthandler.New(tenantservice.New(tenantrepo.New(db)))
	gameHandler := gamehandler.New(gameservice.New(gamerepo.New(db)))
	campaignHandler := campaignhandler.New(campaignservice.New(campaignrepo.New(db)))
	creativeHandler := creativehandler.New(creativeservice.New(creativerepo.New(db)))
	ingestionService := ingestionservice.New(ingestionrepo.New(db))
	ingestionHandler := ingestionhandler.New(ingestionService)
	metricsService := metricsservice.New(metricsrepo.New(db))
	rulesRepository := rulesrepo.New(db)
	attributionRepository := attributionrepo.New(db)
	creativeAnalysisRepository := creativeanalysisrepo.New(db)
	rulesService := rulesservice.New(rulesRepository, metricsService)
	attributionService := attributionservice.New(attributionRepository)
	creativeAnalysisService := creativeanalysisservice.New(creativeAnalysisRepository)
	dataQualityService := dataqualityservice.New(dataqualityrepo.New(db))
	researchService := researchservice.New(researchrepo.New(db), auditService)
	researchHandler := researchhandler.New(researchService)
	pipeline := analysisservice.NewPipeline(metricsService, rulesService, attributionService, creativeAnalysisService)
	metricsHandler := metricshandler.New(metricsService, pipeline)
	rulesHandler := ruleshandler.New(rulesService)
	attributionHandler := attributionhandler.New(attributionService)
	creativeAnalysisHandler := creativeanalysishandler.New(creativeAnalysisService)
	businessService, err := NewBusinessService(cfg, db, enqueuer)
	if err != nil {
		panic(err)
	}
	businessHandler := businesshandler.New(businessService)
	agentRegistry, err := NewAgentRegistry(metricsService, rulesService, attributionService, creativeAnalysisService, businessService, dataQualityService, researchService)
	if err != nil {
		panic(err)
	}
	agentHandler := agenthandler.New(agentRegistry)
	notificationService := notificationservice.New(notificationrepo.New(db))
	notificationHandler := notificationhandler.New(notificationService)
	approvalService := approvalservice.New(approvalrepo.New(db), auditService)
	approvalHandler := approvalhandler.New(approvalService)
	workflowService := workflowservice.New(workflowrepo.New(db), agentRegistry, businessService, notificationService)
	workflowHandler := workflowhandler.New(workflowService, openclawservice.New(workflowService, approvalService, notificationService))
	appsFlyerClient := mmpappsflyer.New(mmpappsflyer.Config{BaseURL: cfg.AppsFlyer.BaseURL, Token: cfg.AppsFlyer.APIToken, Timeout: cfg.AppsFlyer.Timeout, MaxRetries: cfg.AppsFlyer.MaxRetries, PurchaseEvents: cfg.AppsFlyer.PurchaseEvents})
	mmpHandler := mmphandler.New(mmpservice.New(mmprepo.New(db), appsFlyerClient, ingestionService, auditService, cfg.AppsFlyer.MaxRangeDays))
	auditHandler := audithandler.New(auditService)
	recommendationHandler := recommendationhandler.New(recommendationservice.New(recommendationrepo.New(db)))
	modelUsageHandler := modelusagehandler.New(modelusageservice.New(modelusagerepo.New(db)))
	dashboardHandler := dashboardhandler.New(dashboardservice.New(dashboardrepo.New(db)))

	v1 := router.Group("/api/v1")
	authRoutes := v1.Group("/auth")
	authRoutes.GET("/registration-config", authHandler.RegistrationConfig)
	authRoutes.POST("/register", authHandler.Register)
	authRoutes.POST("/verify-email", authHandler.VerifyEmail)
	authRoutes.POST("/resend-verification", authHandler.ResendVerification)
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/refresh", authHandler.Refresh)
	authenticated := v1.Group("")
	authenticated.Use(appmiddleware.Authenticate(authService))
	authenticated.GET("/auth/me", authHandler.Me)
	authenticated.GET("/tenants/current", tenantHandler.Current)
	authenticated.GET("/games", gameHandler.List)
	authenticated.GET("/games/:id", gameHandler.Get)
	authenticated.POST("/games", appmiddleware.RequireRoles("ADMIN", "MANAGER"), gameHandler.Create)
	authenticated.PUT("/games/:id", appmiddleware.RequireRoles("ADMIN", "MANAGER"), gameHandler.Update)
	authenticated.GET("/channels", campaignHandler.ListChannels)
	authenticated.GET("/channels/:id", campaignHandler.GetChannel)
	authenticated.POST("/channels", appmiddleware.RequireRoles("ADMIN"), campaignHandler.CreateChannel)
	authenticated.PUT("/channels/:id", appmiddleware.RequireRoles("ADMIN"), campaignHandler.UpdateChannel)
	authenticated.GET("/campaigns", campaignHandler.ListCampaigns)
	authenticated.GET("/campaigns/:id", campaignHandler.GetCampaign)
	authenticated.POST("/campaigns", appmiddleware.RequireRoles("ADMIN", "MANAGER"), campaignHandler.CreateCampaign)
	authenticated.PUT("/campaigns/:id", appmiddleware.RequireRoles("ADMIN", "MANAGER"), campaignHandler.UpdateCampaign)
	authenticated.GET("/creatives", creativeHandler.List)
	authenticated.GET("/creatives/:id", creativeHandler.Get)
	authenticated.POST("/creatives", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), creativeHandler.Create)
	authenticated.PUT("/creatives/:id", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), creativeHandler.Update)
	authenticated.GET("/imports", ingestionHandler.List)
	authenticated.GET("/imports/:id", ingestionHandler.Get)
	authenticated.POST("/imports/ad-metrics", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), ingestionHandler.ImportAd)
	authenticated.POST("/imports/mmp-metrics", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), ingestionHandler.ImportMMP)
	authenticated.POST("/imports/game-revenue", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), ingestionHandler.ImportRevenue)
	authenticated.POST("/imports/creative-metrics", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), ingestionHandler.ImportCreative)
	authenticated.POST("/imports/batches", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), ingestionHandler.ImportBatch)
	authenticated.GET("/mmp-connections", mmpHandler.ListConnections)
	authenticated.PUT("/mmp-connections/appsflyer", appmiddleware.RequireRoles("ADMIN", "MANAGER"), mmpHandler.ConfigureAppsFlyer)
	authenticated.GET("/mmp-sync-runs", mmpHandler.ListSyncRuns)
	authenticated.POST("/mmp-connections/:id/sync", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR"), mmpHandler.Sync)
	authenticated.GET("/metrics/overview", metricsHandler.Overview)
	authenticated.GET("/metrics/campaigns", metricsHandler.Campaigns)
	authenticated.GET("/metrics/campaigns/:id", metricsHandler.Campaign)
	authenticated.GET("/metrics/trends", metricsHandler.Trends)
	authenticated.POST("/metrics/recalculate", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR", "ANALYST"), metricsHandler.Recalculate)
	authenticated.GET("/analysis/rules", rulesHandler.List)
	authenticated.PUT("/analysis/rules/:id", appmiddleware.RequireRoles("ADMIN", "MANAGER"), rulesHandler.Update)
	authenticated.GET("/analysis/attribution", attributionHandler.List)
	authenticated.GET("/analysis/attribution/:id", attributionHandler.Get)
	authenticated.GET("/analysis/creative", creativeAnalysisHandler.List)
	authenticated.GET("/analysis/creative/:id", creativeAnalysisHandler.Get)
	authenticated.GET("/research/sources", researchHandler.List)
	authenticated.POST("/research/sources", appmiddleware.RequireRoles("ADMIN", "MANAGER", "ANALYST"), researchHandler.Create)
	authenticated.POST("/research/sources/:id/verify", appmiddleware.RequireRoles("ADMIN", "MANAGER"), researchHandler.Verify)
	authenticated.POST("/research/sources/:id/reject", appmiddleware.RequireRoles("ADMIN", "MANAGER"), researchHandler.Reject)
	authenticated.POST("/analysis/business", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR", "ANALYST"), businessHandler.Analyze)
	authenticated.GET("/analysis/business/tasks", businessHandler.List)
	authenticated.GET("/analysis/business/tasks/:id", businessHandler.Get)
	authenticated.GET("/analysis/business/tasks/:id/events", businessHandler.Events)
	authenticated.GET("/analysis/business/tasks/:id/report", businessHandler.Report)
	authenticated.GET("/analysis/business/health", businessHandler.Health)
	authenticated.GET("/agents", agentHandler.List)
	authenticated.GET("/agents/:name", agentHandler.Get)
	authenticated.POST("/workflows/analysis", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR", "ANALYST"), workflowHandler.Start)
	authenticated.GET("/workflows", workflowHandler.List)
	authenticated.GET("/workflows/:id", workflowHandler.Get)
	authenticated.GET("/workflows/:id/events", workflowHandler.Events)
	authenticated.POST("/openclaw/commands", appmiddleware.RequireRoles("ADMIN", "MANAGER", "OPERATOR", "ANALYST"), workflowHandler.OpenClawCommand)
	authenticated.GET("/notifications", notificationHandler.List)
	authenticated.POST("/notifications/:id/read", notificationHandler.MarkRead)
	authenticated.GET("/recommendations", recommendationHandler.List)
	authenticated.GET("/recommendations/:id", recommendationHandler.Get)
	authenticated.GET("/approvals", approvalHandler.List)
	authenticated.GET("/approvals/:id", approvalHandler.Get)
	authenticated.POST("/approvals/:id/approve", approvalHandler.Approve)
	authenticated.POST("/approvals/:id/reject", approvalHandler.Reject)
	authenticated.GET("/model-usage", appmiddleware.RequireRoles("ADMIN", "MANAGER"), modelUsageHandler.List)
	authenticated.GET("/model-usage/summary", appmiddleware.RequireRoles("ADMIN", "MANAGER"), modelUsageHandler.Summary)
	authenticated.GET("/audit-logs", appmiddleware.RequireRoles("ADMIN"), auditHandler.List)
	authenticated.GET("/dashboard/operations", dashboardHandler.Summary)
	authenticated.GET("/admin/registration-applications", appmiddleware.RequireRoles("ADMIN"), authHandler.ListRegistrationApplications)
	authenticated.GET("/admin/roles", appmiddleware.RequireRoles("ADMIN"), authHandler.ListRoles)
	authenticated.POST("/admin/registration-applications/:id/approve", appmiddleware.RequireRoles("ADMIN"), authHandler.ApproveRegistration)
	authenticated.POST("/admin/registration-applications/:id/reject", appmiddleware.RequireRoles("ADMIN"), authHandler.RejectRegistration)
	authenticated.GET("/admin/ping", appmiddleware.RequireRoles("ADMIN"), func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})
	return router
}

func healthHandler(db *sql.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		checks := gin.H{"mysql": "up", "redis": "up"}
		status := http.StatusOK
		if db == nil || db.PingContext(ctx) != nil {
			checks["mysql"] = "down"
			status = http.StatusServiceUnavailable
		}
		if redisClient == nil || redisClient.Ping(ctx).Err() != nil {
			checks["redis"] = "down"
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, response.Envelope{Code: 0, Message: "ok", Data: gin.H{"status": map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK], "checks": checks}, RequestID: requestID(c)})
	}
}

func requestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	id, _ := value.(string)
	return id
}
