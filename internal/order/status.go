package order

type OrderStatus string

const (
	OrderStatusDraft      OrderStatus = "draft"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case
		OrderStatusDraft,
		OrderStatusPending,
		OrderStatusPaid,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusCompleted,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func CanTransition(
	from OrderStatus,
	to OrderStatus,
) bool {
	switch from {
	case OrderStatusPending:
		return to == OrderStatusPaid ||
			to == OrderStatusCancelled

	case OrderStatusPaid:
		return to == OrderStatusProcessing

	case OrderStatusProcessing:
		return to == OrderStatusShipped

	case OrderStatusShipped:
		return to == OrderStatusCompleted

	default:
		return false
	}
}
