package generate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nguyentin05/spectest/internal/domain"
)

func TestParseAPIFlags(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    []domain.APIInput
		wantErr string
	}{
		{
			name:   "preserves input order",
			values: []string{"auth=openapi-auth.yaml", "order=openapi-order.yaml"},
			want: []domain.APIInput{
				{Server: "auth", Path: "openapi-auth.yaml"},
				{Server: "order", Path: "openapi-order.yaml"},
			},
		},
		{
			name:   "splits at first equals sign",
			values: []string{"auth=a=b.yaml"},
			want:   []domain.APIInput{{Server: "auth", Path: "a=b.yaml"}},
		},
		{
			name:    "missing equals sign",
			values:  []string{"auth"},
			wantErr: "expected <server>=<path>",
		},
		{
			name:    "empty server name",
			values:  []string{"=openapi.yaml"},
			wantErr: "server name must match",
		},
		{
			name:    "server name starts with uppercase",
			values:  []string{"Auth=openapi.yaml"},
			wantErr: "server name must match",
		},
		{
			name:    "server name starts with digit",
			values:  []string{"1auth=openapi.yaml"},
			wantErr: "server name must match",
		},
		{
			name:    "server name with dot",
			values:  []string{"auth.v1=openapi.yaml"},
			wantErr: "server name must match",
		},
		{
			name:    "empty path",
			values:  []string{"auth="},
			wantErr: "path must not be empty",
		},
		{
			name:    "duplicate server",
			values:  []string{"auth=a.yaml", "auth=b.yaml"},
			wantErr: `duplicate --api server "auth"`,
		},
		{
			name:    "empty list",
			values:  nil,
			wantErr: "at least one --api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAPIFlags(tt.values)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseAPIFlags(%v) = %v, want error containing %q", tt.values, got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAPIFlags(%v) unexpected error: %v", tt.values, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAPIFlags(%v) = %#v, want %#v", tt.values, got, tt.want)
			}
		})
	}
}
