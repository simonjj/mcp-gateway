package aca

import (
	"testing"
)

func TestParseResources(t *testing.T) {
	tests := []struct {
		name        string
		cpu         string
		memory      string
		expectError bool
	}{
		{
			name:        "valid cores and Gi",
			cpu:         "0.5",
			memory:      "1Gi",
			expectError: false,
		},
		{
			name:        "valid millicores and Mi",
			cpu:         "500m",
			memory:      "512Mi",
			expectError: false,
		},
		{
			name:        "invalid CPU format",
			cpu:         "invalid",
			memory:      "1Gi",
			expectError: true,
		},
		{
			name:        "invalid memory format",
			cpu:         "0.5",
			memory:      "1",
			expectError: true,
		},
		{
			name:        "empty CPU",
			cpu:         "",
			memory:      "1Gi",
			expectError: true,
		},
		{
			name:        "empty memory",
			cpu:         "0.5",
			memory:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := ParseResources(tt.cpu, tt.memory)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resources == nil {
					t.Errorf("expected resources but got nil")
				}
				if resources.CPU != tt.cpu {
					t.Errorf("expected CPU %s, got %s", tt.cpu, resources.CPU)
				}
				if resources.Memory != tt.memory {
					t.Errorf("expected Memory %s, got %s", tt.memory, resources.Memory)
				}
			}
		})
	}
}

func TestDefaultResources(t *testing.T) {
	resources := DefaultResources()

	if resources == nil {
		t.Fatal("expected default resources but got nil")
	}

	if resources.CPU == "" {
		t.Error("default CPU should not be empty")
	}

	if resources.Memory == "" {
		t.Error("default Memory should not be empty")
	}
}
