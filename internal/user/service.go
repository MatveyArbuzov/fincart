package user

import (
	"context"
	"errors"
	"strings"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidUser        = errors.New("invalid user")
)

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

type Service struct {
	transactions TransactionManager
	repository   Repository
}

func NewService(
	transactions TransactionManager,
	repository Repository,
) *Service {
	return &Service{
		transactions: transactions,
		repository:   repository,
	}
}

func (s *Service) Register(
	ctx context.Context,
	email string,
	password string,
) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || password == "" {
		return User{}, ErrInvalidUser
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return User{}, err
	}

	var result User

	err = s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			var err error

			result, err = s.repository.Create(
				ctx,
				tx,
				email,
				string(passwordHash),
				RoleUser,
			)

			return err
		},
	)

	if err != nil {
		return User{}, err
	}

	return result, nil
}

func (s *Service) Login(
	ctx context.Context,
	email string,
	password string,
) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}

	var result User

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			user, storedPassword, err := s.repository.GetByEmail(
				ctx,
				tx,
				email,
			)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					return ErrInvalidCredentials
				}

				return err
			}

			if err := bcrypt.CompareHashAndPassword(
				[]byte(storedPassword),
				[]byte(password),
			); err != nil {
				return ErrInvalidCredentials
			}

			result = user

			return nil
		},
	)

	if err != nil {
		return User{}, err
	}

	return result, nil
}
