// Package main is the entry point for the digital-twin-community backend server.
// It wires together all modules and starts the HTTP server + async workers.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	qdrantpb "github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/digital-twin-community/backend/internal/agent"
	agentdb "github.com/digital-twin-community/backend/internal/agent/db"
	"github.com/digital-twin-community/backend/internal/api"
	"github.com/digital-twin-community/backend/internal/auth"
	authdb "github.com/digital-twin-community/backend/internal/auth/db"
	"github.com/digital-twin-community/backend/internal/config"
	"github.com/digital-twin-community/backend/internal/connection"
	connectiondb "github.com/digital-twin-community/backend/internal/connection/db"
	"github.com/digital-twin-community/backend/internal/discussion"
	discussiondb "github.com/digital-twin-community/backend/internal/discussion/db"
	"github.com/digital-twin-community/backend/internal/embedding"
	"github.com/digital-twin-community/backend/internal/llm"
	"github.com/digital-twin-community/backend/internal/matching"
	apimiddleware "github.com/digital-twin-community/backend/internal/middleware"
	"github.com/digital-twin-community/backend/internal/notification"
	notificationdb "github.com/digital-twin-community/backend/internal/notification/db"
	"github.com/digital-twin-community/backend/internal/report"
	reportdb "github.com/digital-twin-community/backend/internal/report/db"
	"github.com/digital-twin-community/backend/internal/scheduler"
	"github.com/digital-twin-community/backend/internal/sender"
	"github.com/digital-twin-community/backend/internal/topic"
	topicdb "github.com/digital-twin-community/backend/internal/topic/db"
	"github.com/digital-twin-community/backend/internal/worker"
)

func main() {
	// ─── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		os.Exit(1)
	}

	// ─── Logger ───────────────────────────────────────────────────────────────
	var logger *zap.Logger
	if cfg.App.Env == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// ─── PostgreSQL ───────────────────────────────────────────────────────────
	dbPool, err := pgxpool.New(context.Background(), cfg.DB.DSN)
	if err != nil {
		logger.Fatal("connect to postgres", zap.Error(err))
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		logger.Fatal("ping postgres", zap.Error(err))
	}
	logger.Info("connected to PostgreSQL")

	// ─── Redis ────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("connect to redis", zap.Error(err))
	}
	logger.Info("connected to Redis")

	// ─── Asynq Client + Server ────────────────────────────────────────────────
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 20,
		Queues: map[string]int{
			scheduler.QueueDiscussionHigh: 10,
			scheduler.QueueLLMStandard:    20,
			scheduler.QueueNotification:   50,
			scheduler.QueueReport:         5,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			logger.Error("asynq task failed",
				zap.String("type", task.Type()),
				zap.Error(err),
			)
		}),
	})

	// ─── LLM Gateway ──────────────────────────────────────────────────────────
	llmGateway := llm.NewGateway(&cfg.LLM, rdb, logger)

	// ─── Qdrant ───────────────────────────────────────────────────────────────
	qdrantConn, err := newQdrantConn(&cfg.Qdrant)
	if err != nil {
		logger.Fatal("qdrant client init", zap.Error(err))
	}
	defer qdrantConn.Close()
	qdrantPointsClient := qdrantpb.NewPointsClient(qdrantConn)
	qdrantCollectionsClient := qdrantpb.NewCollectionsClient(qdrantConn)
	logger.Info("qdrant client ready",
		zap.String("host", cfg.Qdrant.Host),
		zap.Int("port", cfg.Qdrant.Port),
	)

	// ─── Embedding Service ─────────────────────────────────────────────────────
	embeddingSvc := embedding.NewService(qdrantPointsClient, qdrantCollectionsClient, llmGateway, cfg.Qdrant.Collections.Agents, cfg.Qdrant.Collections.Topics, logger)

	// Ensure Qdrant collections exist (idempotent, runs on every startup).
	initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := embeddingSvc.EnsureCollections(initCtx, []string{
		cfg.Qdrant.Collections.Agents,
		cfg.Qdrant.Collections.Topics,
	}); err != nil {
		initCancel()
		logger.Fatal("ensure qdrant collections", zap.Error(err))
	}
	initCancel()
	logger.Info("qdrant collections ready")

	// ─── Matching ─────────────────────────────────────────────────────────────
	matcher := matching.NewMatcher(logger)

	// ─── Discussion Engine ────────────────────────────────────────────────────
	discussionEngine := discussion.NewEngine(llmGateway, logger)

	// ─── Report Generator ─────────────────────────────────────────────────────
	reportGen := report.NewGenerator(llmGateway, logger, report.DefaultConfig())

	// ─── Repositories ─────────────────────────────────────────────────────────
	authRepo := authdb.NewRepository(dbPool)
	agentRepo := agentdb.NewRepository(dbPool)
	topicRepo := topicdb.NewRepository(dbPool)
	schedTopicRepo := topicdb.NewSchedulerRepository(dbPool)
	discussionRepo := discussiondb.NewRepository(dbPool)
	reportRepo := reportdb.NewRepository(dbPool)
	connRepo := connectiondb.NewRepository(dbPool)
	connAgentRepo := connectiondb.NewAgentRepository(dbPool)
	notifRepo := notificationdb.NewRepository(dbPool)

	// ─── FCM Sender ───────────────────────────────────────────────────────────
	var fcmSender notification.FCMSender
	fcmClient, err := sender.NewFCMClient(cfg.Firebase.CredentialsFile, logger)
	if err != nil {
		logger.Warn("FCM not configured – using no-op sender (push notifications disabled)",
			zap.Error(err))
		fcmSender = sender.NewNoOpFCMSender(logger)
	} else {
		fcmSender = fcmClient
	}

	// ─── Email Sender ─────────────────────────────────────────────────────────
	emailSender := sender.NewSendGridClient(
		cfg.SendGrid.APIKey,
		cfg.SendGrid.FromEmail,
		cfg.SendGrid.FromName,
		logger,
	)

	// ─── Notification Service ─────────────────────────────────────────────────
	notifSvc := notification.NewService(fcmSender, emailSender, notifRepo, notifRepo, logger)

	// ─── Auth Service ─────────────────────────────────────────────────────────
	authSvc := auth.NewService(authRepo, &cfg.JWT, logger)

	// ─── Agent Service ────────────────────────────────────────────────────────
	agentSvc := agent.NewService(agentRepo, logger)

	// ─── Topic Service ────────────────────────────────────────────────────────
	topicSvc := topic.NewService(topicRepo, logger)

	// ─── Connection Service ───────────────────────────────────────────────────
	// Use dedicated encryption key if provided; fall back to JWT-derived key in development only.
	var encKey []byte
	if cfg.Encryption.ContactKeyHex != "" {
		var err error
		encKey, err = hex.DecodeString(cfg.Encryption.ContactKeyHex)
		if err != nil || len(encKey) != 32 {
			logger.Fatal("CONTACT_ENCRYPTION_KEY must be a 64-char hex string (32 bytes)")
		}
	} else if cfg.App.Env == "production" {
		logger.Fatal("CONTACT_ENCRYPTION_KEY is required in production (generate with: openssl rand -hex 32)")
	} else {
		logger.Warn("CONTACT_ENCRYPTION_KEY not set – deriving from JWT_SECRET (development only)")
		keyHash := sha256.Sum256([]byte(cfg.JWT.Secret))
		encKey = keyHash[:]
	}
	connSvc, err := connection.NewService(connRepo, connAgentRepo, encKey, logger)
	if err != nil {
		logger.Fatal("create connection service", zap.Error(err))
	}

	// ─── Scheduler ────────────────────────────────────────────────────────────
	sched := scheduler.NewScheduler(asynqClient, nil, schedTopicRepo, dbPool, logger)

	// ─── API Handlers ─────────────────────────────────────────────────────────
	authHandler := api.NewAuthHandler(authSvc)
	agentHandler := api.NewAgentHandler(agentSvc, embeddingSvc, logger)
	topicHandler := api.NewTopicHandler(topicSvc)
	discussionHandler := api.NewDiscussionHandler(discussionRepo, topicRepo)
	reportHandler := api.NewReportHandler(reportRepo, topicRepo)
	connHandler := api.NewConnectionHandler(connSvc, connRepo)
	userHandler := api.NewUserHandler(authSvc)

	// ─── Echo HTTP Server ─────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		if code >= 500 {
			logErr := err
			if he, ok := err.(*echo.HTTPError); ok && he.Internal != nil {
				logErr = he.Internal
			}
			logger.Error("internal server error",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Error(logErr),
			)
		}
		e.DefaultHTTPErrorHandler(err, c)
	}

	// Global middleware
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: cfg.CORS.AllowedOrigins,
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
		MaxAge:       3600,
	}))
	e.Use(apimiddleware.RequestLogger(logger))
	e.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStore(20)))

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "ok",
			"version": cfg.App.Version,
			"time":    time.Now().UTC(),
		})
	})

	// API v1 routes
	v1 := e.Group("/api/v1")
	registerRoutes(v1, authSvc, authHandler, agentHandler, topicHandler, discussionHandler, reportHandler, connHandler, userHandler, logger)

	// ─── Start everything ─────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start Asynq worker server
	mux := asynq.NewServeMux()
	registerWorkers(mux,
		topicRepo, agentRepo, embeddingSvc, llmGateway, matcher,
		discussionRepo, reportRepo, connRepo, sched, discussionEngine, reportGen,
		notifSvc, dbPool, logger,
	)
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			logger.Error("asynq server error", zap.Error(err))
		}
	}()

	// Start scheduler
	go sched.Start(ctx)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("server starting", zap.String("addr", addr), zap.String("env", cfg.App.Env))

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", zap.Error(err))
		}
	}()

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	asynqServer.Shutdown()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", zap.Error(err))
	}
	logger.Info("server stopped")
}

// registerRoutes wires API routes to handlers.
func registerRoutes(
	v1 *echo.Group,
	authSvc *auth.Service,
	authHandler *api.AuthHandler,
	agentHandler *api.AgentHandler,
	topicHandler *api.TopicHandler,
	discussionHandler *api.DiscussionHandler,
	reportHandler *api.ReportHandler,
	connHandler *api.ConnectionHandler,
	userHandler *api.UserHandler,
	logger *zap.Logger,
) {
	// Auth routes (public)
	authGroup := v1.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)

	// Protected routes
	protected := v1.Group("", apimiddleware.JWTAuth(authSvc, logger))

	// Agents
	agents := protected.Group("/agents")
	agents.POST("", agentHandler.Create)
	agents.GET("", agentHandler.List)
	agents.GET("/:id", agentHandler.Get)
	agents.PUT("/:id", agentHandler.Update)

	// Topics
	topics := protected.Group("/topics")
	topics.POST("", topicHandler.Submit)
	topics.GET("", topicHandler.List)
	topics.GET("/:id", topicHandler.Get)
	topics.DELETE("/:id", topicHandler.Cancel)

	// Discussions
	discussions := protected.Group("/discussions")
	discussions.GET("/:id", discussionHandler.Get)
	discussions.GET("/:id/messages", discussionHandler.GetMessages)

	// Reports
	reports := protected.Group("/reports")
	reports.GET("/:id", reportHandler.Get)
	reports.POST("/:id/rating", reportHandler.Rate)

	// Connections
	connections := protected.Group("/connections")
	connections.POST("", connHandler.Request)
	connections.GET("", connHandler.List)
	connections.POST("/:id/respond", connHandler.Respond)
	connections.GET("/:id/contacts", connHandler.GetContacts)

	// Users
	users := protected.Group("/users")
	users.GET("/me", userHandler.GetMe)
	users.POST("/fcm-token", userHandler.UpdateFCMToken)

	_ = logger
}

// registerWorkers registers Asynq task handlers.
func registerWorkers(
	mux *asynq.ServeMux,
	topicRepo topic.Repository,
	agentRepo agent.Repository,
	embeddingSvc *embedding.Service,
	llmGateway *llm.Gateway,
	matcher *matching.Matcher,
	discussionRepo discussion.Repository,
	reportRepo report.Repository,
	connRepo connection.Repository,
	sched *scheduler.Scheduler,
	discussionEngine *discussion.Engine,
	reportGen *report.Generator,
	notifSvc *notification.Service,
	dbPool *pgxpool.Pool,
	logger *zap.Logger,
) {
	matchWorker := worker.NewMatchTopicWorker(
		topicRepo, agentRepo, embeddingSvc, llmGateway,
		matcher, discussionRepo, sched, logger,
	)
	roundWorker := worker.NewDiscussionRoundWorker(
		discussionRepo, topicRepo, discussionEngine, sched, logger,
	)
	reportWorker := worker.NewReportGenerateWorker(
		discussionRepo, topicRepo, reportRepo, reportGen, dbPool, logger,
	)
	notifyWorker := worker.NewNotifyWorker(
		notifSvc, topicRepo, discussionRepo, reportRepo, logger,
	)
	expireWorker := worker.NewExpireConnectionsWorker(connRepo, logger)

	mux.HandleFunc(scheduler.TaskTypeMatchTopic, matchWorker.Handle)
	mux.HandleFunc(scheduler.TaskTypeDiscussionRound, roundWorker.Handle)
	mux.HandleFunc(scheduler.TaskTypeGenerateReport, reportWorker.Handle)
	mux.HandleFunc(scheduler.TaskTypeNotify1h, notifyWorker.Handle1h)
	mux.HandleFunc(scheduler.TaskTypeNotify12h, notifyWorker.Handle12h)
	mux.HandleFunc(scheduler.TaskTypeNotify48h, notifyWorker.Handle48h)
	mux.HandleFunc(scheduler.TaskTypeExpireConnections, expireWorker.Handle)
}

// newQdrantConn opens a gRPC connection to the Qdrant vector database.
// The connection is lazy (no network I/O until the first RPC call).
func newQdrantConn(cfg *config.QdrantConfig) (*grpc.ClientConn, error) {
	var dialOpts []grpc.DialOption

	if cfg.TLSEnabled {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.APIKey != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(
			func(ctx context.Context, method string, req, reply interface{},
				cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
			) error {
				ctx = metadata.AppendToOutgoingContext(ctx, "api-key", cfg.APIKey)
				return invoker(ctx, method, req, reply, cc, opts...)
			},
		))
	}

	return grpc.NewClient(
		fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		dialOpts...,
	)
}
