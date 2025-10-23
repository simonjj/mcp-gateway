package aca

import (
	"testing"
)

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name        string
		resp        *imdsResponse
		expectError bool
	}{
		{
			name: "valid metadata",
			resp: &imdsResponse{
				Compute: struct {
					SubscriptionID    string `json:"subscriptionId"`
					ResourceGroupName string `json:"resourceGroupName"`
					Name              string `json:"name"`
					Location          string `json:"location"`
					ResourceID        string `json:"resourceId"`
				}{
					SubscriptionID:    "12345678-1234-1234-1234-123456789012",
					ResourceGroupName: "test-rg",
					Name:              "test-app",
					Location:          "eastus",
				},
			},
			expectError: false,
		},
		{
			name:        "nil response",
			resp:        nil,
			expectError: true,
		},
		{
			name: "missing subscription ID",
			resp: &imdsResponse{
				Compute: struct {
					SubscriptionID    string `json:"subscriptionId"`
					ResourceGroupName string `json:"resourceGroupName"`
					Name              string `json:"name"`
					Location          string `json:"location"`
					ResourceID        string `json:"resourceId"`
				}{
					ResourceGroupName: "test-rg",
					Name:              "test-app",
				},
			},
			expectError: true,
		},
		{
			name: "missing resource group",
			resp: &imdsResponse{
				Compute: struct {
					SubscriptionID    string `json:"subscriptionId"`
					ResourceGroupName string `json:"resourceGroupName"`
					Name              string `json:"name"`
					Location          string `json:"location"`
					ResourceID        string `json:"resourceId"`
				}{
					SubscriptionID: "12345678-1234-1234-1234-123456789012",
					Name:           "test-app",
				},
			},
			expectError: true,
		},
		{
			name: "missing app name",
			resp: &imdsResponse{
				Compute: struct {
					SubscriptionID    string `json:"subscriptionId"`
					ResourceGroupName string `json:"resourceGroupName"`
					Name              string `json:"name"`
					Location          string `json:"location"`
					ResourceID        string `json:"resourceId"`
				}{
					SubscriptionID:    "12345678-1234-1234-1234-123456789012",
					ResourceGroupName: "test-rg",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := parseMetadata(tt.resp)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if metadata == nil {
					t.Fatal("expected metadata but got nil")
				}
				if metadata.SubscriptionID == "" {
					t.Error("subscription ID should not be empty")
				}
				if metadata.ResourceGroup == "" {
					t.Error("resource group should not be empty")
				}
				if metadata.AppName == "" {
					t.Error("app name should not be empty")
				}
			}
		})
	}
}
