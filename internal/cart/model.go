package cart

import "time"

type Cart struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Status      string    `json:"status"`
	TotalAmount int64     `json:"total_amount"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	Items       []Item    `json:"items"`
}

type Item struct {
	ID          int64  `json:"id"`
	ProductID   int64  `json:"product_id"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
	Stock       int    `json:"stock"`
}

type AddItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}
