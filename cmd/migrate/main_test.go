package main

import (
	"strings"
	"testing"
)

func TestMigrationInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string

		wantVersion int64
		wantName    string

		wantErr       bool
		wantErrString string
	}{
		{
			name:        "valid migration",
			file:        "migrations/001_create_products.sql",
			wantVersion: 1,
			wantName:    "001_create_products.sql",
		},
		{
			name:        "valid migration with large version",
			file:        "migrations/123456_add_feature.sql",
			wantVersion: 123456,
			wantName:    "123456_add_feature.sql",
		},
		{
			name:        "valid migration with nested path",
			file:        "/tmp/project/migrations/007_test.sql",
			wantVersion: 7,
			wantName:    "007_test.sql",
		},
		{
			name:          "missing underscore",
			file:          "001.sql",
			wantErr:       true,
			wantErrString: "invalid migration filename",
		},
		{
			name:          "empty filename",
			file:          "",
			wantErr:       true,
			wantErrString: "invalid migration filename",
		},
		{
			name:          "invalid version",
			file:          "migrations/abc_create_products.sql",
			wantErr:       true,
			wantErrString: "invalid migration version",
		},
		{
			name:        "negative version",
			file:        "migrations/-001_test.sql",
			wantVersion: -1,
			wantName:    "-001_test.sql",
		},
		{
			name:          "empty version",
			file:          "migrations/_test.sql",
			wantErr:       true,
			wantErrString: "invalid migration version",
		},
		{
			name:          "non numeric version",
			file:          "migrations/1a_test.sql",
			wantErr:       true,
			wantErrString: "invalid migration version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			version, name, err := migrationInfo(tt.file)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("error = nil, want error")
				}

				if tt.wantErrString != "" &&
					!strings.Contains(err.Error(), tt.wantErrString) {
					t.Fatalf(
						"error = %q, want substring %q",
						err.Error(),
						tt.wantErrString,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("error = %v, want nil", err)
			}

			if version != tt.wantVersion {
				t.Fatalf(
					"version = %d, want %d",
					version,
					tt.wantVersion,
				)
			}

			if name != tt.wantName {
				t.Fatalf(
					"name = %q, want %q",
					name,
					tt.wantName,
				)
			}
		})
	}
}
