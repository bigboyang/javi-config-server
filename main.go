package main

import (
	"log"
	"net/http"

	"github.com/javi/config-server/internal/config"
)

func main() {
	service := config.NewService()
	handler := config.NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := ":18888"
	log.Printf("javi-config-server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
