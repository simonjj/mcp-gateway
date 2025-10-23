package gateway

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	"github.com/docker/mcp-gateway/pkg/aca"
	"github.com/docker/mcp-gateway/pkg/docker"
)

// mockDockerClient is a minimal docker.Client implementation for testing
type mockDockerClient struct{}

func (m *mockDockerClient) ContainerExists(ctx context.Context, containerName string) (bool, container.InspectResponse, error) {
	return false, container.InspectResponse{}, nil
}

func (m *mockDockerClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return nil
}

func (m *mockDockerClient) StartContainer(ctx context.Context, containerID string, containerConfig container.Config, hostConfig container.HostConfig, networkingConfig network.NetworkingConfig) error {
	return nil
}

func (m *mockDockerClient) StopContainer(ctx context.Context, containerID string, timeout int) error {
	return nil
}

func (m *mockDockerClient) FindContainerByLabel(ctx context.Context, label string) (string, error) {
	return "", nil
}

func (m *mockDockerClient) FindAllContainersByLabel(ctx context.Context, label string) ([]string, error) {
	return nil, nil
}

func (m *mockDockerClient) InspectContainer(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return container.InspectResponse{}, nil
}

func (m *mockDockerClient) ReadLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockDockerClient) ImageExists(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (m *mockDockerClient) InspectImage(ctx context.Context, name string) (image.InspectResponse, error) {
	return image.InspectResponse{}, nil
}

func (m *mockDockerClient) PullImage(ctx context.Context, name string) error {
	return nil
}

func (m *mockDockerClient) PullImages(ctx context.Context, names ...string) error {
	return nil
}

func (m *mockDockerClient) CreateNetwork(ctx context.Context, name string, internal bool, labels map[string]string) error {
	return nil
}

func (m *mockDockerClient) RemoveNetwork(ctx context.Context, name string) error {
	return nil
}

func (m *mockDockerClient) ConnectNetwork(ctx context.Context, networkName, containerName, hostname string) error {
	return nil
}

func (m *mockDockerClient) InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	return volume.Volume{}, nil
}

func (m *mockDockerClient) ReadSecrets(ctx context.Context, names []string, lenient bool) (map[string]string, error) {
	return nil, nil
}

// Ensure mockDockerClient implements docker.Client interface at compile time
var _ docker.Client = (*mockDockerClient)(nil)

func TestNewRuntimeProvider_NoEnvVar(t *testing.T) {
	mockDocker := &mockDockerClient{}
	provider, err := NewRuntimeProvider("", mockDocker)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider but got nil")
	}

	// Verify it's a DockerProvider (type assertion)
	if _, ok := provider.(*DockerProvider); !ok {
		t.Errorf("expected DockerProvider when MCP_RUNTIME is unset, got %T", provider)
	}
}

func TestNewRuntimeProvider_DockerMode(t *testing.T) {
	mockDocker := &mockDockerClient{}
	provider, err := NewRuntimeProvider("docker", mockDocker)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider but got nil")
	}

	// Verify it's a DockerProvider
	if _, ok := provider.(*DockerProvider); !ok {
		t.Errorf("expected DockerProvider when MCP_RUNTIME=docker, got %T", provider)
	}
}

func TestNewRuntimeProvider_ACAMode(t *testing.T) {
	mockDocker := &mockDockerClient{}
	provider, err := NewRuntimeProvider("ACA", mockDocker)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if provider == nil {
		t.Fatal("expected provider but got nil")
	}

	// Verify it's an ACAProvider
	if _, ok := provider.(*aca.ACAProvider); !ok {
		t.Errorf("expected ACAProvider when MCP_RUNTIME=ACA, got %T", provider)
	}
}

func TestNewRuntimeProvider_InvalidMode(t *testing.T) {
	mockDocker := &mockDockerClient{}
	
	testCases := []string{"invalid", "kubernetes", "ecs", "DOCKER", "aca"}
	
	for _, mode := range testCases {
		t.Run("mode="+mode, func(t *testing.T) {
			provider, err := NewRuntimeProvider(mode, mockDocker)

			if err != nil {
				t.Fatalf("expected no error for invalid mode, got %v", err)
			}

			if provider == nil {
				t.Fatal("expected provider but got nil")
			}

			// Should default to DockerProvider for invalid/unknown modes
			if _, ok := provider.(*DockerProvider); !ok {
				t.Errorf("expected DockerProvider for invalid mode '%s', got %T", mode, provider)
			}
		})
	}
}
