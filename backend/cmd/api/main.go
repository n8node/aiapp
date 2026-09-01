package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/config"
	"github.com/n8node/aiapp/internal/db"
	httpapi "github.com/n8node/aiapp/internal/httpapi"
	"github.com/n8node/aiapp/internal/migrate"
	"github.com/n8node/aiapp/internal/repository"
	"github.com/n8node/aiapp/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	if err := migrate.Up(cfg.DatabaseURL); err != nil {
		logger.Error("migrations", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	handle := pool.Handle()
	users := repository.NewUserRepository(handle)
	workspaces := repository.NewWorkspaceRepository(handle)
	invites := repository.NewInviteRepository(handle)
	settings := repository.NewSettingsRepository(handle)
	sessions := repository.NewSessionRepository(handle)
	audit := repository.NewAuditRepository(handle)
	tokens := authn.NewJWT(cfg.JWTSecret, cfg.CookieSecure)
	authSvc := service.NewAuthService(users, workspaces, invites, settings, sessions, audit, tokens, cfg.TOTPKey, "RigIntel")
	bitrixStore := repository.NewBitrixRepository(handle)
	bitrixSvc := service.NewBitrixService(settings, bitrixStore, audit, cfg.TOTPKey)
	workspaceSvc := service.NewWorkspaceService(workspaces, users, bitrixStore, audit)
	storageRepo := repository.NewStorageSettingsRepository(handle)
	storageSvc := service.NewStorageSettingsService(storageRepo, audit, cfg, cfg.TOTPKey)
	objectStore := service.NewObjectStorage(storageSvc)
	uploadSessions := service.NewUploadSessionService(cfg.JWTSecret)
	diskRepo := repository.NewDiskRepository(handle)
	diskSvc := service.NewDiskService(diskRepo, authSvc, objectStore, uploadSessions, audit)
	docRepo := repository.NewDocumentRepository(handle)
	extractor := service.NewHTTPExtractor(cfg.ExtractURL, cfg.ExtractToken)
	docSvc := service.NewDocumentService(docRepo, diskSvc, objectStore, authSvc, audit, extractor)
	ingestWorker := service.NewIngestWorker(docRepo, objectStore, extractor, logger)
	gw := service.NewGatewayClient(cfg.GatewayURL, cfg.GatewayToken)
	mlRepo := repository.NewMLModelRepository(handle)
	modelSvc := service.NewModelService(mlRepo, audit, cfg.StudioURL, gw)
	kbRepo := repository.NewKnowledgeRepository(handle)
	kbSvc := service.NewKnowledgeService(kbRepo, mlRepo, diskSvc, authSvc, audit, gw)
	vectorizeWorker := service.NewVectorizeWorker(kbRepo, diskRepo, objectStore, extractor, gw, logger)
	trainRepo := repository.NewTrainingRepository(handle)
	trainSvc := service.NewTrainingService(trainRepo, authSvc, audit)

	if cfg.SuperadminEmail != "" && cfg.SuperadminPassword != "" {
		if _, created, err := authSvc.EnsureSuperAdmin(ctx, cfg.SuperadminEmail, cfg.SuperadminPassword, cfg.SuperadminName); err != nil {
			logger.Error("ensure superadmin", "error", err)
			os.Exit(1)
		} else if created {
			logger.Info("superadmin created")
		} else {
			logger.Info("superadmin ready")
		}
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(httpapi.Dependencies{Config: cfg, Ping: pool, Auth: authSvc, Bitrix: bitrixSvc, Workspaces: workspaceSvc, Storage: storageSvc, Disk: diskSvc, Documents: docSvc, Models: modelSvc, Knowledge: kbSvc, Training: trainSvc, Tokens: tokens}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	go ingestWorker.Run(runCtx)
	go vectorizeWorker.Run(runCtx)

	go func() {
		logger.Info("server starting", "addr", httpServer.Addr, "version", config.Version)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	runCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
	}
	logger.Info("server stopped")
}
