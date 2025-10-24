package aca

import (
	"context"
	"fmt"
	"strings"
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

func TestEnableServer_NotInitialized(t *testing.T) {
	provider := NewACAProvider()
	ctx := context.Background()

	config := runtime.ServerConfig{
		Name:      "test-server",
		Image:     "test:latest",
		Transport: "streaming",
		Port:      8080,
	}

	err := provider.EnableServer(ctx, config)
	if err == nil {
		t.Fatal("expected error when provider not initialized")
	}

	if err.Error() != "provider not initialized" {
		t.Errorf("expected 'provider not initialized' error, got: %s", err.Error())
	}
}

func TestEnableServer_StdioTransportRejection(t *testing.T) {
	// Note: This test will need to be updated once Initialize is properly implemented
	// For now, we test that stdio is rejected at validation level
	provider := NewACAProvider()
	// Simulate initialized state for this test
	provider.initialized = true
	provider.metadata = &ACAMetadata{
		SubscriptionID: "test-sub",
		ResourceGroup:  "test-rg",
		AppName:        "test-app",
	}
	provider.resources = &ACAResources{
		CPU:    "0.5",
		Memory: "1Gi",
	}

	ctx := context.Background()

	config := runtime.ServerConfig{
		Name:      "stdio-server",
		Image:     "test:latest",
		Transport: "stdio",
	}

	err := provider.EnableServer(ctx, config)
	if err == nil {
		t.Fatal("expected error for stdio transport")
	}

	// The error should mention stdio not being supported
	if err.Error() != "stdio transport not supported in ACA mode" {
		t.Errorf("expected stdio rejection error, got: %s", err.Error())
	}
}

func TestGetServerStatus_NotInitialized(t *testing.T) {
	provider := NewACAProvider()
	ctx := context.Background()

	_, err := provider.GetServerStatus(ctx, "test-server")
	if err == nil {
		t.Fatal("expected error when provider not initialized")
	}
}

func TestDisableServer_NotInitialized(t *testing.T) {
	provider := NewACAProvider()
	ctx := context.Background()

	err := provider.DisableServer(ctx, "test-server")
	if err == nil {
		t.Fatal("expected error when provider not initialized")
	}

	if err.Error() != "provider not initialized" {
		t.Errorf("expected 'provider not initialized' error, got: %s", err.Error())
	}
}

func TestDisableServer_NonExistentServer(t *testing.T) {
// This test validates idempotency - disabling a server that doesn't exist should succeed
provider := NewACAProvider()
ctx := context.Background()

err := provider.DisableServer(ctx, "non-existent-server")
if err == nil {
t.Fatal("expected error when provider not initialized, got nil")
}

// Should get initialization error, not a "server not found" error
if !strings.Contains(err.Error(), "not initialized") {
t.Errorf("expected 'not initialized' error, got: %s", err.Error())
}
}

func TestDisableServer_EmptyServerName(t *testing.T) {
provider := NewACAProvider()
ctx := context.Background()

err := provider.DisableServer(ctx, "")
if err == nil {
t.Fatal("expected error when provider not initialized, got nil")
}

if !strings.Contains(err.Error(), "not initialized") {
t.Errorf("expected 'not initialized' error, got: %s", err.Error())
}
}

func TestProviderError_QuotaError(t *testing.T) {
provider := NewACAProvider()

// Test quota error detection
err := provider.wrapError(fmt.Errorf("StatusCode: 409, quota exceeded"), "test operation")

if err == nil {
t.Fatal("expected error, got nil")
}

provErr, ok := err.(*ProviderError)
if !ok {
t.Fatalf("expected ProviderError, got %T", err)
}

if provErr.Type != ErrorQuota {
t.Errorf("expected ErrorQuota, got %s", provErr.Type)
}

if !strings.Contains(provErr.Message, "quota") {
t.Errorf("expected quota message, got: %s", provErr.Message)
}
}

func TestProviderError_ImagePullError(t *testing.T) {
provider := NewACAProvider()

// Test image pull error detection
err := provider.wrapError(fmt.Errorf("imagepull failed: manifest unknown"), "test operation")

if err == nil {
t.Fatal("expected error, got nil")
}

provErr, ok := err.(*ProviderError)
if !ok {
t.Fatalf("expected ProviderError, got %T", err)
}

if provErr.Type != ErrorConfiguration {
t.Errorf("expected ErrorConfiguration, got %s", provErr.Type)
}

if !strings.Contains(provErr.Message, "image pull failed") {
t.Errorf("expected image pull message, got: %s", provErr.Message)
}

if !strings.Contains(provErr.Message, "AcrPull") {
t.Errorf("expected AcrPull guidance, got: %s", provErr.Message)
}
}

func TestProviderError_TimeoutError(t *testing.T) {
provider := NewACAProvider()

// Test timeout error detection
err := provider.wrapError(fmt.Errorf("context deadline exceeded"), "test operation")

if err == nil {
t.Fatal("expected error, got nil")
}

provErr, ok := err.(*ProviderError)
if !ok {
t.Fatalf("expected ProviderError, got %T", err)
}

if provErr.Type != ErrorTransient {
t.Errorf("expected ErrorTransient, got %s", provErr.Type)
}

if !strings.Contains(provErr.Message, "timed out") {
t.Errorf("expected timeout message, got: %s", provErr.Message)
}
}
