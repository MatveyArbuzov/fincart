package payment

import (
	"context"
	"testing"
)

func TestNewFakeService(t *testing.T) {
	t.Parallel()

	service := NewFakeService(ResultSuccess)

	if service.Result != ResultSuccess {
		t.Fatalf(
			"Result = %q, want %q",
			service.Result,
			ResultSuccess,
		)
	}
}

func TestFakeService_Pay(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			service := NewFakeService(tt.result)

			got, err := service.Pay(
				context.Background(),
				123,
				1000,
				"USD",
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.result {
				t.Fatalf(
					"result = %q, want %q",
					got,
					tt.result,
				)
			}
		})
	}
}
