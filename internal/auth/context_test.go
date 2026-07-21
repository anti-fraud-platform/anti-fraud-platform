package auth

import (
	"context"
	"testing"
)

func TestContextWithAndWithoutUser(t *testing.T) {
	// Empty context — should return nil, false.
	claims, ok := UserFromContext(context.Background())
	if ok {
		t.Error("expected ok=false for empty context")
	}
	if claims != nil {
		t.Error("expected nil claims for empty context")
	}
}

func TestContextWithUserRoundTrip(t *testing.T) {
	input := &Claims{
		UserID:   10,
		Username: "ctxuser",
		Role:     "editor",
	}

	ctx := ContextWithUser(context.Background(), input)
	claims, ok := UserFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true after ContextWithUser")
	}
	if claims.UserID != 10 {
		t.Errorf("expected UserID 10, got %d", claims.UserID)
	}
	if claims.Username != "ctxuser" {
		t.Errorf("expected Username 'ctxuser', got %q", claims.Username)
	}
	if claims.Role != "editor" {
		t.Errorf("expected Role 'editor', got %q", claims.Role)
	}
}

func TestContextOverwriteClaims(t *testing.T) {
	first := &Claims{UserID: 1, Username: "first", Role: "viewer"}
	second := &Claims{UserID: 2, Username: "second", Role: "admin"}

	ctx := ContextWithUser(context.Background(), first)
	ctx = ContextWithUser(ctx, second)

	claims, ok := UserFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if claims.UserID != 2 || claims.Username != "second" {
		t.Errorf("expected second claim, got %+v", claims)
	}
}

func TestContextTypeSafety(t *testing.T) {
	// Put a non-Claims value into context under the same key — should not interfere.
	ctx := context.WithValue(context.Background(), userClaimsKey, "not-a-claims-struct")
	claims, ok := UserFromContext(ctx)
	if ok {
		t.Error("expected ok=false for wrong type in context")
	}
	if claims != nil {
		t.Error("expected nil claims for wrong type")
	}
}

func TestContextNilClaims(t *testing.T) {
	ctx := context.WithValue(context.Background(), userClaimsKey, (*Claims)(nil))
	claims, ok := UserFromContext(ctx)
	// A typed nil pointer will have ok=true but claims will be nil.
	// This is an edge case the middleware never creates, but we document the behavior.
	if ok && claims != nil {
		t.Error("expected nil claims for typed nil")
	}
}

func TestContextDoesNotLeakBetweenRequests(t *testing.T) {
	claims1 := &Claims{UserID: 1, Username: "a", Role: "viewer"}
	claims2 := &Claims{UserID: 2, Username: "b", Role: "admin"}

	ctx1 := ContextWithUser(context.Background(), claims1)
	ctx2 := ContextWithUser(context.Background(), claims2)

	got1, _ := UserFromContext(ctx1)
	got2, _ := UserFromContext(ctx2)

	if got1.UserID != 1 {
		t.Errorf("ctx1 should have UserID 1, got %d", got1.UserID)
	}
	if got2.UserID != 2 {
		t.Errorf("ctx2 should have UserID 2, got %d", got2.UserID)
	}
}
