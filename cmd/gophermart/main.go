package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authjwt "github.com/Dyuzhovsergey/gophermart/internal/auth/jwt"
	"github.com/Dyuzhovsergey/gophermart/internal/config"
	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
	"github.com/Dyuzhovsergey/gophermart/internal/logger"
	"github.com/Dyuzhovsergey/gophermart/internal/service/auth"
	"github.com/Dyuzhovsergey/gophermart/internal/storage/postgres"

	"go.uber.org/zap"
)

func main() {
	// ---------------- Инициализация логгера ----------------
	log, err := logger.Init()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	// ---------------- Парсинг конфига ----------------
	cfg := config.Parse()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is empty")
	}
	if cfg.JWTTTL <= 0 {
		log.Fatal("JWT_TTL must be positive", zap.String("jwt_ttl", cfg.JWTTTL.String()))
	}

	// ---------------- Контекст завершения приложения ----------------
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---------------- Подключение к Postgres ----------------
	pool, err := postgres.Connect(rootCtx, cfg.DatabaseURI, log)
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	// ---------------- Миграции ----------------
	if err := postgres.RunMigrations(rootCtx, pool); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}
	log.Info("migrations applied")

	// ---------------- Репозиторий пользователей ----------------
	userRepo := postgres.NewUserRepository(pool)

	// ---------------- JWT менеджер ----------------
	// Секрет лучше хранить в ENV. Для разработки можно задать дефолт.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Warn("JWT_SECRET is empty, using dev secret")
		jwtSecret = "dev-secret"
	}

	jwtMgr, err := authjwt.New(jwtSecret, 24*time.Hour)
	if err != nil {
		log.Fatal("jwt init failed", zap.Error(err))
	}

	// ---------------- Auth service ----------------
	authSvc := auth.New(userRepo, jwtMgr)

	// ---------------- HTTP Роутер ----------------
	router := httpserver.NewRouter(httpserver.Deps{
		Logger: log,
		Auth:   authSvc,
	})

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	// ---------------- Запуск HTTP-сервера ----------------
	go func() {
		log.Info("starting server", zap.String("addr", cfg.RunAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen failed", zap.Error(err))
		}
	}()

	// ---------------- Ждём SIGINT / SIGTERM ----------------
	<-rootCtx.Done()
	log.Info("shutting down")

	// ---------------- Корректное завершение HTTP-сервера ----------------
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("stopped")
}
