package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

type claimsContextKey struct{}

func (m *JWTManager) GenerateToken(
	userID int64,
	role string,
) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				now.Add(15 * time.Minute),
			),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(m.secret)
}

func (m *JWTManager) ParseToken(
	tokenString string,
) (int64, string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return m.secret, nil
		},
	)

	if err != nil {
		return 0, "", ErrInvalidToken
	}

	if !token.Valid {
		return 0, "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return 0, "", ErrInvalidToken
	}

	if claims.UserID <= 0 || claims.Role == "" {
		return 0, "", ErrInvalidToken
	}

	return claims.UserID, claims.Role, nil
}

func (m *JWTManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Fields(authHeader)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, role, err := m.ParseToken(parts[1])
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims := Claims{
			UserID: userID,
			Role:   role,
		}

		ctx := context.WithValue(
			r.Context(),
			claimsContextKey{},
			claims,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}
