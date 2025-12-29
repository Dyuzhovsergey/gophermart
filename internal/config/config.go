// Package config provides application configuration.
package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
}

func Parse() *Config {
	// defaults
	const (
		DefaultRunAddress           = "localhost:8080"
		DefaultDatabaseURI          = ""
		DefaultAccrualSystemAddress = ""
	)

	// 1) берём дефолты
	runAddr := DefaultRunAddress
	dbURI := DefaultDatabaseURI
	accrualAddr := DefaultAccrualSystemAddress

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

	// 3) flags переопределяют env/def
	flagRunAddr := flag.String("a", runAddr, "service run address (host:port)")
	flagDataBaseURI := flag.String("d", dbURI, "database uri")
	flagAccrualSystemAddress := flag.String("r", accrualAddr, "accrual system address")
	flag.Parse()

	return &Config{
		RunAddress:           *flagRunAddr,
		DatabaseURI:          *flagDataBaseURI,
		AccrualSystemAddress: *flagAccrualSystemAddress,
	}
}
