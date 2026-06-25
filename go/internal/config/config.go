// Package config loads service configuration from the environment with sane local
// defaults, so the binaries run out of the box for development.
package config

import "os"

// Config is the shared configuration for the gateway and MCP server.
type Config struct {
	GatewayAddr   string // gateway HTTP listen address
	MCPServerAddr string // MCP server HTTP listen address
	MCPServerURL  string // base URL the gateway uses to reach the MCP server
	AgentURL      string // base URL the gateway uses to reach the Python agent
	DatabaseURL   string // Postgres DSN (connect as least-privilege app_user)

	JWTSecret   string // HMAC-SHA256 secret shared by gateway and MCP server
	JWTIssuer   string // issuer claim for the USER JWT
	JWTAudience string // audience claim for the USER JWT

	OTLPEndpoint string // OTLP HTTP collector host:port (e.g. localhost:4318)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads configuration from the environment.
func Load() Config {
	return Config{
		GatewayAddr:   getenv("GATEWAY_ADDR", ":8080"),
		MCPServerAddr: getenv("MCP_SERVER_ADDR", ":7070"),
		MCPServerURL:  getenv("MCP_SERVER_URL", "http://localhost:7070/mcp"),
		AgentURL:      getenv("AGENT_URL", "http://localhost:8000"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://app_user:app_user@localhost:5432/bank?sslmode=disable"),

		JWTSecret:   getenv("JWT_SECRET", "dev-only-change-in-prod"),
		JWTIssuer:   getenv("JWT_ISSUER", "banking-agent"),
		JWTAudience: getenv("JWT_AUDIENCE", "banking-agent"),

		OTLPEndpoint: getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
	}
}
