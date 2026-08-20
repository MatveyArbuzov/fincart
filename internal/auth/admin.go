package auth

import "net/http"

func AdminMiddleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		role, ok := RoleFromContext(r.Context())

		if !ok || role != "admin" {
			http.Error(
				w,
				"forbidden",
				http.StatusForbidden,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
