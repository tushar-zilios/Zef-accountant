package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync"

	"accountant/src/internal/config"
	"accountant/src/internal/db"
	dbaccountant "accountant/src/internal/db/accountant"
	"accountant/src/internal/logger"
	"accountant/src/internal/routes"
	"accountant/src/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Fatal error: %v", err)
		os.Stdout.Sync()
		os.Stderr.Sync()
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}
}

func run() error {
	if err := logger.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Cleanup()

	fmt.Println("Starting Zef Accountant service...")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	port := cfg.Port

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dbCancel()
	if _, err := db.InitDB(dbCtx, cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to initialize Postgres database: %w", err)
	}
	log.Println("DB connection pool initialized successfully.")
	defer db.CloseDB()

	if err := dbaccountant.EnsureGlobalVoucherTypes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure global voucher types: %v", err)
	}

	router := routes.NewRouter()
	serverAddr := ":" + port

	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	healthCtx, healthCancel := context.WithCancel(context.Background())
	defer healthCancel()
	healthDone := worker.StartHealthCheckWorker(healthCtx, []worker.ServiceHealth{
		{Name: "Accountant", URL: cfg.BackendURL + "/health"},
	})

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	serverErrChan := make(chan error, 1)
	go func() {
		log.Printf("Accountant HTTP server is starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	var runErr error
	select {
	case err := <-serverErrChan:
		log.Printf("HTTP server failed: %v", err)
		runErr = fmt.Errorf("HTTP server failed: %w", err)
	case sig := <-shutdownChan:
		log.Printf("Received signal %v, shutting down server gracefully...", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Warning: HTTP server Shutdown failed: %v", err)
	} else {
		log.Println("HTTP server shutdown successfully.")
	}

	healthCancel()
	var stopWg sync.WaitGroup
	stopWg.Add(1)
	go func() {
		defer stopWg.Done()
		select {
		case <-healthDone:
			log.Println("[HEALTH WORKER] Health check worker stopped gracefully.")
		case <-time.After(5 * time.Second):
			log.Println("[HEALTH WORKER] Warning: Health check worker shutdown timed out.")
		}
	}()
	stopWg.Wait()

	return runErr
}
