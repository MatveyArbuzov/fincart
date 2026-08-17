package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

type JWTManager struct {
	secret     []byte
	expiration time.Duration
}

func NewJWTManager(
	secret string,
	expiration time.Duration,
) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

func (m *JWTManager) GenerateToken(
	userID int64,
	role string,
) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(m.expiration),
			),
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
) (Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"%w: unexpected signing method",
					ErrInvalidToken,
				)
			}

			return m.secret, nil
		},
	)

	if err != nil {
		return Claims{}, fmt.Errorf(
			"%w: %v",
			ErrInvalidToken,
			err,
		)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	if claims.UserID <= 0 {
		return Claims{}, ErrInvalidToken
	}

	if claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}

	return *claims, nil
}
