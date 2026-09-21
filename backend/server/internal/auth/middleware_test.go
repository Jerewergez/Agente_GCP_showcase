package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromContext_NilWhenNotSet(t *testing.T) {
	u := FromContext(context.Background())
	if u != nil {
		t.Fatalf("expected nil, got %+v", u)
	}
}

func TestFromContext_ReturnsUserInfo(t *testing.T) {
	expected := &UserInfo{
		Email: "test@example.com",
		Name:  "Test User",
		Sub:   "12345",
	}
	ctx := context.WithValue(context.Background(), userInfoKey, expected)
	u := FromContext(ctx)
	if u == nil {
		t.Fatal("expected non-nil user info")
	}
	if u.Email != expected.Email {
		t.Fatalf("expected email %q, got %q", expected.Email, u.Email)
	}
	if u.Name != expected.Name {
		t.Fatalf("expected name %q, got %q", expected.Name, u.Name)
	}
	if u.Sub != expected.Sub {
		t.Fatalf("expected sub %q, got %q", expected.Sub, u.Sub)
	}
}

func TestMiddleware_MissingAuthHeader(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_InvalidFormat(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Invalid token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestProtected_ExtractsUserInfo(t *testing.T) {
	// With a real token validation we'd need actual Google tokens.
	// This test verifies that the Protected wrapper structure works
	// by checking the error path when no token is provided.
	handler := Protected(func(w http.ResponseWriter, r *http.Request, u *UserInfo) {
		t.Error("handler should not be called without auth")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
