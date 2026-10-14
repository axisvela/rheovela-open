package rheovela

import "context"

// CredentialSource yields a bearer token for Rheovela requests. Consumers that
// need a governed credential (for example a short-lived workload assertion)
// provide their own implementation.
type CredentialSource interface {
	Token(ctx context.Context) (string, error)
}

type staticTokenSource string

func (s staticTokenSource) Token(context.Context) (string, error) { return string(s), nil }

// StaticTokenSource returns a CredentialSource with a fixed token.
func StaticTokenSource(token string) CredentialSource { return staticTokenSource(token) }

// TokenFuncSource adapts a function into a CredentialSource.
type TokenFuncSource func(ctx context.Context) (string, error)

// Token calls the underlying function.
func (f TokenFuncSource) Token(ctx context.Context) (string, error) { return f(ctx) }
