// Package policy holds banking business rules as pure functions: card-limit
// eligibility, PIX daily limits, and step-up thresholds. Pure functions keep this
// trivially table-testable and out of both the LLM and the execution glue.
//
// Used by the MCP tools in M6 (eligibility) and M7 (PIX limits / step-up).
package policy
