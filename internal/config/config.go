// Package config provides application configuration (FLAGS > ENV > DEFAULT)
package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string

	JWTSecret string
	JWTTTL    time.Duration
}

func Parse() *Config {
	// defaults
	const (
		DefaultRunAddress           = "localhost:8080"
		DefaultDatabaseURI          = ""
		DefaultAccrualSystemAddress = ""

		DefaultJWTTTL = 24 * time.Hour
	)

	// 1) берём дефолты
	runAddr := DefaultRunAddress
	dbURI := DefaultDatabaseURI
	accrualAddr := DefaultAccrualSystemAddress
	jwtSecret := ""
	jwtTTL := DefaultJWTTTL

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
	flagJWTSecret := flag.String("jwt-secret", jwtSecret, "jwt secret (required)")
	flagJWTTTL := flag.String("jwt-ttl", jwtTTL.String(), "jwt ttl (e.g. 24h, 30m)")
	flag.Parse()

	// парсим ttl уже после флагов
	parsedTTL, err := time.ParseDuration(*flagJWTTTL)
	if err != nil {
		parsedTTL = DefaultJWTTTL
	}

	return &Config{
		RunAddress:           *flagRunAddr,
		DatabaseURI:          *flagDataBaseURI,
		AccrualSystemAddress: *flagAccrualSystemAddress,
		JWTSecret:            *flagJWTSecret,
		JWTTTL:               parsedTTL,
	}
}
