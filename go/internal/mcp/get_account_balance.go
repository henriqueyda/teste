package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

// balanceInput takes no arguments: the account is always resolved from the verified
// token, so there is no LLM-injectable identity parameter and no IDOR surface.
type balanceInput struct{}

type balanceOutput struct {
	AccountID     string `json:"account_id"`
	BalanceCents  int64  `json:"balance_cents"`
	PixDailyLimit int64  `json:"pix_daily_limit_cents"`
}

func (d Deps) getAccountBalance(ctx context.Context, req *mcpsdk.CallToolRequest, _ balanceInput) (*mcpsdk.CallToolResult, balanceOutput, error) {
	ctx, span := startToolSpan(ctx, req, "get_account_balance")
	defer span.End()
	subject, corrID, err := d.subjectFromRequest(req)
	if err != nil {
		return nil, balanceOutput{}, err
	}

	spec := domain.ToolSpec{Name: "get_account_balance", RequiredPermission: "account.read"}

	var out balanceOutput
	err = d.runToolPipeline(ctx, subject, corrID, spec, map[string]any{}, func(tx pgx.Tx) (map[string]any, error) {
		if subject.CustomerID == "" {
			return nil, fmt.Errorf("caller is not a customer")
		}
		acc, err := store.GetAccountByCustomerID(ctx, tx, subject.CustomerID)
		if err != nil {
			return nil, err
		}
		out = balanceOutput{
			AccountID:     acc.ID,
			BalanceCents:  acc.BalanceCents,
			PixDailyLimit: acc.PixDailyLimit,
		}
		return map[string]any{
			"account_id": acc.ID, "balance_cents": acc.BalanceCents, "pix_daily_limit_cents": acc.PixDailyLimit,
		}, nil
	})
	if err != nil {
		return nil, balanceOutput{}, err
	}
	return nil, out, nil
}
