package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	authjwt "github.com/Dyuzhovsergey/gophermart/internal/auth/jwt"
	"github.com/Dyuzhovsergey/gophermart/internal/config"
	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
	"github.com/Dyuzhovsergey/gophermart/internal/logger"
	"github.com/Dyuzhovsergey/gophermart/internal/service/accounts"
	"github.com/Dyuzhovsergey/gophermart/internal/service/auth"
	"github.com/Dyuzhovsergey/gophermart/internal/service/orders"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawals"

	"github.com/Dyuzhovsergey/gophermart/internal/accrual"
	"github.com/Dyuzhovsergey/gophermart/internal/storage/postgres"
	"github.com/Dyuzhovsergey/gophermart/internal/worker"

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

	if cfg.JWTTTL <= 0 {
		log.Fatal("JWT_TTL must be positive", zap.String("jwt_ttl", cfg.JWTTTL.String()))
	}

	// ---------------- Контекст завершения приложения ----------------
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---------------- Подключение к Postgres ----------------
	connectCtx, cancelConnect := context.WithTimeout(rootCtx, 5*time.Second)
	defer cancelConnect()

	pool, err := postgres.Connect(connectCtx, cfg.DatabaseURI, log)
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	// ---------------- Миграции ----------------
	migCtx, cancelMig := context.WithTimeout(rootCtx, 5*time.Second)
	defer cancelMig()

	if err := postgres.RunMigrations(migCtx, pool); err != nil {
		log.Fatal("migrations failed", zap.Error(err))
	}
	log.Info("migrations applied")

	// ---------------- JWT менеджер ----------------

	jwtMgr, err := authjwt.New(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatal("jwt init failed", zap.Error(err))
	}

	// ---------------- Репозитории ----------------
	userRepo := postgres.NewUserRepository(pool)
	ordersRepo := postgres.NewOrdersRepository(pool)
	accountsRepo := postgres.NewAccountsRepository(pool)
	withdrawalsRepo := postgres.NewWithdrawalsRepository(pool)

	// ---------------- Сервисы ----------------
	accountsSvc := accounts.New(accountsRepo)
	ordersSvc := orders.New(ordersRepo)
	authSvc := auth.New(userRepo, jwtMgr, accountsSvc)
	withdrawalsSvc := withdrawals.New(withdrawalsRepo)

	// ---------------- HTTP Роутер ----------------
	router := httpserver.NewRouter(httpserver.Deps{
		Logger:      log,
		Auth:        authSvc,
		JWT:         jwtMgr,
		Orders:      ordersSvc,
		Accounts:    accountsSvc,
		Withdrawals: withdrawalsSvc,
	})

	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	// ---------------- Accrual client + Worker ----------------
	acClient, err := accrual.New(cfg.AccrualSystemAddress)
	if err != nil {
		log.Fatal("accrual client init failed", zap.Error(err))
	}

	w := worker.New(log, ordersRepo, acClient, 1*time.Second, 5)
	go w.Run(rootCtx)

	// ---------------- Запуск HTTP-сервера ----------------
	go func() {
		log.Info("starting server", zap.String("addr", cfg.RunAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen failed", zap.Error(err))
		}
	}()

	// ---------------- Ждём SIGINT / SIGTERM ----------------
	<-rootCtx.Done()
	log.Info("shutting down")

	// ---------------- Корректное завершение HTTP-сервера ----------------
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("stopped")
}
