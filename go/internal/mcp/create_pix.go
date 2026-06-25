package mcp

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/henrique-yda/teste-tecnico-itau/internal/auth"
	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

type createPixInput struct {
	RecipientKey    string `json:"recipient_key"`
	AmountCents     int64  `json:"amount_cents"`
	TransactionPin  string `json:"transaction_pin,omitempty"`   // direct bcrypt (testing/CLI)
	PinSessionToken string `json:"pin_session_token,omitempty"` // preferred: one-use token from /step-up
}

type createPixOutput struct {
	TransactionID   string `json:"transaction_id"`
	RecipientKey    string `json:"recipient_key"`
	AmountCents     int64  `json:"amount_cents"`
	NewBalanceCents int64  `json:"new_balance_cents"`
}

// createPix executes a PIX transfer after verifying the caller's transaction PIN.
//
// If transaction_pin is absent the tool returns pin_required so the Python step_up_node
// can interrupt and collect it from the user via chat, then retry with the PIN.
// PIN verification is bcrypt server-side in Go — the LLM never sees or decides on it.
func (d Deps) createPix(ctx context.Context, req *mcpsdk.CallToolRequest, in createPixInput) (*mcpsdk.CallToolResult, createPixOutput, error) {
	ctx, span := startToolSpan(ctx, req, "create_pix")
	defer span.End()
	subject, corrID, err := d.subjectFromRequest(req)
	if err != nil {
		return nil, createPixOutput{}, err
	}
	if subject.CustomerID == "" {
		return nil, createPixOutput{}, fmt.Errorf("caller is not a customer")
	}
	if in.AmountCents <= 0 {
		return nil, createPixOutput{}, fmt.Errorf("amount_cents must be positive")
	}
	switch {
	case in.PinSessionToken != "":
		if err := store.ConsumePinSession(ctx, d.Pool, subject.UserID, in.PinSessionToken); err != nil {
			return nil, createPixOutput{}, err
		}
	case in.TransactionPin != "":
		pinHash, err := store.GetTransactionPINHash(ctx, d.Pool, subject.UserID)
		if err != nil {
			return nil, createPixOutput{}, fmt.Errorf("internal error")
		}
		ok, err := auth.VerifyPassword(in.TransactionPin, pinHash)
		if err != nil || !ok {
			return nil, createPixOutput{}, fmt.Errorf("senha de transação incorreta")
		}
	default:
		return nil, createPixOutput{}, fmt.Errorf(
			`{"error":"pin_required","message":"Informe a senha de transação para continuar"}`,
		)
	}

	spec := domain.ToolSpec{
		Name:               "create_pix",
		RequiredPermission: "pix.create",
		OwnerCustomerID:    subject.CustomerID,
	}
	args := map[string]any{"recipient_key": in.RecipientKey, "amount_cents": in.AmountCents}

	var out createPixOutput
	err = d.runToolPipeline(ctx, subject, corrID, spec, args, func(tx pgx.Tx) (map[string]any, error) {
		acc, err := store.GetAccountByCustomerID(ctx, tx, subject.CustomerID)
		if err != nil {
			return nil, err
		}
		if in.AmountCents > acc.PixDailyLimit {
			return nil, fmt.Errorf("valor excede o limite diário de PIX (R$%.2f)", float64(acc.PixDailyLimit)/100)
		}
		newBalance, err := store.DebitAccount(ctx, tx, subject.CustomerID, in.AmountCents)
		if err != nil {
			return nil, err
		}
		txID := newCorrelationID()
		out = createPixOutput{
			TransactionID:   txID,
			RecipientKey:    in.RecipientKey,
			AmountCents:     in.AmountCents,
			NewBalanceCents: newBalance,
		}
		return map[string]any{
			"transaction_id": txID, "recipient_key": in.RecipientKey,
			"amount_cents": in.AmountCents, "new_balance_cents": newBalance,
		}, nil
	})
	if err != nil {
		return nil, createPixOutput{}, err
	}
	return nil, out, nil
}
