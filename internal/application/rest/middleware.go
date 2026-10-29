package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

type userIDCtxKey struct{}

type TokenParser interface {
	ParseToken(accessToken string) (int, error)
}

type AuthMiddleware struct {
	tokenParser TokenParser
	reply       reply
}

func NewAuthMiddleware(
	tokenParser TokenParser,
	reply reply,
) *AuthMiddleware {
	return &AuthMiddleware{
		tokenParser: tokenParser,
		reply:       reply,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get(authorizationHeader)
		if header == "" {
			m.reply.ErrorUnauthorized(w, errors.New("empty auth header"))

			return
		}

		token, ok := strings.CutPrefix(header, bearerPrefix)
		if !ok || token == "" {
			m.reply.ErrorUnauthorized(w, errors.New("invalid auth header"))

			return
		}

		userID, err := m.tokenParser.ParseToken(token)
		if err != nil {
			m.reply.ErrorUnauthorized(w, fmt.Errorf("invalid token: %w", err))

			return
		}

		ctx := context.WithValue(r.Context(), userIDCtxKey{}, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDCtxKey{}).(int)

	return userID, ok
}
