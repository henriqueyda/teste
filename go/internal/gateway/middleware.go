package gateway

import (
	"context"
	"net/http"
	"strings"

	"github.com/henrique-yda/teste-tecnico-itau/internal/auth"
	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

type ctxKey int

const (
	subjectKey ctxKey = iota
	tokenKey
)

func withSubject(ctx context.Context, s domain.Subject) context.Context {
	return context.WithValue(ctx, subjectKey, s)
}

func subjectFrom(ctx context.Context) (domain.Subject, bool) {
	s, ok := ctx.Value(subjectKey).(domain.Subject)
	return s, ok
}

// withToken stores the raw bearer token so handleChat can forward it to the agent.
func withToken(ctx context.Context, tok string) context.Context {
	return context.WithValue(ctx, tokenKey, tok)
}

func tokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(tokenKey).(string)
	return t
}

// authMiddleware verifies the USER JWT and attaches the resulting Subject to the request
// context. This is where identity is established at the edge — from a verified token, not
// from anything the caller asserts in the body.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		subject, err := auth.VerifyUserToken(s.secret, s.cfg.JWTIssuer, s.cfg.JWTAudience, token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := withSubject(r.Context(), subject)
		ctx = withToken(ctx, token)
		next(w, r.WithContext(ctx))
	}
}
