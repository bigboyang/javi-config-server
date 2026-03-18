package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/javi/config-server/internal/config"
)

func main() {
	stateFile := os.Getenv("STATE_FILE")
	if stateFile == "" {
		stateFile = "config-state.json"
	}

	service := config.NewService(stateFile)
	handler := config.NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := ":18888"
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("javi-config-server starting on %s (state: %s)", addr, stateFile)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
