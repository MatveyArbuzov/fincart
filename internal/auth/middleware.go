package auth

import (
	"context"
	"net/http"
)

func ClaimsFromContext(
	ctx context.Context,
) (Claims, bool) {
	claims, ok := ctx.Value(
		claimsContextKey{},
	).(Claims)

	return claims, ok
}

func RequireRole(
	role string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				claims, ok := ClaimsFromContext(
					r.Context(),
				)

				if !ok {
					http.Error(
						w,
						"unauthorized",
						http.StatusUnauthorized,
					)
					return
				}

				if claims.Role != role {
					http.Error(
						w,
						"forbidden",
						http.StatusForbidden,
					)
					return
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}
