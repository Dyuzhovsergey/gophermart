// Package config provides application configuration (FLAGS > ENV > DEFAULT)
package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Addr
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string

	// JWT
	JWTSecret string
	JWTTTL    time.Duration

	// Postgres pool
	DBMaxConns          int32
	DBMinConns          int32
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	DBHealthCheckPeriod time.Duration

	// Тайминги приложения
	DBConnectTimeout      time.Duration
	HTTPShutdownTimeout   time.Duration
	WorkerInterval        time.Duration
	WorkerBatchSize       int
	WorkerShutdownTimeout time.Duration
}

func Parse() *Config {
	// defaults
	const (
		DefaultRunAddress           = "localhost:8080"
		DefaultDatabaseURI          = ""
		DefaultAccrualSystemAddress = "localhost:8081"

		DefaultJWTSecret = "dev-secret"
		DefaultJWTTTL    = 24 * time.Hour

		DefaultDBMaxConns          int32 = 10
		DefaultDBMinConns          int32 = 1
		DefaultDBMaxConnLifetime         = 30 * time.Minute
		DefaultDBMaxConnIdleTime         = 5 * time.Minute
		DefaultDBHealthCheckPeriod       = 30 * time.Second

		DefaultDBConnectTimeout      = 5 * time.Second
		DefaultHTTPShutdownTimeout   = 5 * time.Second
		DefaultWorkerInterval        = 1 * time.Second
		DefaultWorkerBatchSize       = 5
		DefaultWorkerShutdownTimeout = 15 * time.Second
	)

	// 1) берём дефолты
	runAddr := DefaultRunAddress
	dbURI := DefaultDatabaseURI
	accrualAddr := DefaultAccrualSystemAddress

	jwtSecret := DefaultJWTSecret
	jwtTTL := DefaultJWTTTL

	dbMaxConns := DefaultDBMaxConns
	dbMinConns := DefaultDBMinConns
	dbMaxConnLifetime := DefaultDBMaxConnLifetime
	dbMaxConnIdleTime := DefaultDBMaxConnIdleTime
	dbHealthCheckPeriod := DefaultDBHealthCheckPeriod

	dbConnectTimeout := DefaultDBConnectTimeout
	httpShutdownTimeout := DefaultHTTPShutdownTimeout
	workerInterval := DefaultWorkerInterval
	workerBatchSize := DefaultWorkerBatchSize
	workerShutdownTimeout := DefaultWorkerShutdownTimeout

	// 2) env переопределяет дефолты
	if v := os.Getenv("RUN_ADDRESS"); v != "" {
		runAddr = v
	}
	if v := os.Getenv("DATABASE_URI"); v != "" {
		dbURI = v
	}
	if v := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); v != "" {
		accrualAddr = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		jwtSecret = v
	}
	if v := os.Getenv("JWT_TTL"); v != "" {
		ttl, err := time.ParseDuration(v)
		if err == nil {
			jwtTTL = ttl
		}
	}
	if v := os.Getenv("DB_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			dbConnectTimeout = d
		}
	}

	if v := os.Getenv("HTTP_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			httpShutdownTimeout = d
		}
	}

	if v := os.Getenv("WORKER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			workerInterval = d
		}
	}

	if v := os.Getenv("WORKER_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workerBatchSize = n
		}
	}

	if v := os.Getenv("WORKER_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			workerShutdownTimeout = d
		}
	}

	// 3) flags переопределяют env/def
	flagRunAddr := flag.String("a", runAddr, "service run address (host:port)")
	flagDataBaseURI := flag.String("d", dbURI, "database uri")
	flagAccrualSystemAddress := flag.String("r", accrualAddr, "accrual system address")

	flagJWTSecret := flag.String("jwt-secret", jwtSecret, "jwt secret")
	flagJWTTTL := flag.String("jwt-ttl", jwtTTL.String(), "jwt ttl")

	flagDBConnectTimeout := flag.String("db-connect-timeout", dbConnectTimeout.String(), "db connect timeout")
	flagHTTPShutdownTimeout := flag.String("http-shutdown-timeout", httpShutdownTimeout.String(), "http shutdown timeout")
	flagWorkerInterval := flag.String("worker-interval", workerInterval.String(), "worker tick interval")
	flagWorkerBatchSize := flag.Int("worker-batch", workerBatchSize, "worker batch size")
	flagWorkerShutdownTimeout := flag.String("worker-shutdown-timeout", workerShutdownTimeout.String(), "worker shutdown timeout")

	flag.Parse()

	if d, err := time.ParseDuration(*flagDBConnectTimeout); err == nil && d > 0 {
		dbConnectTimeout = d
	}
	if d, err := time.ParseDuration(*flagHTTPShutdownTimeout); err == nil && d > 0 {
		httpShutdownTimeout = d
	}
	if d, err := time.ParseDuration(*flagWorkerInterval); err == nil && d > 0 {
		workerInterval = d
	}
	if *flagWorkerBatchSize > 0 {
		workerBatchSize = *flagWorkerBatchSize
	}
	if d, err := time.ParseDuration(*flagWorkerShutdownTimeout); err == nil && d > 0 {
		workerShutdownTimeout = d
	}

	// парсим ttl уже после флагов
	parsedTTL, err := time.ParseDuration(*flagJWTTTL)
	if err != nil {
		parsedTTL = jwtTTL
	}

	return &Config{
		RunAddress:           *flagRunAddr,
		DatabaseURI:          *flagDataBaseURI,
		AccrualSystemAddress: *flagAccrualSystemAddress,
		JWTSecret:            *flagJWTSecret,
		JWTTTL:               parsedTTL,

		DBMaxConns:          dbMaxConns,
		DBMinConns:          dbMinConns,
		DBMaxConnLifetime:   dbMaxConnLifetime,
		DBMaxConnIdleTime:   dbMaxConnIdleTime,
		DBHealthCheckPeriod: dbHealthCheckPeriod,

		DBConnectTimeout:      dbConnectTimeout,
		HTTPShutdownTimeout:   httpShutdownTimeout,
		WorkerInterval:        workerInterval,
		WorkerBatchSize:       workerBatchSize,
		WorkerShutdownTimeout: workerShutdownTimeout,
	}
}
