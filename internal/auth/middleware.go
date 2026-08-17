package auth

import (
	"net/http"
	"strings"
)

func (m *JWTManager) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			if header == "" {
				http.Error(
					w,
					"missing authorization header",
					http.StatusUnauthorized,
				)
				return
			}

			parts := strings.SplitN(
				header,
				" ",
				2,
			)

			if len(parts) != 2 ||
				!strings.EqualFold(parts[0], "Bearer") ||
				parts[1] == "" {
				http.Error(
					w,
					"invalid authorization header",
					http.StatusUnauthorized,
				)
				return
			}

			claims, err := m.ParseToken(parts[1])
			if err != nil {
				http.Error(
					w,
					"invalid token",
					http.StatusUnauthorized,
				)
				return
			}

			ctx := WithClaims(
				r.Context(),
				claims,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}
