package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/henrique-yda/teste-tecnico-itau/internal/auth"
	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
}

// handleLogin verifies credentials and issues a USER JWT. On any failure it returns a
// single generic "invalid credentials" so it does not leak which usernames exist.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	u, err := store.GetUserByUsername(r.Context(), s.pool, req.Username)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil || !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	subject := domain.Subject{UserID: u.ID, CustomerID: u.CustomerID, Roles: u.Roles}
	token, err := auth.MintUserToken(s.secret, s.cfg.JWTIssuer, s.cfg.JWTAudience, subject)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: token, TokenType: "Bearer", ExpiresIn: 900,
		UserID: u.ID, Roles: u.Roles,
	})
}

type chatRequest struct {
	Message  string `json:"message"`
	ThreadID string `json:"thread_id"`
}

// handleChat is the conversational entrypoint. It forwards the user's bearer token to the
// Python agent out-of-band (X-Run-As-Token header). The agent passes it to the MCP server,
// which re-verifies it independently — the LLM never sees the token.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if _, ok := subjectFrom(r.Context()); !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	threadID := req.ThreadID
	if threadID == "" {
		threadID = newCorrelationID()
	}

	corrID := newCorrelationID()

	result, err := s.agent.Invoke(r.Context(), tokenFrom(r.Context()), corrID, req.Message, threadID)
	if err != nil {
		http.Error(w, "agent call failed", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"reply":                  result.Reply,
		"data":                   result.Data,
		"thread_id":              threadID,
		"correlation_id":         corrID,
		"awaiting_confirmation":  result.AwaitingConfirmation,
		"awaiting_pin":           result.AwaitingPin,
	})
}

type stepUpRequest struct {
	Pin string `json:"pin"`
}

type stepUpResponse struct {
	SessionToken string `json:"session_token"`
}

// handleStepUp verifies the caller's transaction PIN (bcrypt) and issues a short-lived
// one-use session token. The frontend sends this token back via /chat to resume the
// paused graph — the raw PIN never enters the LangGraph checkpoint.
func (s *Server) handleStepUp(w http.ResponseWriter, r *http.Request) {
	subject, ok := subjectFrom(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if subject.CustomerID == "" {
		http.Error(w, "staff accounts have no transaction PIN", http.StatusForbidden)
		return
	}
	var req stepUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Pin == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	hash, err := store.GetTransactionPINHash(r.Context(), s.pool, subject.UserID)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	ok2, err := auth.VerifyPassword(req.Pin, hash)
	if err != nil || !ok2 {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := store.CreatePinSession(r.Context(), s.pool, subject.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, stepUpResponse{SessionToken: token})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newCorrelationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
