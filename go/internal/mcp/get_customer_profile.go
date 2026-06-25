package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/jackc/pgx/v5"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
	"github.com/henrique-yda/teste-tecnico-itau/internal/store"
)

// profileInput has no fields: get_customer_profile takes no identity argument. Any
// arguments the LLM tries to inject are irrelevant — identity comes from the verified
// token, so there is no IDOR surface.
type profileInput struct{}

// profileOutput is the typed result. The SDK marshals it into both the structured and
// text content of the tool result automatically.
type profileOutput struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Document  string `json:"document"`
	IsRetiree bool   `json:"is_retiree"`
}

// getCustomerProfile returns the authenticated caller's own profile.
func (d Deps) getCustomerProfile(ctx context.Context, req *mcpsdk.CallToolRequest, _ profileInput) (*mcpsdk.CallToolResult, profileOutput, error) {
	ctx, span := startToolSpan(ctx, req, "get_customer_profile")
	defer span.End()
	subject, corrID, err := d.subjectFromRequest(req)
	if err != nil {
		return nil, profileOutput{}, err
	}

	spec := domain.ToolSpec{Name: "get_customer_profile", RequiredPermission: "customer.read"}

	var out profileOutput
	err = d.runToolPipeline(ctx, subject, corrID, spec, map[string]any{}, func(tx pgx.Tx) (map[string]any, error) {
		if subject.CustomerID == "" {
			return nil, fmt.Errorf("caller is not a customer")
		}
		c, err := store.GetCustomerByID(ctx, tx, subject.CustomerID)
		if err != nil {
			return nil, err
		}
		out = profileOutput{ID: c.ID, FullName: c.FullName, Document: c.Document, IsRetiree: c.IsRetiree}
		return map[string]any{
			"id": c.ID, "full_name": c.FullName, "document": c.Document, "is_retiree": c.IsRetiree,
		}, nil
	})
	if err != nil {
		return nil, profileOutput{}, err
	}
	return nil, out, nil // SDK fills Content + StructuredContent from out
}
