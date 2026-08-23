package integration

import (
	"net/http"
	"testing"
)

type registerResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type loginResponse struct {
	User struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`

	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestAuth_RegisterLoginRefreshLogout(t *testing.T) {
	resetDatabase(t)

	register := request(
		t,
		http.MethodPost,
		"/api/v1/auth/register",
		map[string]interface{}{
			"email":    "user@example.com",
			"password": "password123",
		},
		"",
	)

	assertStatus(
		t,
		register,
		http.StatusCreated,
	)

	login := request(
		t,
		http.MethodPost,
		"/api/v1/auth/login",
		map[string]interface{}{
			"email":    "USER@example.com",
			"password": "password123",
		},
		"",
	)

	assertStatus(
		t,
		login,
		http.StatusOK,
	)

	loginData := decodeJSON[loginResponse](
		t,
		login,
	)

	if loginData.User.Email != "user@example.com" {
		t.Fatalf(
			"unexpected email: %s",
			loginData.User.Email,
		)
	}

	if loginData.User.Role != "user" {
		t.Fatalf(
			"unexpected role: %s",
			loginData.User.Role,
		)
	}

	if loginData.AccessToken == "" {
		t.Fatal("access token is empty")
	}

	if loginData.RefreshToken == "" {
		t.Fatal("refresh token is empty")
	}

	oldRefreshToken := loginData.RefreshToken

	refresh := request(
		t,
		http.MethodPost,
		"/api/v1/auth/refresh",
		map[string]interface{}{
			"refresh_token": oldRefreshToken,
		},
		"",
	)

	assertStatus(
		t,
		refresh,
		http.StatusOK,
	)

	refreshData := decodeJSON[refreshResponse](
		t,
		refresh,
	)

	if refreshData.AccessToken == "" {
		t.Fatal("new access token is empty")
	}

	if refreshData.RefreshToken == "" {
		t.Fatal("new refresh token is empty")
	}

	if refreshData.RefreshToken == oldRefreshToken {
		t.Fatal(
			"refresh token was not rotated",
		)
	}

	// Old refresh token must no longer work.

	refreshOld := request(
		t,
		http.MethodPost,
		"/api/v1/auth/refresh",
		map[string]interface{}{
			"refresh_token": oldRefreshToken,
		},
		"",
	)

	assertError(
		t,
		refreshOld,
		http.StatusUnauthorized,
		"invalid_refresh_token",
	)

	// Logout new refresh token.

	logout := request(
		t,
		http.MethodPost,
		"/api/v1/auth/logout",
		map[string]interface{}{
			"refresh_token": refreshData.RefreshToken,
		},
		"",
	)

	assertStatus(
		t,
		logout,
		http.StatusNoContent,
	)

	// Revoked token cannot be refreshed.

	refreshRevoked := request(
		t,
		http.MethodPost,
		"/api/v1/auth/refresh",
		map[string]interface{}{
			"refresh_token": refreshData.RefreshToken,
		},
		"",
	)

	assertError(
		t,
		refreshRevoked,
		http.StatusUnauthorized,
		"invalid_refresh_token",
	)
}

func TestAuth_InvalidLogin(t *testing.T) {
	resetDatabase(t)

	register := request(
		t,
		http.MethodPost,
		"/api/v1/auth/register",
		map[string]interface{}{
			"email":    "user@example.com",
			"password": "password123",
		},
		"",
	)

	assertStatus(
		t,
		register,
		http.StatusCreated,
	)

	login := request(
		t,
		http.MethodPost,
		"/api/v1/auth/login",
		map[string]interface{}{
			"email":    "user@example.com",
			"password": "wrong-password",
		},
		"",
	)

	assertError(
		t,
		login,
		http.StatusUnauthorized,
		"invalid_credentials",
	)
}

func TestAuth_ProtectedEndpointWithoutToken(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/cart",
		nil,
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusUnauthorized,
	)
}
