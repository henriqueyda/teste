// Package auth handles all token and step-up mechanics: verifying the USER JWT,
// minting the short-lived internal run-as token, and the mock OTP step-up flow
// (challenge issuance + action-bound signed assertion). Asymmetric signing keeps
// verification JWKS-ready.
//
// USER JWT + internal token in M1; step-up (scenario D) in M7.
package auth
