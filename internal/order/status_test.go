package order

import "testing"

func TestOrderStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status OrderStatus
		valid  bool
	}{
		{
			name:   "pending",
			status: OrderStatusPending,
			valid:  true,
		},
		{
			name:   "paid",
			status: OrderStatusPaid,
			valid:  true,
		},
		{
			name:   "processing",
			status: OrderStatusProcessing,
			valid:  true,
		},
		{
			name:   "shipped",
			status: OrderStatusShipped,
			valid:  true,
		},
		{
			name:   "completed",
			status: OrderStatusCompleted,
			valid:  true,
		},
		{
			name:   "cancelled",
			status: OrderStatusCancelled,
			valid:  true,
		},
		{
			name:   "unknown",
			status: OrderStatus("banana"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Fatalf(
					"expected %v, got %v",
					tt.valid,
					got,
				)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from OrderStatus
		to   OrderStatus
		want bool
	}{
		{
			name: "pending to paid",
			from: OrderStatusPending,
			to:   OrderStatusPaid,
			want: true,
		},
		{
			name: "pending to cancelled",
			from: OrderStatusPending,
			to:   OrderStatusCancelled,
			want: true,
		},
		{
			name: "paid to processing",
			from: OrderStatusPaid,
			to:   OrderStatusProcessing,
			want: true,
		},
		{
			name: "processing to shipped",
			from: OrderStatusProcessing,
			to:   OrderStatusShipped,
			want: true,
		},
		{
			name: "shipped to completed",
			from: OrderStatusShipped,
			to:   OrderStatusCompleted,
			want: true,
		},
		{
			name: "cancelled to paid",
			from: OrderStatusCancelled,
			to:   OrderStatusPaid,
			want: false,
		},
		{
			name: "completed to cancelled",
			from: OrderStatusCompleted,
			to:   OrderStatusCancelled,
			want: false,
		},
		{
			name: "pending to completed",
			from: OrderStatusPending,
			to:   OrderStatusCompleted,
			want: false,
		},
		{
			name: "shipped to pending",
			from: OrderStatusShipped,
			to:   OrderStatusPending,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf(
					"expected %v, got %v",
					tt.want,
					got,
				)
			}
		})
	}
}
