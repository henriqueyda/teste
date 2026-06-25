// Command mcp-server is the security kernel: the only path from agent reasoning to
// action. It speaks the Model Context Protocol (Streamable HTTP) and enforces token
// verification, RBAC/ownership, and transactional audit on every tool call.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/henrique-yda/teste-tecnico-itau/internal/config"
	"github.com/henrique-yda/teste-tecnico-itau/internal/mcp"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
	"github.com/henrique-yda/teste-tecnico-itau/internal/telemetry"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	shutdown, err := telemetry.Setup(ctx, "banking-mcp", cfg.OTLPEndpoint)
	if err != nil {
		log.Printf("mcp-server: telemetry setup failed (continuing without tracing): %v", err)
	} else {
		defer func() { _ = shutdown(ctx) }()
	}

	pool, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("mcp-server: db connect: %v", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewHTTPHandler(mcp.Deps{
		Pool:     pool,
		Secret:   []byte(cfg.JWTSecret),
		Issuer:   cfg.JWTIssuer,
		Audience: cfg.JWTAudience,
	}))

	log.Printf("mcp-server (MCP/Streamable HTTP) listening on %s/mcp", cfg.MCPServerAddr)
	if err := http.ListenAndServe(cfg.MCPServerAddr, mux); err != nil {
		log.Fatalf("mcp-server: %v", err)
	}
}
