// Command server is the My Bad, Coach backend entry point.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alecbakholdin/mybadcoach/server/internal/api"
	"github.com/alecbakholdin/mybadcoach/server/internal/db"
)

const (
	defaultAddr        = ":8080"
	defaultDSN         = "mbc.db"
	readHeaderTimeout  = 5 * time.Second
	productionEnvValue = "production"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	dsn := envOrDefault("MBC_DB_DSN", defaultDSN)
	addr := envOrDefault("MBC_ADDR", defaultAddr)

	// db.Open applies the embedded migrations before returning.
	sqlDB, err := db.Open(dsn)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			log.Printf("close database: %v", closeErr)
		}
	}()

	if seedErr := db.Seed(context.Background(), sqlDB); seedErr != nil {
		return seedErr
	}

	handler, err := api.NewHandler(api.Config{
		DB:            sqlDB,
		SecureCookies: os.Getenv("MBC_ENV") == productionEnvValue,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
