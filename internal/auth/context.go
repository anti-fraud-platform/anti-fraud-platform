package auth

import "context"

type contextKey string

const userClaimsKey contextKey = "user_claims"

// ContextWithUser stores JWT claims in the request context.
func ContextWithUser(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

// UserFromContext retrieves the JWT claims stored by the auth middleware.
// Returns nil and false if no claims are present.
func UserFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*Claims)
	return claims, ok
}
