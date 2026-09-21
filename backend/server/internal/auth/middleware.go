// Package auth provides OAuth2 middleware for Bearer token validation
// against Google OAuth2 API, extracting user identity.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	golangOAuth2 "golang.org/x/oauth2"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// UserInfo holds the authenticated user's identity from the OAuth2 token.
type UserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Sub   string `json:"sub"`
}

type contextKey string

const userInfoKey contextKey = "userInfo"

// Middleware validates Bearer tokens via Google OAuth2 API and injects
// the parsed UserInfo into the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid Authorization format, expected Bearer token"}`, http.StatusUnauthorized)
			return
		}
		token := parts[1]

		userInfo, err := validateToken(r.Context(), token)
		if err != nil {
			log.Printf("auth: token validation failed: %v", err)
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userInfoKey, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken validates the access token using Google's tokeninfo
// endpoint and then fetches user profile details from userinfo.v2.me.
func validateToken(ctx context.Context, token string) (*UserInfo, error) {
	// Use the token as the credential for the OAuth2 API client
	svc, err := oauth2.NewService(ctx, option.WithTokenSource(
		&staticTokenSource{accessToken: token},
	))
	if err != nil {
		return nil, fmt.Errorf("create oauth2 service: %w", err)
	}

	// First validate the token via tokeninfo
	tokenInfoCall := svc.Tokeninfo()
	tokenInfoCall.AccessToken(token)
	info, err := tokenInfoCall.Do()
	if err != nil {
		return nil, fmt.Errorf("tokeninfo call: %w", err)
	}

	if info.Email == "" {
		return nil, fmt.Errorf("tokeninfo returned empty email")
	}

	// Build base UserInfo from tokeninfo data
	userInfo := &UserInfo{
		Email: info.Email,
		Sub:   info.UserId,
		Name:  info.Email, // fallback to email
	}

	// Try to get display name from userinfo endpoint
	userinfoCall := svc.Userinfo.V2.Me.Get()
	profile, err := userinfoCall.Do()
	if err == nil && profile != nil && profile.Name != "" {
		userInfo.Name = profile.Name
	}

	return userInfo, nil
}

// staticTokenSource is a simple token source that always returns the
// same access token.
type staticTokenSource struct {
	accessToken string
}

func (s *staticTokenSource) Token() (*golangOAuth2.Token, error) {
	return &golangOAuth2.Token{AccessToken: s.accessToken}, nil
}

// FromContext extracts the UserInfo from the request context.
// Returns nil if not present (unauthenticated request).
func FromContext(ctx context.Context) *UserInfo {
	u, ok := ctx.Value(userInfoKey).(*UserInfo)
	if !ok {
		return nil
	}
	return u
}

// Protected is a wrapper that applies the auth middleware and passes
// the extracted UserInfo to the handler.
func Protected(next func(w http.ResponseWriter, r *http.Request, u *UserInfo)) http.Handler {
	return Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := FromContext(r.Context())
		if u == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r, u)
	}))
}

// WriteUserInfoJSON is a convenience handler that returns the current
// user's identity as JSON. Useful for debugging and auth verification.
func WriteUserInfoJSON(w http.ResponseWriter, r *http.Request) {
	u := FromContext(r.Context())
	if u == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}
