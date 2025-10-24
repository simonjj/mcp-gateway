package aca

import (
	"testing"

	"github.com/docker/mcp-gateway/pkg/runtime"
)

func TestBuildContainerSpec(t *testing.T) {
	resources := &ACAResources{
		CPU:    "0.5",
		Memory: "1Gi",
	}

	tests := []struct {
		name      string
		config    runtime.ServerConfig
		resources *ACAResources
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid streaming transport",
			config: runtime.ServerConfig{
				Name:      "test-server",
				Image:     "mcr.microsoft.com/mcp/test:latest",
				Transport: "streaming",
				Port:      8080,
				Env: map[string]string{
					"API_KEY": "test123",
					"DEBUG":   "true",
				},
			},
			resources: resources,
			wantErr:   false,
		},
		{
			name: "valid sse transport",
			config: runtime.ServerConfig{
				Name:      "sse-server",
				Image:     "mcr.microsoft.com/mcp/sse:latest",
				Transport: "sse",
				Port:      3000,
			},
			resources: resources,
			wantErr:   false,
		},
		{
			name: "stdio transport should fail",
			config: runtime.ServerConfig{
				Name:      "stdio-server",
				Image:     "mcr.microsoft.com/mcp/stdio:latest",
				Transport: "stdio",
			},
			resources: resources,
			wantErr:   true,
			errMsg:    "stdio transport not supported in ACA mode",
		},
		{
			name: "missing name",
			config: runtime.ServerConfig{
				Image:     "mcr.microsoft.com/mcp/test:latest",
				Transport: "streaming",
			},
			resources: resources,
			wantErr:   true,
			errMsg:    "server name is required",
		},
		{
			name: "missing image",
			config: runtime.ServerConfig{
				Name:      "test",
				Transport: "streaming",
			},
			resources: resources,
			wantErr:   true,
			errMsg:    "image is required",
		},
		{
			name: "invalid transport",
			config: runtime.ServerConfig{
				Name:      "test",
				Image:     "mcr.microsoft.com/mcp/test:latest",
				Transport: "invalid",
			},
			resources: resources,
			wantErr:   true,
			errMsg:    "transport must be 'streaming' or 'sse'",
		},
		{
			name: "default port",
			config: runtime.ServerConfig{
				Name:      "default-port",
				Image:     "mcr.microsoft.com/mcp/test:latest",
				Transport: "streaming",
				Port:      0, // Should default to 8080
			},
			resources: resources,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := buildContainerSpec(tt.config, tt.resources)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if spec == nil {
				t.Fatal("expected container spec but got nil")
			}

			// Verify naming convention
			expectedName := "mcp-server-" + tt.config.Name
			if *spec.Name != expectedName {
				t.Errorf("expected name '%s', got '%s'", expectedName, *spec.Name)
			}

			// Verify image
			if *spec.Image != tt.config.Image {
				t.Errorf("expected image '%s', got '%s'", tt.config.Image, *spec.Image)
			}

			// Verify resources applied
			if spec.Resources == nil {
				t.Fatal("expected resources but got nil")
			}

			// Verify environment variables
			if tt.config.Env != nil {
				if spec.Env == nil {
					t.Fatal("expected env vars but got nil")
				}
				// We add MCP_TRANSPORT automatically, so expect +1
				expectedCount := len(tt.config.Env) + 1
				if len(spec.Env) != expectedCount {
					t.Errorf("expected %d env vars (including MCP_TRANSPORT), got %d", expectedCount, len(spec.Env))
				}
			}
		})
	}
}

func TestContainerNamingConvention(t *testing.T) {
	tests := []struct {
		serverName   string
		expectedName string
	}{
		{"duckduckgo", "mcp-server-duckduckgo"},
		{"github", "mcp-server-github"},
		{"my-custom-server", "mcp-server-my-custom-server"},
	}

	resources := &ACAResources{CPU: "0.5", Memory: "1Gi"}

	for _, tt := range tests {
		t.Run(tt.serverName, func(t *testing.T) {
			config := runtime.ServerConfig{
				Name:      tt.serverName,
				Image:     "test:latest",
				Transport: "streaming",
				Port:      8080,
			}

			spec, err := buildContainerSpec(config, resources)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if *spec.Name != tt.expectedName {
				t.Errorf("expected name '%s', got '%s'", tt.expectedName, *spec.Name)
			}
		})
	}
}
