package order

type CreateOrderRequest struct {
	Items []CreateOrderItem `json:"items"`
}

type CreateOrderItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}
