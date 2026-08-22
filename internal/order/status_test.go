package order

import "testing"

func TestOrderStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status OrderStatus
		want   bool
	}{
		{
			name:   "draft",
			status: OrderStatusDraft,
			want:   true,
		},
		{
			name:   "pending",
			status: OrderStatusPending,
			want:   true,
		},
		{
			name:   "paid",
			status: OrderStatusPaid,
			want:   true,
		},
		{
			name:   "processing",
			status: OrderStatusProcessing,
			want:   true,
		},
		{
			name:   "shipped",
			status: OrderStatusShipped,
			want:   true,
		},
		{
			name:   "completed",
			status: OrderStatusCompleted,
			want:   true,
		},
		{
			name:   "cancelled",
			status: OrderStatusCancelled,
			want:   true,
		},
		{
			name:   "empty",
			status: "",
			want:   false,
		},
		{
			name:   "unknown",
			status: OrderStatus("unknown"),
			want:   false,
		},
		{
			name:   "similar status",
			status: OrderStatus("Pending"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.status.IsValid(); got != tt.want {
				t.Fatalf(
					"IsValid() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from OrderStatus
		to   OrderStatus
		want bool
	}{
		// Pending
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
			name: "pending to processing",
			from: OrderStatusPending,
			to:   OrderStatusProcessing,
			want: false,
		},

		// Paid
		{
			name: "paid to processing",
			from: OrderStatusPaid,
			to:   OrderStatusProcessing,
			want: true,
		},
		{
			name: "paid to cancelled",
			from: OrderStatusPaid,
			to:   OrderStatusCancelled,
			want: false,
		},

		// Processing
		{
			name: "processing to shipped",
			from: OrderStatusProcessing,
			to:   OrderStatusShipped,
			want: true,
		},
		{
			name: "processing to completed",
			from: OrderStatusProcessing,
			to:   OrderStatusCompleted,
			want: false,
		},

		// Shipped
		{
			name: "shipped to completed",
			from: OrderStatusShipped,
			to:   OrderStatusCompleted,
			want: true,
		},
		{
			name: "shipped to cancelled",
			from: OrderStatusShipped,
			to:   OrderStatusCancelled,
			want: false,
		},

		// States without outgoing transitions
		{
			name: "draft to pending",
			from: OrderStatusDraft,
			to:   OrderStatusPending,
			want: false,
		},
		{
			name: "completed to anything",
			from: OrderStatusCompleted,
			to:   OrderStatusPending,
			want: false,
		},
		{
			name: "cancelled to anything",
			from: OrderStatusCancelled,
			to:   OrderStatusPending,
			want: false,
		},

		// Self transitions
		{
			name: "pending to pending",
			from: OrderStatusPending,
			to:   OrderStatusPending,
			want: false,
		},
		{
			name: "paid to paid",
			from: OrderStatusPaid,
			to:   OrderStatusPaid,
			want: false,
		},

		// Invalid statuses
		{
			name: "unknown from",
			from: OrderStatus("unknown"),
			to:   OrderStatusPaid,
			want: false,
		},
		{
			name: "unknown to",
			from: OrderStatusPending,
			to:   OrderStatus("unknown"),
			want: false,
		},
		{
			name: "both unknown",
			from: OrderStatus("foo"),
			to:   OrderStatus("bar"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf(
					"CanTransition(%q, %q) = %v, want %v",
					tt.from,
					tt.to,
					got,
					tt.want,
				)
			}
		})
	}
}
