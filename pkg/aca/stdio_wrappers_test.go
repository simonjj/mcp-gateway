package aca

import (
	"testing"
)

func TestGetSSEWrapper(t *testing.T) {
	// Initialize wrappers
	if err := InitWrappers(); err != nil {
		t.Fatalf("Failed to initialize wrappers: %v", err)
	}

	tests := []struct {
		name        string
		stdioImage  string
		wantSSE     string
		wantFound   bool
	}{
		{
			name:       "exact match - fetch",
			stdioImage: "mcp/fetch@sha256:ef9535a3f07249142f9ca5a6033d7024950afdb6dc05e98292794a23e9f5dfbe",
			wantSSE:    "simon.azurecr.io/mcp-fetch-sse@sha256:f0e7be3893a1603e9fdb5ddd4d302b6d17458b5e52536ebed10f76bfd90d68d4",
			wantFound:  true,
		},
		{
			name:       "exact match - slack",
			stdioImage: "mcp/slack@sha256:4cc10c3f4bd988bd2dce40e3068fe38fa3b3bad1da99f9653eb5fa5cce35baa1",
			wantSSE:    "simon.azurecr.io/mcp-slack-sse@sha256:3f5c5c76b7709d2da1afc5be459ee1de4bcaaf119a585bd0a0cf2b4e5c970fc1",
			wantFound:  true,
		},
		{
			name:       "regex match - duckduckgo variant 1",
			stdioImage: "mcp/duckduckgo@sha256:68eb20db6109f5c312a695fc5ec3386ad15d93ffb765a0b4eb1baf4328dec14f",
			wantSSE:    "simon.azurecr.io/mcp-duckduckgo-sse@sha256:7ee139be30689a096798bbe80a2d2eec48e40670c623a806985e0d258038b0e2",
			wantFound:  true,
		},
		{
			name:       "regex match - duckduckgo variant 2",
			stdioImage: "mcp/duckduckgo@sha256:68eb20db9ea07ba4494ef5c7b8d62c8cce064b2bb2e88f66d95ef9d73e0b4f5f",
			wantSSE:    "simon.azurecr.io/mcp-duckduckgo-sse@sha256:7ee139be30689a096798bbe80a2d2eec48e40670c623a806985e0d258038b0e2",
			wantFound:  true,
		},
		{
			name:       "regex match - duckduckgo different sha",
			stdioImage: "mcp/duckduckgo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantSSE:    "simon.azurecr.io/mcp-duckduckgo-sse@sha256:7ee139be30689a096798bbe80a2d2eec48e40670c623a806985e0d258038b0e2",
			wantFound:  true,
		},
		{
			name:       "no match - unknown server",
			stdioImage: "mcp/unknown@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantSSE:    "",
			wantFound:  false,
		},
		{
			name:       "no match - invalid format",
			stdioImage: "not-a-valid-image",
			wantSSE:    "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSSE, gotFound := GetSSEWrapper(tt.stdioImage)
			if gotFound != tt.wantFound {
				t.Errorf("GetSSEWrapper() found = %v, want %v", gotFound, tt.wantFound)
			}
			if gotSSE != tt.wantSSE {
				t.Errorf("GetSSEWrapper() SSE = %v, want %v", gotSSE, tt.wantSSE)
			}
		})
	}
}

func TestHasSSEWrapper(t *testing.T) {
	// Initialize wrappers
	if err := InitWrappers(); err != nil {
		t.Fatalf("Failed to initialize wrappers: %v", err)
	}

	tests := []struct {
		name       string
		stdioImage string
		want       bool
	}{
		{
			name:       "has wrapper - fetch",
			stdioImage: "mcp/fetch@sha256:ef9535a3f07249142f9ca5a6033d7024950afdb6dc05e98292794a23e9f5dfbe",
			want:       true,
		},
		{
			name:       "has wrapper - duckduckgo regex",
			stdioImage: "mcp/duckduckgo@sha256:1111111111111111111111111111111111111111111111111111111111111111",
			want:       true,
		},
		{
			name:       "no wrapper",
			stdioImage: "mcp/unknown@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSSEWrapper(tt.stdioImage); got != tt.want {
				t.Errorf("HasSSEWrapper() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitWrappers(t *testing.T) {
	// Test that initialization works without errors
	err := InitWrappers()
	if err != nil {
		t.Errorf("InitWrappers() error = %v", err)
	}

	// Test that calling it again doesn't cause issues (idempotent)
	err = InitWrappers()
	if err != nil {
		t.Errorf("InitWrappers() second call error = %v", err)
	}

	// Verify we have some wrappers loaded
	wrapperMutex.RLock()
	count := len(wrappers)
	wrapperMutex.RUnlock()

	if count == 0 {
		t.Error("InitWrappers() loaded 0 wrappers, expected at least 1")
	}

	t.Logf("Loaded %d wrappers", count)
}
