package cart

import (
	"context"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type Repository interface {
	GetDraft(
		ctx context.Context,
		tx database.Tx,
		userID int64,
	) (Cart, error)

	CreateDraft(
		ctx context.Context,
		tx database.Tx,
		userID int64,
	) (Cart, error)

	GetItem(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		productID int64,
	) (Item, error)

	AddItem(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		productID int64,
		quantity int,
		unitPrice int64,
	) (Item, error)

	UpdateItem(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		productID int64,
		quantity int,
	) error

	DeleteItem(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		productID int64,
	) error

	GetItems(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
	) ([]Item, error)

	UpdateTotal(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		totalAmount int64,
		currency string,
	) error

	Checkout(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
	) error

	GetDraftByID(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
	) (Cart, error)

	UpdateItemPrice(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		productID int64,
		unitPrice int64,
	) error
}
