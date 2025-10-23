package aca

import (
	"context"
	"testing"

	"github.com/docker/mcp-gateway/pkg/runtime"
)

func TestNewACAProvider(t *testing.T) {
	provider := NewACAProvider()

	if provider == nil {
		t.Fatal("expected provider but got nil")
	}

	if provider.initialized {
		t.Error("new provider should not be initialized")
	}
}

func TestProviderError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ProviderError
		expected string
	}{
		{
			name: "error with cause",
			err: &ProviderError{
				Type:    ErrorAuthentication,
				Message: "auth failed",
				Cause:   context.DeadlineExceeded,
			},
			expected: "authentication error: auth failed: context deadline exceeded",
		},
		{
			name: "error without cause",
			err: &ProviderError{
				Type:    ErrorPermission,
				Message: "permission denied",
			},
			expected: "permission error: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected error '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestProviderNotInitialized(t *testing.T) {
	provider := NewACAProvider()
	ctx := context.Background()

	// Test that methods fail when not initialized
	err := provider.Cleanup(ctx)
	if err == nil {
		t.Error("Cleanup should fail when not initialized")
	}

	err = provider.EnableServer(ctx, runtime.ServerConfig{})
	if err == nil {
		t.Error("EnableServer should fail when not initialized")
	}

	err = provider.DisableServer(ctx, "test")
	if err == nil {
		t.Error("DisableServer should fail when not initialized")
	}

	_, err = provider.ListServers(ctx)
	if err == nil {
		t.Error("ListServers should fail when not initialized")
	}

	_, err = provider.GetServerStatus(ctx, "test")
	if err == nil {
		t.Error("GetServerStatus should fail when not initialized")
	}
}
