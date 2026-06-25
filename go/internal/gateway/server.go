package gateway

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/henrique-yda/teste-tecnico-itau/internal/agentclient"
	"github.com/henrique-yda/teste-tecnico-itau/internal/config"
)

// Server is the gateway/BFF: the public front door. It owns AuthN (login + token verify),
// mints the run-as token, and forwards conversation turns to the Python agent.
type Server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	secret []byte // HMAC-SHA256 secret: signs USER + internal run-as tokens
	agent  *agentclient.Client
}

// New constructs the gateway server.
func New(cfg config.Config, pool *pgxpool.Pool, secret []byte) *Server {
	return &Server{cfg: cfg, pool: pool, secret: secret, agent: agentclient.New(cfg.AgentURL)}
}

// Routes returns the HTTP handler. /chat sits behind the auth middleware; /login is public
// (it issues tokens); /healthz is for readiness checks.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /chat", s.authMiddleware(s.handleChat))
	mux.HandleFunc("POST /step-up", s.authMiddleware(s.handleStepUp))
	return mux
}
