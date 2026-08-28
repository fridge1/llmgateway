package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/alerting"
	"github.com/zhulang/llm-gateway/internal/anthropic"
	"github.com/zhulang/llm-gateway/internal/api"
	"github.com/zhulang/llm-gateway/internal/apikey"
	"github.com/zhulang/llm-gateway/internal/audit"
	"github.com/zhulang/llm-gateway/internal/balancer"
	"github.com/zhulang/llm-gateway/internal/bandwidth"
	"github.com/zhulang/llm-gateway/internal/billing"
	"github.com/zhulang/llm-gateway/internal/chat"
	"github.com/zhulang/llm-gateway/internal/checkin"
	"github.com/zhulang/llm-gateway/internal/codex"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/email"
	"github.com/zhulang/llm-gateway/internal/gemini"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/image"
	"github.com/zhulang/llm-gateway/internal/imageshare"
	"github.com/zhulang/llm-gateway/internal/invoice"
	"github.com/zhulang/llm-gateway/internal/lottery"
	"github.com/zhulang/llm-gateway/internal/rechargelottery"
	"github.com/zhulang/llm-gateway/internal/metrics"
	"github.com/zhulang/llm-gateway/internal/middleware"
	"github.com/zhulang/llm-gateway/internal/moderation"
	"github.com/zhulang/llm-gateway/internal/notification"
	"github.com/zhulang/llm-gateway/internal/notifier"
	"github.com/zhulang/llm-gateway/internal/payment"
	"github.com/zhulang/llm-gateway/internal/ppt"
	"github.com/zhulang/llm-gateway/internal/pricing"
	"github.com/zhulang/llm-gateway/internal/proxy"
	"github.com/zhulang/llm-gateway/internal/public"
	"github.com/zhulang/llm-gateway/internal/referral"
	"github.com/zhulang/llm-gateway/internal/responses"
	"github.com/zhulang/llm-gateway/internal/retention"
	"github.com/zhulang/llm-gateway/internal/router"
	"github.com/zhulang/llm-gateway/internal/sms"
	"github.com/zhulang/llm-gateway/internal/storage"
	"github.com/zhulang/llm-gateway/internal/store"
	"github.com/zhulang/llm-gateway/internal/subscription"
	"github.com/zhulang/llm-gateway/internal/task"
	"github.com/zhulang/llm-gateway/internal/ticket"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	migrationsPath := flag.String("migrations", "migrations", "path to migration files")
	flag.Parse()

	// Load YAML config (used for server settings and as seed).
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Server.LogLevel),
	}))
	slog.SetDefault(logger)

	// Database DSN.
	dsn := cfg.Database.DSN
	if dsn == "" {
		slog.Error("database.dsn is required in config")
		os.Exit(1)
	}
	jwtSecret := cfg.Admin.JWTSecret
	if jwtSecret == "" || jwtSecret == "llm-gateway-default-secret" || jwtSecret == "change-me-in-production" {
		slog.Error("admin.jwt_secret must be set to a unique, secure value (do not use default)")
		os.Exit(1)
	}

	// Open PostgreSQL.
	pgStore, err := store.OpenPostgres(dsn, cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns, cfg.Database.ConnMaxLifetime)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer pgStore.Close()

	// Run migrations.
	driver, err := postgres.WithInstance(pgStore.DB(), &postgres.Config{})
	if err != nil {
		slog.Error("failed to create migrate driver", "error", err)
		os.Exit(1)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+*migrationsPath,
		"postgres", driver,
	)
	if err != nil {
		slog.Error("failed to create migrate instance", "error", err)
		os.Exit(1)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")

	// Build config from DB models + YAML server settings.
	dbCfg, err := store.ToConfig(pgStore, cfg)
	if err != nil {
		slog.Error("failed to load config from DB", "error", err)
		os.Exit(1)
	}
	cfgHolder := config.NewHolder(dbCfg)

	// Build initial router, balancer, API key cache, pricing cache, proxy handler, and admin handler.
	rt := router.NewFromConfig(dbCfg)
	lb := balancer.NewRoundRobin()
	keyCache := apikey.NewCache(1000, 5*time.Minute)
	pricingCache := pricing.NewCache(pgStore, 5*time.Minute)
	billingService := billing.NewBillingService(pgStore, pricingCache, cfgHolder)
	sharedClient := proxy.NewSharedHTTPClient()
	touchBatcher := apikey.NewTouchBatcher(pgStore, 5*time.Second)
	defer touchBatcher.Stop()
	activeBatcher := apikey.NewActiveBatcher(pgStore, 60*time.Second)
	defer activeBatcher.Stop()

	// Subscription service.
	subscriptionService := subscription.NewService(pgStore, pricingCache)
	subscriptionHandler := subscription.NewHandler(subscriptionService, pgStore)

	// Create shared Core for all proxy handlers (billing worker pool lives here).
	core := proxy.NewCore(cfgHolder, rt, lb, pgStore, keyCache, billingService, sharedClient, touchBatcher)
	core.SubscriptionService = subscriptionService
	core.ActiveBatcher = activeBatcher
	defer core.StopBillingWorkers()

	proxyHandler := proxy.NewHandler(cfgHolder, rt, lb, pgStore, keyCache, billingService, sharedClient, touchBatcher, core)
	playgroundHandler := proxy.NewPlaygroundHandler(cfgHolder, pgStore, core, sharedClient)
	adminHandler := admin.NewHandler(cfgHolder, rt, pgStore.DB(), pgStore)
	adminHandler.SetAPIKeyAuth(func(r *http.Request) bool {
		_, err := core.AuthenticateAny(r)
		return err == nil
	})
	anthropicHandler := anthropic.NewHandler(core)
	responsesHandler := responses.NewHandler(core)
	var bwLimiter *bandwidth.Limiter
	if cfg.Server.ImageBandwidth.MaxConcurrent > 0 {
		bwLimiter = bandwidth.NewLimiter(
			cfg.Server.ImageBandwidth.MaxConcurrent,
			cfg.Server.ImageBandwidth.QueueTimeout,
		)
		slog.Info("image bandwidth limiter enabled",
			"max_concurrent", cfg.Server.ImageBandwidth.MaxConcurrent,
			"queue_timeout", cfg.Server.ImageBandwidth.QueueTimeout)
	}

	imageHandler := image.NewHandler(core, pricingCache, bwLimiter)
	geminiHandler := gemini.NewHandler(core, bwLimiter)

	// TOS client for image storage
	var tosClient *storage.TOSClient
	tosConfigComplete := cfg.TOS.Endpoint != "" &&
		cfg.TOS.Region != "" &&
		cfg.TOS.Bucket != "" &&
		cfg.TOS.AccessKey != "" &&
		cfg.TOS.SecretKey != ""

	if tosConfigComplete {
		var err error
		tosClient, err = storage.NewTOSClient(storage.TOSConfig{
			Endpoint:  cfg.TOS.Endpoint,
			Region:    cfg.TOS.Region,
			Bucket:    cfg.TOS.Bucket,
			AccessKey: cfg.TOS.AccessKey,
			SecretKey: cfg.TOS.SecretKey,
			URLPrefix: cfg.TOS.URLPrefix,
		})
		if err != nil {
			slog.Error("failed to create TOS client, image generation will be unavailable",
				"error", err)
			tosClient = nil
		} else {
			slog.Info("TOS client initialized successfully",
				"endpoint", cfg.TOS.Endpoint,
				"bucket", cfg.TOS.Bucket)
		}
	} else {
		slog.Warn("TOS configuration incomplete, image generation will be unavailable. " +
			"Please configure TOS settings in config.yaml")
	}

	// Image generation service and handlers
	imageService := image.NewService(pgStore, billingService, subscriptionService, tosClient, core, cfg.Server.ImageUpstreamMaxConcurrency)
	imageService.StartWorkers(8)
	taskHandler := image.NewTaskHandler(imageService)
	publicTaskHandler := image.NewPublicTaskHandler(imageService)

	// Image-share (per-user dispatch keys)
	imageShareStore := imageshare.NewStore(pgStore.DB())
	imageShareAuth := imageshare.NewAuthHandler(imageShareStore, jwtSecret)
	imageShareKeys := imageshare.NewKeysHandler(imageShareStore)
	imageShareAdmin := imageshare.NewAdminHandler(imageShareStore)
	taskHandler.SetImageShareStore(imageShareStore)
	imageService.SetImageShareStore(imageShareStore)

	// PPT generation service and handlers
	pptService := ppt.NewService(pgStore, billingService, core, tosClient)
	pptService.StartWorkers(3)
	pptHandler := ppt.NewHandler(pptService)

	// Rebuild router from DB (called after model CRUD operations).
	// currentRouter tracks the active router to preserve circuit breaker state across rebuilds.
	currentRouter := rt
	rebuildRouter := func() {
		newCfg, err := store.ToConfig(pgStore, cfg)
		if err != nil {
			slog.Error("failed to rebuild config from DB", "error", err)
			return
		}
		newRt := router.NewFromConfigWithBreakers(newCfg, currentRouter)
		currentRouter = newRt
		cfgHolder.Swap(newCfg)
		proxyHandler.SetRouter(newRt)
		adminHandler.SetRouter(newRt)
		anthropicHandler.SetRouter(newRt)
		responsesHandler.SetRouter(newRt)
		imageHandler.SetRouter(newRt)
		geminiHandler.SetRouter(newRt)
		slog.Info("router rebuilt from DB")
	}

	// SMS sender and code store.
	smsSender := sms.NewVolcengineSender(cfg.SMS.AccessKey, cfg.SMS.SecretKey, cfg.SMS.SignName, cfg.SMS.SmsAccount)
	smsCodeStore := sms.NewCodeStore()

	// Email sender and code store.
	// 注意：需要在 config.yaml 中配置 email 字段，并在 .env 中设置环境变量
	// EMAIL_ACCESS_KEY, EMAIL_SECRET_KEY 等
	var emailSender email.Sender
	emailCodeStore := email.NewCodeStore()
	if cfg.Email.Provider == "aliyun" && cfg.Email.AccessKey != "" {
		var err error
		emailSender, err = email.NewAliyunSender(
			cfg.Email.AccessKey,
			cfg.Email.SecretKey,
			cfg.Email.Region,
			cfg.Email.FromEmail,
			cfg.Email.FromName,
		)
		if err != nil {
			slog.Error("failed to create email sender", "error", err)
			slog.Warn("email authentication will be unavailable")
			emailSender = nil
		} else {
			slog.Info("email sender initialized successfully", "provider", "aliyun")
		}
	} else {
		slog.Warn("email configuration not found or incomplete, email authentication will be unavailable")
		emailSender = nil
	}

	// API handlers.
	authHandler := admin.NewAuthHandler(
		pgStore,
		jwtSecret,
		smsSender,
		smsCodeStore,
		cfg.SMS.Templates,
		emailSender,
		emailCodeStore,
		cfg.Email.Templates,
		cfg.Promotion.TrialCreditsCNY,
		cfg.Promotion.FirstRechargeBonusCNY,
		cfg.Admin.InitToken,
	)
	modelHandler := admin.NewModelHandler(pgStore, rebuildRouter)
	apikeyHandler := apikey.NewHandler(pgStore, keyCache)
	billingHandler := billing.NewBillingHandler(pgStore)
	pricingHandler := pricing.NewPricingHandler(pgStore, pricingCache)
	auditLogger := audit.NewLogger(pgStore.DB())
	auditHandler := audit.NewHandler(pgStore.DB())
	userHandler := admin.NewUserHandler(pgStore, auditLogger)
	announcementHandler := admin.NewAnnouncementHandler(pgStore)
	rechargePromotionHandler := admin.NewRechargePromotionHandler(pgStore)
	notificationHandler := notification.NewHandler(pgStore)
	notifierService := notifier.New(pgStore, cfgHolder, smsSender)
	notifierService.Start()
	defer notifierService.Stop()
	billingService.SetNotifyFunc(notifierService.Notify)
	notifierHandler := notifier.NewHandler(pgStore)
	alertingHandler := alerting.NewHandler(pgStore)
	moderationService := moderation.NewService(pgStore)
	moderationService.Start()
	defer moderationService.Stop()
	core.SetModeration(moderationService)
	moderationHandler := moderation.NewHandler(pgStore, moderationService)
	ticketHandler := ticket.NewHandler(pgStore, auditLogger)
	ticketHandler.SetNotifyFunc(notifierService.Notify)
	// refundHandler needs alipayClient which is created below; declared here so
	// the admin route registration further down can reference it.
	var refundHandler *payment.RefundHandler
	rechargeLotteryHandler := rechargelottery.NewHandler(pgStore)
	lotteryHandler := lottery.NewHandler(pgStore)
	lotteryHandler.SetNotifyFunc(notifierService.Notify)
	checkinHandler := checkin.NewHandler(pgStore, cfgHolder)
	growthTaskHandler := task.NewHandler(pgStore)
	referralHandler := referral.NewHandler(pgStore)

	tenantHandler := admin.NewTenantHandler(pgStore, pricingCache)
	tenantHandler.SetAudit(auditLogger)
	tenantHandler.SetOnUpstreamsUpdate(rebuildRouter)
	tenantHandler.SetKeyCache(keyCache)
	subUserHandler := api.NewTenantSubUserHandler(pgStore)
	subUserAuthHandler := api.NewSubUserAuthHandler(pgStore, jwtSecret)

	userPricingHandler := admin.NewUserPricingHandler(pgStore, pricingCache)

	chatHandler := chat.NewHandler(pgStore)

	invoiceHandler := invoice.NewHandler(pgStore, tosClient)
	invoiceAdminHandler := invoice.NewAdminHandler(pgStore, tosClient)

	// Alipay payment client (optional — skip if not configured).
	var paymentHandler *payment.Handler
	var alipayClient *payment.AlipayClient
	if cfg.Payment.Alipay.AppID != "" {
		alipayCfg := payment.AlipayConfig{
			AppID:               cfg.Payment.Alipay.AppID,
			PrivateKeyPath:      cfg.Payment.Alipay.PrivateKeyPath,
			AlipayPublicKeyPath: cfg.Payment.Alipay.AlipayPublicKeyPath,
			NotifyURL:           cfg.Payment.Alipay.NotifyURL,
			ReturnURL:           cfg.Payment.Alipay.ReturnURL,
			QuitURL:             cfg.Payment.Alipay.QuitURL,
			IsProduction:        cfg.Payment.Alipay.IsProduction,
		}
		var err error
		alipayClient, err = payment.NewAlipayClient(alipayCfg)
		if err != nil {
			slog.Warn("failed to create alipay client, payment disabled", "error", err)
		} else {
			paymentHandler = payment.NewHandler(pgStore, alipayClient, cfg.Promotion.FirstRechargeBonusCNY, cfg.Promotion.ReferralInviterBonusCNY, cfg.Promotion.ReferralInviteeBonusCNY)
			slog.Info("alipay payment enabled", "is_production", cfg.Payment.Alipay.IsProduction)
			if cfg.Payment.Alipay.NotifyURL == "" {
				slog.Warn("payment.alipay.notify_url is empty; async notify callbacks will not work until set")
			} else {
				slog.Info("alipay urls (must match open.alipay.com app settings)", "notify_url", cfg.Payment.Alipay.NotifyURL, "return_url", cfg.Payment.Alipay.ReturnURL)
			}
		}
	} else {
		slog.Warn("alipay not configured, payment endpoints disabled")
	}
	refundHandler = payment.NewRefundHandler(pgStore, alipayClient, auditLogger)

	// Codex 代充服务 handlers
	var codexHandler *codex.Handler
	var codexAdminHandler *codex.AdminHandler
	if alipayClient != nil {
		codexHandler = codex.NewHandler(pgStore, alipayClient)
		codexAdminHandler = codex.NewAdminHandler(pgStore, alipayClient)
		slog.Info("codex service enabled")
	} else {
		slog.Warn("codex service disabled (alipay not configured)")
	}

	// Wire cross-references between subscription and payment handlers.
	if paymentHandler != nil {
		subscriptionHandler.SetPaymentCreator(alipayClient)
		paymentHandler.SetSubscribeFunc(func(userID string, planID int) error {
			_, err := subscriptionService.Subscribe(userID, planID)
			return err
		})
		paymentHandler.SetNotifyFunc(notifierService.Notify)
		// Codex 代充与充值共用同一个支付宝 notify URL，按订单号前缀(CDX)分发。
		paymentHandler.SetCodexNotifyFunc(func(orderNo string, callbackData []byte) error {
			return pgStore.MarkCodexOrderPaid(orderNo, callbackData)
		})
	}
	jwtMiddleware := middleware.JWTAuth(jwtSecret)
	adminMiddleware := middleware.BackofficeAccess()
	corsMiddleware := middleware.CORS(cfg.Server.CORSOrigins, cfg.Server.CORSMode)
	securityMiddleware := middleware.SecurityHeaders()
	rateLimitMW, rateLimiter := middleware.RateLimit(middleware.CompositeRateLimitConfig{
		PerIPRPS:        cfg.Server.RateLimit.RequestsPerSecond,
		PerIPBurst:      cfg.Server.RateLimit.Burst,
		PerUserRPS:      cfg.Server.RateLimit.PerUserRPS,
		PerUserBurst:    cfg.Server.RateLimit.PerUserBurst,
		PerKeyRPS:       cfg.Server.RateLimit.PerKeyRPS,
		PerKeyBurst:     cfg.Server.RateLimit.PerKeyBurst,
		CleanupInterval: cfg.Server.RateLimit.CleanupInterval,
	})

	// IP blocker middleware: checks blocked IPs table.
	ipBlocker := middleware.NewIPBlocker(pgStore)

	// Auth-specific rate limiter: 5 requests per minute per IP.
	authLimiter := middleware.NewRateLimiter(5.0/60.0, 5, time.Minute)

	// Register routes.
	mux := http.NewServeMux()

	// Metrics endpoint (no auth required for internal monitoring).
	mux.HandleFunc("/metrics", metrics.Handler())

	// Silently ignore unknown telemetry endpoints.
	mux.HandleFunc("/api/event_logging/batch", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Public endpoints.
	mux.HandleFunc("/health", adminHandler.HandleHealth)
	// OpenAPI spec (public; rendered interactively by the console docs page).
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, filepath.Join("docs", "openapi.yaml"))
	})
	mux.HandleFunc("/v1/models", adminHandler.HandleListModels)
	mux.HandleFunc("/v1/balance", core.ServeBalance)
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Cursor Agent posts Responses API bodies to /v1/chat/completions and
		// expects Responses-shaped SSE/JSON back. Route through the responses
		// handler so both request and response conversion stay symmetric.
		if responses.IsResponsesShapedBody(body) {
			slog.Info("routing Responses API format on /v1/chat/completions through responses handler")
			r.Body = io.NopCloser(bytes.NewReader(body))
			responsesHandler.ServeHTTP(w, r)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		proxyHandler.ServeHTTP(w, r)
	})
	mux.HandleFunc("/v1/images/generations", imageHandler.ServeGPTImageGenerations)
	mux.HandleFunc("/v1/images/edits", imageHandler.ServeGPTImageEdits)
	// Async image task API (API-key auth; reuses the worker pipeline).
	mux.HandleFunc("/v1/images/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			publicTaskHandler.SubmitGenerate(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/v1/images/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// POST /v1/images/tasks/edits — submit an edit task.
		if strings.TrimSuffix(r.URL.Path, "/") == "/v1/images/tasks/edits" {
			if r.Method == http.MethodPost {
				publicTaskHandler.SubmitEdit(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// GET /v1/images/tasks/{id} — poll a task.
		if r.Method == http.MethodGet {
			publicTaskHandler.GetTask(w, r)
			return
		}
		// DELETE /v1/images/tasks/{id} — delete a non-processing task.
		if r.Method == http.MethodDelete {
			publicTaskHandler.DeleteTask(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/v1/messages/count_tokens", anthropicHandler.ServeCountTokens)
	mux.HandleFunc("/v1/messages", anthropicHandler.ServeHTTP)
	mux.HandleFunc("/v1/responses", responsesHandler.ServeHTTP)
	mux.HandleFunc("/v1/responses/", responsesHandler.ServeGetHTTP)
	mux.HandleFunc("/gemini/v1/", geminiHandler.ServeHTTP)
	mux.HandleFunc("/gemini/v1beta/", geminiHandler.ServeHTTP)
	mux.HandleFunc("/v1beta/", geminiHandler.ServeHTTP)

	// Auth endpoints (rate limited).
	authRateLimit := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := httputil.ClientIP(r)
			if !authLimiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":{"message":"too many attempts, please try later","type":"rate_limit_error"}}`, http.StatusTooManyRequests)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/api/login", authRateLimit(authHandler.HandleLogin))
	mux.HandleFunc("/api/logout", authHandler.HandleLogout)

	// SMS and registration endpoints (public, rate limited).
	mux.HandleFunc("/api/sms/send", authRateLimit(authHandler.HandleSendCode))
	mux.HandleFunc("/api/register", authRateLimit(authHandler.HandleRegister))
	mux.HandleFunc("/api/reset-password", authRateLimit(authHandler.HandleResetPassword))
	mux.HandleFunc("/api/system/setup-status", authHandler.HandleSetupStatus)

	// Public landing-page aggregates (no auth, in-memory cached).
	publicHandler := public.New(pgStore)
	mux.HandleFunc("/api/public/plans", publicHandler.HandleListPlans)
	mux.HandleFunc("/api/public/stats", publicHandler.HandleStats)
	mux.HandleFunc("/api/public/models", publicHandler.HandleListModels)
	mux.HandleFunc("/api/public/pricing", publicHandler.HandleListPricing)

	// Alipay notify endpoint (public — called by Alipay servers, no JWT).
	if paymentHandler != nil {
		mux.HandleFunc("/api/payment/alipay/notify", paymentHandler.HandleAlipayNotify)
	}

	// Codex 公开接口（无需认证）
	if codexHandler != nil {
		mux.HandleFunc("/api/codex/products", codexHandler.HandleListProducts)
		mux.HandleFunc("/api/codex/orders/create", codexHandler.HandleCreateOrder)
		mux.HandleFunc("/api/codex/orders/", codexHandler.HandleGetOrder)
		mux.HandleFunc("/api/codex/payment/alipay/notify", codexHandler.HandleAlipayNotify)
	}

	// Sub-user auth endpoints (public — sub-users have their own login flow).
	mux.HandleFunc("/api/sub-user/login", subUserAuthHandler.HandleSubUserLogin)
	mux.HandleFunc("/api/sub-user/logout", subUserAuthHandler.HandleSubUserLogout)

	// Protected API endpoints (JWT required).
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/me", authHandler.HandleMe)
	apiMux.HandleFunc("/api/sub-user/me", subUserAuthHandler.HandleSubUserMe)
	apiMux.HandleFunc("/api/sub-user/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			subUserAuthHandler.HandleSubUserListKeys(w, r)
		case http.MethodPost:
			subUserAuthHandler.HandleSubUserCreateKey(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/sub-user/keys/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			subUserAuthHandler.HandleSubUserDeleteKey(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/sub-user/transactions", subUserAuthHandler.HandleSubUserTransactions)
	apiMux.HandleFunc("/api/sub-user/stats", subUserAuthHandler.HandleSubUserStats)
	apiMux.HandleFunc("/api/promotion/rules", authHandler.HandlePromotionRules)
	// OpenAI-compatible models list endpoint (for sub-users and regular users with JWT)
	apiMux.HandleFunc("/api/v1/models", adminHandler.HandleListModels)
	// Image-share session model list (filtered to image-generation models, role=image_share only)
	apiMux.HandleFunc("/api/image/models", adminHandler.HandleListImageShareModels)
	apiMux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			modelHandler.HandleList(w, r)
		case http.MethodPost:
			modelHandler.HandleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/models/test", admin.HandleTestUpstream)
	apiMux.HandleFunc("/api/models/list-remote", admin.HandleListRemoteModels)
	apiMux.HandleFunc("/api/models/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			modelHandler.HandleUpdate(w, r)
		case http.MethodDelete:
			modelHandler.HandleDelete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/status", adminHandler.HandleGetStatusNoAuth)
	apiMux.HandleFunc("/api/config", adminHandler.HandleGetConfigNoAuth)
	apiMux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			apikeyHandler.HandleList(w, r)
		case http.MethodPost:
			apikeyHandler.HandleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/keys/revoke-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apikeyHandler.HandleRevokeAll(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/keys/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			apikeyHandler.HandleRevoke(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Billing endpoints (JWT required).
	apiMux.HandleFunc("/api/billing/balance", billingHandler.HandleGetBalance)
	apiMux.HandleFunc("/api/billing/transactions", billingHandler.HandleListTransactions)
	apiMux.HandleFunc("/api/billing/transactions/export", billingHandler.HandleExportTransactions)
	apiMux.HandleFunc("/api/billing/stats", billingHandler.HandleGetStats)
	apiMux.HandleFunc("/api/billing/token-stats", billingHandler.HandleGetTokenStats)
	apiMux.HandleFunc("/api/billing/key-usage", billingHandler.HandleGetKeyUsage)
	apiMux.HandleFunc("/api/billing/key-usage/{keyID}/transactions", billingHandler.HandleGetKeyTransactions)

	// Recharge lottery user endpoint (public, JWT required).
	apiMux.HandleFunc("GET /api/recharge-lottery", rechargeLotteryHandler.HandlePublicGet)
	apiMux.HandleFunc("GET /api/recharge-lottery/rounds", rechargeLotteryHandler.HandlePublicRounds)

	// Lottery user endpoints (JWT required).
	apiMux.HandleFunc("GET /api/lottery/current", lotteryHandler.HandleUserGetCurrent)
	apiMux.HandleFunc("GET /api/lottery/records", lotteryHandler.HandleUserGetRecords)
	apiMux.HandleFunc("GET /api/lottery/my-records", lotteryHandler.HandleUserGetMyRecords)

	// Daily check-in endpoints (JWT required).
	apiMux.HandleFunc("/api/checkin", checkinHandler.HandleCheckin)
	apiMux.HandleFunc("/api/checkin/status", checkinHandler.HandleStatus)

	// Growth task endpoints (JWT required).
	apiMux.HandleFunc("/api/tasks", growthTaskHandler.HandleList)
	apiMux.HandleFunc("/api/tasks/", growthTaskHandler.HandleClaim)

	// Referral endpoint (JWT required).
	apiMux.HandleFunc("/api/referral", referralHandler.HandleInfo)

	// Subscription endpoints (JWT required).
	apiMux.HandleFunc("/api/subscription/plans", subscriptionHandler.HandleListPlans)
	apiMux.HandleFunc("/api/subscription/current", subscriptionHandler.HandleGetCurrent)
	apiMux.HandleFunc("/api/subscription/history", subscriptionHandler.HandleListHistory)
	apiMux.HandleFunc("/api/subscription/usage", subscriptionHandler.HandleGetUsage)
	apiMux.HandleFunc("/api/subscription/subscribe", subscriptionHandler.HandleSubscribe)
	apiMux.HandleFunc("/api/subscription/create-payment", subscriptionHandler.HandleCreateSubscriptionPayment)

	// Payment endpoints (JWT required).
	if paymentHandler != nil {
		apiMux.HandleFunc("/api/payment/create", paymentHandler.HandleCreate)
		apiMux.HandleFunc("/api/payment/repay", paymentHandler.HandleRepay)
		apiMux.HandleFunc("/api/payment/orders", paymentHandler.HandleListOrders)
	}

	// Playground proxy (JWT required, uses key_id for ownership verification).
	apiMux.HandleFunc("/api/playground/completions", playgroundHandler.ServeHTTP)

	// Chat session endpoints (JWT required).
	apiMux.HandleFunc("/api/chat/sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			chatHandler.HandleListSessions(w, r)
		case http.MethodPost:
			chatHandler.HandleCreateSession(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/chat/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/messages"):
			switch r.Method {
			case http.MethodGet:
				chatHandler.HandleGetMessages(w, r)
			case http.MethodPost:
				chatHandler.HandleAddMessage(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			switch r.Method {
			case http.MethodPut:
				chatHandler.HandleUpdateSession(w, r)
			case http.MethodDelete:
				chatHandler.HandleDeleteSession(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})

	// Public pricing endpoint (JWT required).
	apiMux.HandleFunc("/api/pricing", pricingHandler.HandleListActivePricing)

	// Published announcements endpoint (JWT required).
	apiMux.HandleFunc("/api/announcements", announcementHandler.HandleListPublished)

	// Notification endpoints (JWT required).
	apiMux.HandleFunc("/api/notifications/unread-count", notificationHandler.HandleUnreadCount)
	apiMux.HandleFunc("/api/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			notificationHandler.HandleMarkAllRead(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/read") && r.Method == http.MethodPut {
			notificationHandler.HandleMarkRead(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/notifications", notificationHandler.HandleList)

	// Support tickets (user side).
	apiMux.HandleFunc("/api/tickets", ticketHandler.HandleTickets)
	apiMux.HandleFunc("/api/tickets/", ticketHandler.HandleTicketByID)

	// Notification channel preferences.
	apiMux.HandleFunc("/api/notification/preferences", notifierHandler.HandlePreferences)


	// Image task endpoints (JWT required).
	apiMux.HandleFunc("/api/image/tasks/edit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskHandler.SubmitEdit(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/image/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// /api/image/tasks/{id}/images — single image deletion
		if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/images") {
			if r.Method == http.MethodDelete {
				taskHandler.DeleteResultImage(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// /api/image/tasks/{id}
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetTask(w, r)
		case http.MethodDelete:
			taskHandler.DeleteTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/image/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.ListTasks(w, r)
		case http.MethodPost:
			taskHandler.SubmitGenerate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Image-share parent-user CRUD (JWT required + image_share_enabled=true).
	apiMux.HandleFunc("/api/image-share/keys", imageShareKeys.HandleKeysRouter)
	apiMux.HandleFunc("/api/image-share/keys/", imageShareKeys.HandleKeysRouter)
	apiMux.HandleFunc("/api/image-share/me", imageShareAuth.HandleMe)
	apiMux.HandleFunc("/api/image-share/logout", imageShareAuth.HandleLogout)

	// PPT task endpoints (JWT required).
	apiMux.HandleFunc("/api/ppt/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/confirm") && r.Method == http.MethodPost {
			pptHandler.ConfirmOutline(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/presentation") && r.Method == http.MethodPut {
			pptHandler.UpdatePresentation(w, r)
		} else if r.Method == http.MethodGet {
			pptHandler.GetTask(w, r)
		} else if r.Method == http.MethodDelete {
			pptHandler.DeleteTask(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/ppt/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pptHandler.ListTasks(w, r)
		case http.MethodPost:
			pptHandler.SubmitTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Invoice title endpoints (JWT required).
	apiMux.HandleFunc("/api/invoice/titles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			invoiceHandler.HandleListTitles(w, r)
		case http.MethodPost:
			invoiceHandler.HandleCreateTitle(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/invoice/titles/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/default") && r.Method == http.MethodPut:
			invoiceHandler.HandleSetDefaultTitle(w, r)
		case r.Method == http.MethodPut:
			invoiceHandler.HandleUpdateTitle(w, r)
		case r.Method == http.MethodDelete:
			invoiceHandler.HandleDeleteTitle(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/invoice/company/search", invoiceHandler.HandleCompanySearch)
	apiMux.HandleFunc("/api/invoice/available-orders", invoiceHandler.HandleAvailableOrders)
	apiMux.HandleFunc("/api/invoice/requests", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			invoiceHandler.HandleListRequests(w, r)
		case http.MethodPost:
			invoiceHandler.HandleCreateRequest(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/invoice/requests/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/cancel") && r.Method == http.MethodPut:
			invoiceHandler.HandleCancelRequest(w, r)
		case strings.HasSuffix(path, "/download") && r.Method == http.MethodGet:
			invoiceHandler.HandleDownload(w, r)
		case r.Method == http.MethodGet:
			invoiceHandler.HandleGetRequest(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Admin endpoints (JWT + admin role required).
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/api/admin/dashboard", userHandler.HandleDashboard)
	adminMux.HandleFunc("/api/admin/alert/rules", alertingHandler.HandleRules)
	adminMux.HandleFunc("/api/admin/alert/events", alertingHandler.HandleEvents)
	adminMux.HandleFunc("/api/admin/moderation/settings", moderationHandler.HandleSettings)
	adminMux.HandleFunc("/api/admin/moderation/keywords", moderationHandler.HandleKeywords)
	adminMux.HandleFunc("/api/admin/moderation/keywords/", moderationHandler.HandleDeleteKeyword)
	adminMux.HandleFunc("/api/admin/moderation/hits", moderationHandler.HandleHits)
	adminMux.HandleFunc("/api/admin/tickets", ticketHandler.HandleAdminTickets)
	adminMux.HandleFunc("/api/admin/tickets/", ticketHandler.HandleAdminTicketByID)
	adminMux.HandleFunc("/api/admin/referral/rules", referralHandler.HandleAdminRules)
	adminMux.HandleFunc("/api/admin/audit-logs", auditHandler.HandleList)
	adminMux.HandleFunc("/api/admin/upstreams/test", adminHandler.HandleTestUpstreamByName)
	adminMux.HandleFunc("/api/admin/consumption-stats", userHandler.HandleConsumptionStats)
	adminMux.HandleFunc("/api/admin/funnel-stats", userHandler.HandleFunnelStats)
	adminMux.HandleFunc("/api/admin/image-duration-stats", userHandler.HandleImageDurationStats)
	adminMux.HandleFunc("/api/admin/orders", userHandler.HandleListAllOrders)
	adminMux.HandleFunc("/api/admin/orders/", refundHandler.HandleCreateRefund) // POST {orderNo}/refund
	adminMux.HandleFunc("/api/admin/refunds", refundHandler.HandleListRefunds)
	adminMux.HandleFunc("/api/admin/pricing", pricingHandler.HandleListPricing)
	adminMux.HandleFunc("/api/admin/pricing/change-logs", pricingHandler.HandleListPricingChangeLogs)
	adminMux.HandleFunc("/api/admin/pricing/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			pricingHandler.HandleUpsertPricing(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.HandleListUsers(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/status") && r.Method == http.MethodPut:
			userHandler.HandleUpdateUserStatus(w, r)
		case strings.HasSuffix(path, "/role") && r.Method == http.MethodPut:
			// Role assignment is admin-only even though support can reach
			// /api/admin/users via the RBAC matrix.
			if role, _ := r.Context().Value(admin.CtxRoleKey).(string); role != "admin" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			userHandler.HandleUpdateUserRole(w, r)
		case strings.HasSuffix(path, "/recharge") && r.Method == http.MethodPost:
			userHandler.HandleRecharge(w, r)
		case strings.HasSuffix(path, "/revoke-keys") && r.Method == http.MethodPost:
			apikeyHandler.HandleAdminRevokeAllKeys(w, r)
		case strings.HasSuffix(path, "/image-share") && r.Method == http.MethodPatch:
			imageShareAdmin.HandleToggle(w, r)
		case strings.HasSuffix(path, "/transactions/export") && r.Method == http.MethodGet:
			userHandler.HandleExportUserTransactions(w, r)
		case strings.HasSuffix(path, "/transactions") && r.Method == http.MethodGet:
			userHandler.HandleUserTransactions(w, r)
		case strings.HasSuffix(path, "/consumption-stats") && r.Method == http.MethodGet:
			userHandler.HandleUserConsumptionStats(w, r)
		case strings.Contains(path, "/pricing/") && r.Method == http.MethodPut:
			userPricingHandler.HandleUpsertUserPricing(w, r)
		case strings.Contains(path, "/pricing/") && r.Method == http.MethodDelete:
			userPricingHandler.HandleDeleteUserPricing(w, r)
		case strings.HasSuffix(path, "/pricing") && r.Method == http.MethodGet:
			userPricingHandler.HandleListUserPricing(w, r)
		case r.Method == http.MethodDelete:
			userHandler.HandleDeleteUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Announcement admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/announcements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			announcementHandler.HandleList(w, r)
		case http.MethodPost:
			announcementHandler.HandleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/announcements/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			announcementHandler.HandleUpdate(w, r)
		case http.MethodDelete:
			announcementHandler.HandleDelete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Subscription admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/subscription-orders/stats", subscriptionHandler.HandleAdminSubscriptionOrderStats)
	adminMux.HandleFunc("/api/admin/subscription-orders", subscriptionHandler.HandleAdminListSubscriptionOrders)
	adminMux.HandleFunc("/api/admin/subscriptions", subscriptionHandler.HandleAdminListSubscriptions)
	adminMux.HandleFunc("/api/admin/subscriptions/grant", subscriptionHandler.HandleAdminGrantSubscription)
	adminMux.HandleFunc("/api/admin/subscription-users-usage", subscriptionHandler.HandleAdminSubscriptionUsersUsage)
	adminMux.HandleFunc("/api/admin/subscription-plans", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			subscriptionHandler.HandleAdminListPlans(w, r)
		case http.MethodPost:
			subscriptionHandler.HandleAdminCreatePlan(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/subscription-plans/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			subscriptionHandler.HandleAdminUpdatePlan(w, r)
		case http.MethodDelete:
			subscriptionHandler.HandleAdminDeletePlan(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Invoice admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/invoice/requests", invoiceAdminHandler.HandleAdminListRequests)
	adminMux.HandleFunc("/api/admin/invoice/requests/batch-approve", invoiceAdminHandler.HandleAdminBatchApprove)
	adminMux.HandleFunc("/api/admin/invoice/requests/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/process") && r.Method == http.MethodPut:
			invoiceAdminHandler.HandleAdminProcess(w, r)
		case strings.HasSuffix(path, "/complete") && r.Method == http.MethodPut:
			invoiceAdminHandler.HandleAdminComplete(w, r)
		case strings.HasSuffix(path, "/reject") && r.Method == http.MethodPut:
			invoiceAdminHandler.HandleAdminReject(w, r)
		case r.Method == http.MethodGet:
			invoiceAdminHandler.HandleAdminGetRequest(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Recharge promotion admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/recharge-promotions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rechargePromotionHandler.HandleList(w, r)
		case http.MethodPost:
			rechargePromotionHandler.HandleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/recharge-promotions/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			rechargePromotionHandler.HandleUpdate(w, r)
		case http.MethodDelete:
			rechargePromotionHandler.HandleDelete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Recharge lottery admin endpoints.
	adminMux.HandleFunc("/api/admin/recharge-lottery", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rechargeLotteryHandler.HandleAdminGet(w, r)
		case http.MethodPost:
			rechargeLotteryHandler.HandleAdminCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/recharge-lottery/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/rounds") && r.Method == http.MethodGet {
			rechargeLotteryHandler.HandleAdminRounds(w, r)
			return
		}
		if r.Method == http.MethodPut {
			rechargeLotteryHandler.HandleAdminUpdate(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Lottery admin endpoints.
	adminMux.HandleFunc("/api/admin/lottery/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			lotteryHandler.HandleAdminListEvents(w, r)
		case http.MethodPost:
			lotteryHandler.HandleAdminCreateEvent(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/lottery/events/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/prizes") {
			switch r.Method {
			case http.MethodGet:
				lotteryHandler.HandleAdminListPrizes(w, r)
			case http.MethodPost:
				lotteryHandler.HandleAdminCreatePrize(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		if strings.HasSuffix(r.URL.Path, "/records") && r.Method == http.MethodGet {
			lotteryHandler.HandleAdminListRecords(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/draw") && r.Method == http.MethodPost {
			lotteryHandler.HandleAdminDrawEvent(w, r)
			return
		}
		if r.Method == http.MethodPut {
			lotteryHandler.HandleAdminUpdateEvent(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	adminMux.HandleFunc("/api/admin/lottery/prizes/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			lotteryHandler.HandleAdminUpdatePrize(w, r)
		case http.MethodDelete:
			lotteryHandler.HandleAdminDeletePrize(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// IP blocking admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/blocked-ips", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			admin.ListBlockedIPsHandler(pgStore)(w, r)
		case http.MethodPost:
			admin.BlockIPHandler(pgStore)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/blocked-ips/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			admin.UnblockIPHandler(pgStore)(w, r)
		} else if r.Method == http.MethodGet {
			admin.GetBlockedIPHandler(pgStore)(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Tenant admin endpoints (JWT + admin role required).
	adminMux.HandleFunc("/api/admin/tenants", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tenantHandler.HandleAdminListTenants(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/tenants/enterprise", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			tenantHandler.HandleCreateEnterpriseTenant(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	adminMux.HandleFunc("/api/admin/tenants/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/enterprise-info") && r.Method == http.MethodPut:
			tenantHandler.HandleUpdateEnterpriseInfo(w, r)
		case strings.HasSuffix(path, "/recharge") && r.Method == http.MethodPost:
			tenantHandler.HandleRechargeTenant(w, r)
		case strings.HasSuffix(path, "/balance") && r.Method == http.MethodGet:
			tenantHandler.HandleGetTenantBalance(w, r)
		case strings.HasSuffix(path, "/transactions") && r.Method == http.MethodGet:
			tenantHandler.HandleListTenantTransactions(w, r)
		case strings.Contains(path, "/pricing/") && r.Method == http.MethodPut:
			tenantHandler.HandleUpsertTenantPricing(w, r)
		case strings.Contains(path, "/pricing/") && r.Method == http.MethodDelete:
			tenantHandler.HandleDeleteTenantPricing(w, r)
		case strings.HasSuffix(path, "/pricing") && r.Method == http.MethodGet:
			tenantHandler.HandleListTenantPricing(w, r)
		case strings.Contains(path, "/model-upstreams/") && r.Method == http.MethodPut:
			tenantHandler.HandleReplaceTenantModelUpstreams(w, r)
		case strings.Contains(path, "/model-upstreams/") && r.Method == http.MethodDelete:
			tenantHandler.HandleDeleteTenantModelUpstreams(w, r)
		case strings.HasSuffix(path, "/model-upstreams") && r.Method == http.MethodGet:
			tenantHandler.HandleListTenantModelUpstreams(w, r)
		case r.Method == http.MethodGet:
			tenantHandler.HandleGetTenant(w, r)
		case r.Method == http.MethodDelete:
			tenantHandler.HandleDeleteTenant(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Codex 管理后台接口（需要 admin 权限）
	if codexAdminHandler != nil {
		adminMux.HandleFunc("/api/admin/codex/orders", codexAdminHandler.HandleListOrders)
		adminMux.HandleFunc("/api/admin/codex/orders/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			switch {
			case strings.HasSuffix(path, "/ship") && r.Method == http.MethodPost:
				codexAdminHandler.HandleShipOrder(w, r)
			case strings.HasSuffix(path, "/refund") && r.Method == http.MethodPost:
				codexAdminHandler.HandleRefundOrder(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
		// Codex 商品管理
		adminMux.HandleFunc("/api/admin/codex/products", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				codexAdminHandler.HandleListProducts(w, r)
			case http.MethodPost:
				codexAdminHandler.HandleCreateProduct(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
		adminMux.HandleFunc("/api/admin/codex/products/", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				codexAdminHandler.HandleUpdateProduct(w, r)
			case http.MethodDelete:
				codexAdminHandler.HandleDeleteProduct(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	// Tenant endpoints (JWT required, role-based access within handlers).
	tenantMemberMW := middleware.TenantRoleRequired(pgStore)
	tenantAdminMW := middleware.TenantRoleRequired(pgStore, "owner", "admin")
	tenantOwnerMW := middleware.TenantRoleRequired(pgStore, "owner")

	apiMux.HandleFunc("/api/tenants", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tenantHandler.HandleListTenants(w, r)
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"tenant creation is admin-only","code":"forbidden"}`))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/invitations/pending", tenantHandler.HandlePendingInvitations)
	apiMux.HandleFunc("/api/invitations/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/accept") && r.Method == http.MethodPost {
			tenantHandler.HandleAcceptInvitation(w, r)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
	apiMux.Handle("/api/tenants/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// Balance (any member)
		case strings.HasSuffix(path, "/balance") && r.Method == http.MethodGet:
			tenantMemberMW(http.HandlerFunc(tenantHandler.HandleGetTenantBalance)).ServeHTTP(w, r)
		// Transactions (owner/admin)
		case strings.HasSuffix(path, "/transactions") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleListTenantTransactions)).ServeHTTP(w, r)
		// Audit log (owner/admin)
		case strings.HasSuffix(path, "/audit") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(auditHandler.HandleTenantList)).ServeHTTP(w, r)
		// Members list (any member)
		case strings.HasSuffix(path, "/members") && r.Method == http.MethodGet:
			tenantMemberMW(http.HandlerFunc(tenantHandler.HandleListMembers)).ServeHTTP(w, r)
		// Invite member (owner/admin)
		case strings.HasSuffix(path, "/members/invite") && r.Method == http.MethodPost:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleInviteMember)).ServeHTTP(w, r)
		// Update member role (owner only)
		case strings.Contains(path, "/members/") && strings.HasSuffix(path, "/role") && r.Method == http.MethodPut:
			tenantOwnerMW(http.HandlerFunc(tenantHandler.HandleUpdateMemberRole)).ServeHTTP(w, r)
		// Remove member (owner/admin)
		case strings.Contains(path, "/members/") && r.Method == http.MethodDelete:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleRemoveMember)).ServeHTTP(w, r)
		// Transfer ownership (owner only)
		case strings.HasSuffix(path, "/transfer") && r.Method == http.MethodPost:
			tenantOwnerMW(http.HandlerFunc(tenantHandler.HandleTransferOwnership)).ServeHTTP(w, r)
		// Keys list (owner/admin)
		case strings.HasSuffix(path, "/keys") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleListTenantKeys)).ServeHTTP(w, r)
		// Create key (owner/admin)
		case strings.HasSuffix(path, "/keys") && r.Method == http.MethodPost:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleCreateTenantKey)).ServeHTTP(w, r)
		// Delete key (owner/admin)
		case strings.Contains(path, "/keys/") && r.Method == http.MethodDelete:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleDeleteTenantKey)).ServeHTTP(w, r)
		// Invitations list (owner/admin)
		case strings.HasSuffix(path, "/invitations") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleListInvitations)).ServeHTTP(w, r)
		// Delete invitation (owner/admin)
		case strings.Contains(path, "/invitations/") && r.Method == http.MethodDelete:
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleDeleteInvitation)).ServeHTTP(w, r)
		// Sub-users list (owner/admin)
		case strings.HasSuffix(path, "/sub-users") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleListSubUsers)).ServeHTTP(w, r)
		// Create sub-user (owner/admin)
		case strings.HasSuffix(path, "/sub-users") && r.Method == http.MethodPost:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleCreateSubUser)).ServeHTTP(w, r)
		// All sub-users transactions (owner/admin)
		case strings.HasSuffix(path, "/all-transactions") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleListAllSubUserTransactions)).ServeHTTP(w, r)
		// Tenant stats (owner/admin)
		case strings.HasSuffix(path, "/stats") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleTenantStats)).ServeHTTP(w, r)
		// Export transactions (owner/admin)
		case strings.HasSuffix(path, "/export-transactions") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleExportTransactions)).ServeHTTP(w, r)
		// Sub-user detail (owner/admin)
		case strings.Contains(path, "/sub-users/") && !strings.Contains(path[strings.Index(path, "/sub-users/")+11:], "/") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleGetSubUser)).ServeHTTP(w, r)
		// Delete sub-user (owner only)
		case strings.Contains(path, "/sub-users/") && !strings.Contains(path[strings.Index(path, "/sub-users/")+11:], "/") && r.Method == http.MethodDelete:
			tenantOwnerMW(http.HandlerFunc(subUserHandler.HandleDeleteSubUser)).ServeHTTP(w, r)
		// Update sub-user quota (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/quota") && r.Method == http.MethodPut:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleUpdateSubUserQuota)).ServeHTTP(w, r)
		// Reset sub-user password (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/password") && r.Method == http.MethodPut:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleResetSubUserPassword)).ServeHTTP(w, r)
		// Sub-user keys list (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/keys") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleListSubUserKeys)).ServeHTTP(w, r)
		// Create sub-user key (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/keys") && r.Method == http.MethodPost:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleCreateSubUserKey)).ServeHTTP(w, r)
		// Delete sub-user key (owner/admin)
		case strings.Contains(path, "/keys/") && r.Method == http.MethodDelete && strings.Contains(path, "/sub-users/"):
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleDeleteSubUserKey)).ServeHTTP(w, r)
		// Sub-user transactions (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/transactions") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleListSubUserTransactions)).ServeHTTP(w, r)
		// Sub-user model stats (owner/admin)
		case strings.Contains(path, "/sub-users/") && strings.HasSuffix(path, "/model-stats") && r.Method == http.MethodGet:
			tenantAdminMW(http.HandlerFunc(subUserHandler.HandleGetSubUserModelStats)).ServeHTTP(w, r)
		// Recharge (platform admin only)
		case strings.HasSuffix(path, "/recharge") && r.Method == http.MethodPost:
			tenantOwnerMW(http.HandlerFunc(tenantHandler.HandleRechargeTenant)).ServeHTTP(w, r)
		// Get tenant detail (any member)
		case r.Method == http.MethodGet && !strings.Contains(path[len("/api/tenants/")+36:], "/"):
			tenantMemberMW(http.HandlerFunc(tenantHandler.HandleGetTenant)).ServeHTTP(w, r)
		// Update tenant (owner/admin)
		case r.Method == http.MethodPut && !strings.Contains(path[len("/api/tenants/")+36:], "/"):
			tenantAdminMW(http.HandlerFunc(tenantHandler.HandleUpdateTenant)).ServeHTTP(w, r)
		// Delete tenant (owner only)
		case r.Method == http.MethodDelete && !strings.Contains(path[len("/api/tenants/")+36:], "/"):
			tenantOwnerMW(http.HandlerFunc(tenantHandler.HandleDeleteTenant)).ServeHTTP(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))

	apiMux.Handle("/api/admin/", adminMiddleware(adminMux))

	mux.Handle("/api/", jwtMiddleware(apiMux))

	// Public image-share login (no JWT). Mounted on the root mux so it bypasses jwtMiddleware.
	mux.HandleFunc("/api/image-share/login", imageShareAuth.HandleLogin)

	// Legacy admin endpoints (keep for backward compat).
	mux.HandleFunc("/admin/config", func(w http.ResponseWriter, r *http.Request) {
		adminHandler.HandleGetConfig(w, r)
	})
	mux.HandleFunc("/admin/status", adminHandler.HandleGetStatus)

	handler := corsMiddleware(securityMiddleware(ipBlocker(rateLimitMW(middleware.AccessLog(middleware.Recovery(middleware.RequestID(mux)))))))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())

	// Cleanup expired IP blocks every hour.
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if deleted, err := pgStore.CleanupExpiredBlockedIPs(); err != nil {
					slog.Error("Failed to cleanup expired blocked IPs", "error", err)
				} else if deleted > 0 {
					slog.Info("Cleaned up expired blocked IPs", "count", deleted)
				}
			case <-bgCtx.Done():
				return
			}
		}
	}()

	// Retention background jobs: weekly usage report + silent-user winback.
	retentionService := retention.New(pgStore, cfgHolder)
	retentionService.Start()
	defer retentionService.Stop()

	// Ops alerting: watches metric deltas + DB health against alert_rules.
	alertingService := alerting.New(pgStore, pgStore.DB(), cfgHolder, smsSender)
	alertingService.Start()
	defer alertingService.Stop()

	// Order expiry goroutine: expire pending orders every minute.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n, err := pgStore.ExpireOrders(); err != nil {
					slog.Error("failed to expire orders", "error", err)
				} else if n > 0 {
					slog.Info("expired pending orders", "count", n)
				}
				// 一并清理过期的 Codex 代充订单（CDX 前缀）。
				if cn, err := pgStore.ExpireCodexOrders(); err != nil {
					slog.Error("failed to expire codex orders", "error", err)
				} else if cn > 0 {
					slog.Info("expired codex orders", "count", cn)
				}
			case <-bgCtx.Done():
				return
			}
		}
	}()

	// Subscription expiry goroutine: expire subscriptions every minute.
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n, err := pgStore.ExpireExpiredSubscriptions(); err != nil {
					slog.Error("failed to expire subscriptions", "error", err)
				} else if n > 0 {
					slog.Info("expired subscriptions", "count", n)
				}
			case <-bgCtx.Done():
				return
			}
		}
	}()

	// SIGHUP: reload from DB.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)
		for {
			select {
			case <-sigCh:
				slog.Info("received SIGHUP, rebuilding router from DB")
				rebuildRouter()
			case <-bgCtx.Done():
				return
			}
		}
	}()

	// SIGINT/SIGTERM: graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down server")
		bgCancel()
		imageService.StopWorkers()
		pptService.Stop()
		if rateLimiter != nil {
			rateLimiter.Stop()
		}
		keyCache.Stop()
		smsCodeStore.Stop()
		emailCodeStore.Stop()
		pricingCache.Stop()
		shutdownTimeout := cfgHolder.Get().Server.ShutdownTimeout
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}
	}()

	slog.Info("starting llm-gateway", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// parseLogLevel converts a string log level to slog.Level.
// Defaults to INFO if the string is empty or unrecognized.
func parseLogLevel(s string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return level
}
