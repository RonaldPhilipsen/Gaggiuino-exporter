package cmd

import (
	"reflect"
	"testing"
)

func TestParseOTLPHeaders(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		expected  map[string]string
		wantError bool
	}{
		{
			name:     "empty string",
			raw:      "",
			expected: map[string]string{},
		},
		{
			name: "json object",
			raw:  `{"Authorization":"Bearer token","X-Scope-OrgID":"tenant-a"}`,
			expected: map[string]string{
				"Authorization": "Bearer token",
				"X-Scope-OrgID": "tenant-a",
			},
		},
		{
			name: "key value csv",
			raw:  "Authorization=Bearer token,X-Scope-OrgID=tenant-a",
			expected: map[string]string{
				"Authorization": "Bearer token",
				"X-Scope-OrgID": "tenant-a",
			},
		},
		{
			name:      "invalid pair",
			raw:       "Authorization",
			wantError: true,
		},
		{
			name:      "invalid json",
			raw:       `{"Authorization":123}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseOTLPHeaders(tt.raw)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}
