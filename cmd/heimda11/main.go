package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/app"
	"github.com/ricardoaldape/Heimda11/internal/httpapi"
	"github.com/ricardoaldape/Heimda11/internal/secure"
	"github.com/ricardoaldape/Heimda11/internal/store"
)

func main() {
	addr := getenv("HEIMDA11_ADDR", ":8080")
	dataPath := getenv("HEIMDA11_DATA_PATH", "./data/heimda11.json")
	edition := getenv("HEIMDA11_EDITION", "community")
	adminToken := os.Getenv("HEIMDA11_ADMIN_TOKEN")
	masterKey := os.Getenv("HEIMDA11_MASTER_KEY")
	if adminToken == "" {
		log.Fatal("HEIMDA11_ADMIN_TOKEN is required")
	}
	if len(adminToken) < 24 {
		log.Fatal("HEIMDA11_ADMIN_TOKEN must be at least 24 characters")
	}
	cipher, err := secure.NewCipher(masterKey)
	if err != nil {
		log.Fatalf("initialize vault cipher: %v", err)
	}
	st, err := store.NewFileStore(dataPath)
	if err != nil {
		log.Fatalf("initialize store: %v", err)
	}
	svc := app.NewService(st, cipher, edition)
	handler := httpapi.New(svc, adminToken)

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("Heimda11 %s (%s) listening on %s", app.Version, edition, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}