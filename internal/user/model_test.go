package user

import "testing"

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{
			name: "user",
			role: RoleUser,
			want: true,
		},
		{
			name: "admin",
			role: RoleAdmin,
			want: true,
		},
		{
			name: "unknown",
			role: Role("banana"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Fatalf(
					"expected %v, got %v",
					tt.want,
					got,
				)
			}
		})
	}
}
