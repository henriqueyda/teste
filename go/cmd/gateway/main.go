// Command gateway is the public front door (BFF): AuthN, run-as token mint, and
// forwarding conversation turns to the Python agent.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/henrique-yda/teste-tecnico-itau/internal/config"
	"github.com/henrique-yda/teste-tecnico-itau/internal/gateway"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
	"github.com/henrique-yda/teste-tecnico-itau/internal/telemetry"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdown, err := telemetry.Setup(ctx, "banking-gateway", cfg.OTLPEndpoint)
	if err != nil {
		log.Printf("gateway: telemetry setup failed (continuing without tracing): %v", err)
	} else {
		defer func() { _ = shutdown(ctx) }()
	}

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("gateway: db connect: %v", err)
	}
	defer pool.Close()

	srv := gateway.New(cfg, pool, []byte(cfg.JWTSecret))
	log.Printf("gateway listening on %s", cfg.GatewayAddr)
	if err := http.ListenAndServe(cfg.GatewayAddr, srv.Routes()); err != nil {
		log.Fatalf("gateway: %v", err)
	}
}
