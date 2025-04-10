package core

import (
	"testing"
)

func TestGetPackageName(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{
			name: "interface with pointer",
			v:    tracer,
			want: "go.opentelemetry.io/otel/internal/global",
		},
		{
			name: "nil",
			v:    nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPackageName(tt.v); got != tt.want {
				t.Errorf("getPackageName() = %v, want %v", got, tt.want)
			}
		})
	}
}
