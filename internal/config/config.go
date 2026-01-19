// Package config provides application configuration (FLAGS > ENV > DEFAULT)
package config

import (
	"flag"
	"os"
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

	// 3) flags переопределяют env/def
	flagRunAddr := flag.String("a", runAddr, "service run address (host:port)")
	flagDataBaseURI := flag.String("d", dbURI, "database uri")
	flagAccrualSystemAddress := flag.String("r", accrualAddr, "accrual system address")
	flagJWTSecret := flag.String("jwt-secret", jwtSecret, "jwt secret")
	flagJWTTTL := flag.String("jwt-ttl", jwtTTL.String(), "jwt ttl")
	flag.Parse()

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
	}
}
