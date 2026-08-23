package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/auth"
)

func request(
	t *testing.T,
	method string,
	path string,
	body interface{},
	token string,
) *http.Response {
	t.Helper()

	var reader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		reader = bytes.NewReader(data)
	}

	req := httptest.NewRequest(
		method,
		path,
		reader,
	)

	if body != nil {
		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	if token != "" {
		req.Header.Set(
			"Authorization",
			"Bearer "+token,
		)
	}

	recorder := httptest.NewRecorder()

	testServer.router.ServeHTTP(
		recorder,
		req,
	)

	return recorder.Result()
}

func assertStatus(
	t *testing.T,
	resp *http.Response,
	expected int,
) {
	t.Helper()

	defer resp.Body.Close()

	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)

		t.Fatalf(
			"unexpected status: got %d, want %d, body=%s",
			resp.StatusCode,
			expected,
			string(body),
		)
	}
}

func assertError(
	t *testing.T,
	resp *http.Response,
	expectedStatus int,
	expectedError string,
) {
	t.Helper()

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)

		t.Fatalf(
			"unexpected status: got %d, want %d, body=%s",
			resp.StatusCode,
			expectedStatus,
			string(body),
		)
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode error response: %v",
			err,
		)
	}

	if response.Error != expectedError {
		t.Fatalf(
			"unexpected error: got %q, want %q",
			response.Error,
			expectedError,
		)
	}
}

func path(
	format string,
	args ...interface{},
) string {
	return fmt.Sprintf(format, args...)
}

type testUser struct {
	ID           int64
	AccessToken  string
	RefreshToken string
}

func createUser(
	t *testing.T,
	email string,
	password string,
) testUser {
	t.Helper()

	register := request(
		t,
		http.MethodPost,
		"/api/v1/auth/register",
		map[string]interface{}{
			"email":    email,
			"password": password,
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
			"email":    email,
			"password": password,
		},
		"",
	)

	assertStatus(
		t,
		login,
		http.StatusOK,
	)

	data := decodeJSON[loginResponse](
		t,
		login,
	)

	return testUser{
		ID:           data.User.ID,
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
	}
}

func createAdmin(
	t *testing.T,
	email string,
	password string,
) testUser {
	t.Helper()

	user := createUser(
		t,
		email,
		password,
	)

	_, err := testServer.db.Exec(
		`
		UPDATE users
		SET role = 'admin'
		WHERE id = $1
		`,
		user.ID,
	)

	if err != nil {
		t.Fatalf(
			"failed to promote user to admin: %v",
			err,
		)
	}

	// Important:
	// createUser already generated a JWT with role=user.
	// We need a new token containing role=admin.

	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		jwtSecret = "integration-test-secret"
	}

	jwtManager := auth.NewJWTManager(jwtSecret)

	accessToken, err := jwtManager.GenerateToken(
		user.ID,
		"admin",
	)

	if err != nil {
		t.Fatalf(
			"failed to generate admin token: %v",
			err,
		)
	}

	user.AccessToken = accessToken

	return user
}
