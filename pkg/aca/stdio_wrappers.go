package aca

// StdioToSSEWrappers maps stdio MCP server images to their SSE-wrapped equivalents
// These wrappers use mcp-proxy to convert stdio transport to SSE for ACA compatibility
var StdioToSSEWrappers = map[string]string{
	// Top 5 most popular stdio servers (by pulls + stars*10)
	"mcp/fetch@sha256:ef9535a3f07249142f9ca5a6033d7024950afdb6dc05e98292794a23e9f5dfbe":               "simon.azurecr.io/mcp-fetch-sse@sha256:f0e7be3893a1603e9fdb5ddd4d302b6d17458b5e52536ebed10f76bfd90d68d4",
	"mcp/slack@sha256:4cc10c3f4bd988bd2dce40e3068fe38fa3b3bad1da99f9653eb5fa5cce35baa1":               "simon.azurecr.io/mcp-slack-sse@sha256:3f5c5c76b7709d2da1afc5be459ee1de4bcaaf119a585bd0a0cf2b4e5c970fc1",
	"mcp/time@sha256:9c46a918633fb474bf8035e3ee90ebac6bcf2b18ccb00679ac4c179cba0ebfcf":                "simon.azurecr.io/mcp-time-sse@sha256:61b8bbf653f84c400e5a1a2142948c6d394c40d16021aba8df79d7b33c9a143f",
	"mcp/postgres@sha256:b72c08f19e864b3e8beea8e90d048b9271924e704113345a29683f6d1e95a8f0":           "simon.azurecr.io/mcp-postgres-sse@sha256:5b9cff8cad33b3bcf4fb8b3d99db8971d371a04e50e5ca333207d465b6b3d047",
	"mcp/sequentialthinking@sha256:cd3174b2ecf37738654cf7671fb1b719a225c40a78274817da00c4241f465e5f": "simon.azurecr.io/mcp-sequentialthinking-sse@sha256:ea8e286c6c67d8a9b8ad421d084ce977e9714e8653e3f5a946039ccdfcec9f4f",

	// Additional common servers (add as wrappers are built)
	"mcp/duckduckgo@sha256:68eb20db6109f5c312a695fc5ec3386ad15d93ffb765a0b4eb1baf4328dec14f": "simon.azurecr.io/mcp-duckduckgo-sse@sha256:7ee139be30689a096798bbe80a2d2eec48e40670c623a806985e0d258038b0e2",
	"mcp/duckduckgo@sha256:68eb20db9ea07ba4494ef5c7b8d62c8cce064b2bb2e88f66d95ef9d73e0b4f5f": "simon.azurecr.io/mcp-duckduckgo-sse@sha256:7ee139be30689a096798bbe80a2d2eec48e40670c623a806985e0d258038b0e2",
}

// GetSSEWrapper returns the SSE wrapper image for a stdio server image, if available
// Returns the wrapper image and true if found, empty string and false otherwise
func GetSSEWrapper(stdioImage string) (string, bool) {
	wrapper, ok := StdioToSSEWrappers[stdioImage]
	return wrapper, ok
}

// HasSSEWrapper checks if a stdio server has an SSE wrapper available
func HasSSEWrapper(stdioImage string) bool {
	_, ok := StdioToSSEWrappers[stdioImage]
	return ok
}
