package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

type updateCardLimitInput struct {
	CardID        string `json:"card_id"`
	NewLimitCents int64  `json:"new_limit_cents"`
}

type updateCardLimitOutput struct {
	CardID           string `json:"card_id"`
	PreviousLimit    int64  `json:"previous_limit_cents"`
	NewLimitCents    int64  `json:"new_limit_cents"`
	MaxLimitCents    int64  `json:"max_limit_cents"`
}

func (d Deps) updateCardLimit(ctx context.Context, req *mcpsdk.CallToolRequest, in updateCardLimitInput) (*mcpsdk.CallToolResult, updateCardLimitOutput, error) {
	ctx, span := startToolSpan(ctx, req, "update_card_limit")
	defer span.End()
	subject, corrID, err := d.subjectFromRequest(req)
	if err != nil {
		return nil, updateCardLimitOutput{}, err
	}

	// Pre-load to derive the resource owner for the authz ownership check.
	// Scenario C: if the LLM passes a card_id belonging to another customer, this
	// owner will differ from subject.CustomerID and authz will deny with "ownership_violation".
	card, err := store.GetCardByID(ctx, d.Pool, in.CardID)
	if err != nil {
		return nil, updateCardLimitOutput{}, err
	}

	spec := domain.ToolSpec{
		Name:               "update_card_limit",
		RequiredPermission: "card_limit.update",
		OwnerCustomerID:    card.CustomerID, // Scenario C enforced here
	}

	var out updateCardLimitOutput
	err = d.runToolPipeline(ctx, subject, corrID, spec, map[string]any{"card_id": in.CardID, "new_limit_cents": in.NewLimitCents}, func(tx pgx.Tx) (map[string]any, error) {
		c, err := store.GetCardByID(ctx, tx, in.CardID)
		if err != nil {
			return nil, err
		}
		if err := store.UpdateCardLimit(ctx, tx, in.CardID, in.NewLimitCents); err != nil {
			return nil, err
		}
		out = updateCardLimitOutput{
			CardID:        c.ID,
			PreviousLimit: c.LimitCents,
			NewLimitCents: in.NewLimitCents,
			MaxLimitCents: c.MaxLimitCents,
		}
		return map[string]any{
			"card_id": c.ID, "previous_limit_cents": c.LimitCents,
			"new_limit_cents": in.NewLimitCents, "max_limit_cents": c.MaxLimitCents,
		}, nil
	})
	if err != nil {
		return nil, updateCardLimitOutput{}, err
	}
	return nil, out, nil
}
