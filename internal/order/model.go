package order

import "time"

type Order struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Status      string    `json:"status"`
	TotalAmount int64     `json:"total_amount"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderItem struct {
	ID        int64 `json:"id"`
	OrderID   int64 `json:"order_id"`
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	UnitPrice int64 `json:"unit_price"`
}
