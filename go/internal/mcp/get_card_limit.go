package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

type cardLimitInput struct {
	CardID string `json:"card_id"`
}

type cardLimitOutput struct {
	CardID        string `json:"card_id"`
	LimitCents    int64  `json:"limit_cents"`
	MaxLimitCents int64  `json:"max_limit_cents"`
}

func (d Deps) getCardLimit(ctx context.Context, req *mcpsdk.CallToolRequest, in cardLimitInput) (*mcpsdk.CallToolResult, cardLimitOutput, error) {
	ctx, span := startToolSpan(ctx, req, "get_card_limit")
	defer span.End()
	subject, corrID, err := d.subjectFromRequest(req)
	if err != nil {
		return nil, cardLimitOutput{}, err
	}

	// Pre-load the card outside the pipeline to derive the owner. This read is a
	// non-transactional authorization pre-check; the authoritative read (and audit)
	// happen inside the pipeline transaction below.
	card, err := store.GetCardByID(ctx, d.Pool, in.CardID)
	if err != nil {
		return nil, cardLimitOutput{}, err
	}

	spec := domain.ToolSpec{
		Name:               "get_card_limit",
		RequiredPermission: "card_limit.read",
		OwnerCustomerID:    card.CustomerID, // authz enforces this inside the pipeline
	}

	var out cardLimitOutput
	err = d.runToolPipeline(ctx, subject, corrID, spec, map[string]any{"card_id": in.CardID}, func(tx pgx.Tx) (map[string]any, error) {
		c, err := store.GetCardByID(ctx, tx, in.CardID)
		if err != nil {
			return nil, err
		}
		out = cardLimitOutput{CardID: c.ID, LimitCents: c.LimitCents, MaxLimitCents: c.MaxLimitCents}
		return map[string]any{"card_id": c.ID, "limit_cents": c.LimitCents, "max_limit_cents": c.MaxLimitCents}, nil
	})
	if err != nil {
		return nil, cardLimitOutput{}, err
	}
	return nil, out, nil
}
