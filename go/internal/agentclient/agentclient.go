// Package agentclient is the gateway's client to the Python agent service. It forwards the
// user's message plus the run-as token (as an out-of-band header, never in the chat body)
// and returns the agent's reply.
package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client calls the agent's /invoke endpoint.
type Client struct {
	baseURL string
	httpc   *http.Client
}

// New constructs an agent client.
func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpc: &http.Client{Timeout: 60 * time.Second}}
}

type invokeRequest struct {
	Message  string `json:"message"`
	ThreadID string `json:"thread_id"`
}

// InvokeResult is the agent's response: the natural-language reply plus any structured
// data the tools returned (useful for assertions/observability).
type InvokeResult struct {
	Reply                string         `json:"reply"`
	Data                 map[string]any `json:"data,omitempty"`
	AwaitingConfirmation bool           `json:"awaiting_confirmation"`
	AwaitingPin          bool           `json:"awaiting_pin"`
}

// Invoke sends a turn to the agent. The run-as token travels in the X-Run-As-Token header;
// the agent forwards it to the MCP server and never exposes it to the LLM.
func (c *Client) Invoke(ctx context.Context, runAsToken, corrID, message, threadID string) (InvokeResult, error) {
	body, err := json.Marshal(invokeRequest{Message: message, ThreadID: threadID})
	if err != nil {
		return InvokeResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/invoke", bytes.NewReader(body))
	if err != nil {
		return InvokeResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Run-As-Token", runAsToken)
	req.Header.Set("X-Correlation-Id", corrID)

	resp, err := c.httpc.Do(req)
	if err != nil {
		return InvokeResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return InvokeResult{}, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}
	var out InvokeResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return InvokeResult{}, fmt.Errorf("decode agent response: %w", err)
	}
	return out, nil
}
