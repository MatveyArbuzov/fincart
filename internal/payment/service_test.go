package payment

import (
	"context"
	"testing"
)

func TestFakeService_Pay(t *testing.T) {
	tests := []struct {
		name   string
		result Result
	}{
		{
			name:   "success",
			result: ResultSuccess,
		},
		{
			name:   "failed",
			result: ResultFailed,
		},
		{
			name:   "timeout",
			result: ResultTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewFakeService(tt.result)

			result, err := service.Pay(
				context.Background(),
				100,
				300000,
				"EUR",
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.result {
				t.Fatalf(
					"expected %s, got %s",
					tt.result,
					result,
				)
			}
		})
	}
}
