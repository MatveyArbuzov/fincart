package user

import (
	"context"
	"errors"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type mockTransactionManager struct {
	withinTransactionFunc func(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

func (m *mockTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	if m.withinTransactionFunc != nil {
		return m.withinTransactionFunc(ctx, fn)
	}

	return nil
}

type mockRepository struct {
	createFunc func(
		ctx context.Context,
		tx database.Tx,
		email string,
		passwordHash string,
		role Role,
	) (User, error)

	getByEmailFunc func(
		ctx context.Context,
		tx database.Tx,
		email string,
	) (User, string, error)

	getRoleByIDFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (string, error)
}

func (m *mockRepository) Create(
	ctx context.Context,
	tx database.Tx,
	email string,
	passwordHash string,
	role Role,
) (User, error) {
	if m.createFunc != nil {
		return m.createFunc(
			ctx,
			tx,
			email,
			passwordHash,
			role,
		)
	}

	return User{}, nil
}

func (m *mockRepository) GetByEmail(
	ctx context.Context,
	tx database.Tx,
	email string,
) (User, string, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(
			ctx,
			tx,
			email,
		)
	}

	return User{}, "", nil
}

func (m *mockRepository) GetRoleByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (string, error) {
	if m.getRoleByIDFunc != nil {
		return m.getRoleByIDFunc(
			ctx,
			tx,
			id,
		)
	}

	return "", nil
}

func TestService_Register(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")
	transactionErr := errors.New("transaction error")

	tests := []struct {
		name string

		email    string
		password string

		repositoryErr error

		transactionErr error

		want          User
		wantErr       error
		wantCreate    bool
		wantEmail     string
		wantRole      Role
		checkPassword bool
	}{
		{
			name: "success",

			email:    "  TEST@Example.COM  ",
			password: "password123",

			want: User{
				ID:    1,
				Email: "test@example.com",
				Role:  RoleUser,
			},

			wantCreate:    true,
			wantEmail:     "test@example.com",
			wantRole:      RoleUser,
			checkPassword: true,
		},

		{
			name: "invalid email",

			email:    "   ",
			password: "password123",

			wantErr:    ErrInvalidUser,
			wantCreate: false,
		},

		{
			name: "empty password",

			email:    "test@example.com",
			password: "",

			wantErr:    ErrInvalidUser,
			wantCreate: false,
		},

		{
			name: "repository error",

			email:    "test@example.com",
			password: "password123",

			repositoryErr: repositoryErr,

			wantErr:    repositoryErr,
			wantCreate: true,
			wantEmail:  "test@example.com",
			wantRole:   RoleUser,
		},

		{
			name: "transaction error",

			email:    "test@example.com",
			password: "password123",

			transactionErr: transactionErr,

			wantErr:    transactionErr,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var createCalled bool

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					if tt.transactionErr != nil {
						return tt.transactionErr
					}

					return fn(nil)
				},
			}

			repository := &mockRepository{
				createFunc: func(
					ctx context.Context,
					tx database.Tx,
					email string,
					passwordHash string,
					role Role,
				) (User, error) {
					createCalled = true

					if email != tt.wantEmail {
						t.Fatalf(
							"Create email = %q, want %q",
							email,
							tt.wantEmail,
						)
					}

					if role != tt.wantRole {
						t.Fatalf(
							"Create role = %q, want %q",
							role,
							tt.wantRole,
						)
					}

					if tt.checkPassword {
						if err := bcrypt.CompareHashAndPassword(
							[]byte(passwordHash),
							[]byte(tt.password),
						); err != nil {
							t.Fatalf(
								"password hash does not match password: %v",
								err,
							)
						}
					}

					return tt.want, tt.repositoryErr
				},
			}

			service := NewService(
				transactionManager,
				repository,
			)

			got, err := service.Register(
				context.Background(),
				tt.email,
				tt.password,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Register() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"Register() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if createCalled != tt.wantCreate {
				t.Fatalf(
					"Create called = %v, want %v",
					createCalled,
					tt.wantCreate,
				)
			}
		})
	}
}

func TestService_Register_PasswordIsHashed(t *testing.T) {
	t.Parallel()

	var (
		gotHash string
		gotUser User
	)

	transactionManager := &mockTransactionManager{
		withinTransactionFunc: func(
			ctx context.Context,
			fn func(tx database.Tx) error,
		) error {
			return fn(nil)
		},
	}

	repository := &mockRepository{
		createFunc: func(
			ctx context.Context,
			tx database.Tx,
			email string,
			passwordHash string,
			role Role,
		) (User, error) {
			gotHash = passwordHash

			gotUser = User{
				ID:    1,
				Email: email,
				Role:  role,
			}

			return gotUser, nil
		},
	}

	service := NewService(
		transactionManager,
		repository,
	)

	const password = "super-secret-password"

	got, err := service.Register(
		context.Background(),
		"test@example.com",
		password,
	)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if got != gotUser {
		t.Fatalf(
			"Register() = %+v, want %+v",
			got,
			gotUser,
		)
	}

	if gotHash == "" {
		t.Fatal("password hash is empty")
	}

	if gotHash == password {
		t.Fatal("password was stored without hashing")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(gotHash),
		[]byte(password),
	); err != nil {
		t.Fatalf(
			"password hash does not match password: %v",
			err,
		)
	}
}

func TestService_Login(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")
	transactionErr := errors.New("transaction error")

	password := "password123"

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}

	tests := []struct {
		name string

		email    string
		password string

		repositoryErr error

		transactionErr error

		want      User
		wantErr   error
		wantGet   bool
		wantEmail string
	}{
		{
			name: "success",

			email:    "  TEST@Example.COM  ",
			password: password,

			want: User{
				ID:    1,
				Email: "test@example.com",
				Role:  RoleUser,
			},

			wantGet:   true,
			wantEmail: "test@example.com",
		},

		{
			name: "invalid email",

			email:    "   ",
			password: password,

			wantErr: ErrInvalidCredentials,
			wantGet: false,
		},

		{
			name: "empty password",

			email:    "test@example.com",
			password: "",

			wantErr: ErrInvalidCredentials,
			wantGet: false,
		},

		{
			name: "user not found",

			email:    "test@example.com",
			password: password,

			repositoryErr: ErrUserNotFound,

			wantErr:   ErrInvalidCredentials,
			wantGet:   true,
			wantEmail: "test@example.com",
		},

		{
			name: "wrong password",

			email:    "test@example.com",
			password: "wrong-password",

			wantGet:   true,
			wantEmail: "test@example.com",

			want: User{
				// ID:    1,
				// Email: "test@example.com",
				// Role:  RoleUser,
			},
			wantErr: ErrInvalidCredentials,
		},

		{
			name: "repository error",

			email:    "test@example.com",
			password: password,

			repositoryErr: repositoryErr,

			wantErr:   repositoryErr,
			wantGet:   true,
			wantEmail: "test@example.com",
		},

		{
			name: "transaction error",

			email:    "test@example.com",
			password: password,

			transactionErr: transactionErr,

			wantErr: transactionErr,
			wantGet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var getCalled bool

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					if tt.transactionErr != nil {
						return tt.transactionErr
					}

					return fn(nil)
				},
			}

			repository := &mockRepository{
				getByEmailFunc: func(
					ctx context.Context,
					tx database.Tx,
					email string,
				) (User, string, error) {
					getCalled = true

					if email != tt.wantEmail {
						t.Fatalf(
							"GetByEmail email = %q, want %q",
							email,
							tt.wantEmail,
						)
					}

					if tt.repositoryErr != nil {
						return User{}, "", tt.repositoryErr
					}

					return tt.want, string(passwordHash), nil
				},
			}

			service := NewService(
				transactionManager,
				repository,
			)

			got, err := service.Login(
				context.Background(),
				tt.email,
				tt.password,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Login() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"Login() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if getCalled != tt.wantGet {
				t.Fatalf(
					"GetByEmail called = %v, want %v",
					getCalled,
					tt.wantGet,
				)
			}
		})
	}
}

func TestService_Login_InvalidCredentials_DoesNotLeakUserExistence(
	t *testing.T,
) {
	t.Parallel()

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}

	tests := []struct {
		name string

		repositoryErr error
		password      string
	}{
		{
			name:          "user not found",
			repositoryErr: ErrUserNotFound,
			password:      "correct-password",
		},
		{
			name:     "wrong password",
			password: "wrong-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			}

			repository := &mockRepository{
				getByEmailFunc: func(
					ctx context.Context,
					tx database.Tx,
					email string,
				) (User, string, error) {
					if tt.repositoryErr != nil {
						return User{}, "", tt.repositoryErr
					}

					return User{
							ID:    1,
							Email: email,
							Role:  RoleUser,
						},
						string(passwordHash),
						nil
				},
			}

			service := NewService(
				transactionManager,
				repository,
			)

			_, err := service.Login(
				context.Background(),
				"test@example.com",
				tt.password,
			)

			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf(
					"Login() error = %v, want %v",
					err,
					ErrInvalidCredentials,
				)
			}
		})
	}
}
