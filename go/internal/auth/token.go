package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

// The USER JWT minted at /login is also forwarded to the MCP server as the run-as token.
// The agent receives it in the X-Run-As-Token header and forwards it out-of-band —
// it never enters the LLM context, so the model cannot read or forge it.
const userTokenTTL = 90 * time.Minute

type claims struct {
	CustomerID string   `json:"customer_id,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	SessionID  string   `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

func MintUserToken(secret []byte, issuer, audience string, s domain.Subject) (string, error) {
	now := time.Now()
	c := claims{
		CustomerID: s.CustomerID,
		Roles:      s.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   s.UserID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(userTokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

func VerifyUserToken(secret []byte, issuer, audience, tokenStr string) (domain.Subject, error) {
	return verify(secret, tokenStr, issuer, audience)
}

func verify(secret []byte, tokenStr, issuer, audience string) (domain.Subject, error) {
	var c claims
	_, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return domain.Subject{}, err
	}
	if c.Subject == "" {
		return domain.Subject{}, errors.New("token missing subject")
	}
	return domain.Subject{UserID: c.Subject, CustomerID: c.CustomerID, Roles: c.Roles}, nil
}
