package user

import "testing"

func TestRole_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		role Role
		want bool
	}{
		{
			role: RoleUser,
			want: true,
		},
		{
			role: RoleAdmin,
			want: true,
		},
		{
			role: Role(""),
			want: false,
		},
		{
			role: Role("manager"),
			want: false,
		},
		{
			role: Role("ADMIN"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			t.Parallel()

			if got := tt.role.IsValid(); got != tt.want {
				t.Fatalf(
					"Role(%q).IsValid() = %v, want %v",
					tt.role,
					got,
					tt.want,
				)
			}
		})
	}
}
