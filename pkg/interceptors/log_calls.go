package interceptors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/docker/mcp-gateway/pkg/log"
)

func LogCallsMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// Only log tools/call method
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			start := time.Now()

			// Extract tool name from request
			var toolName string
			var arguments any

			// Try to extract from request
			if callReq, ok := req.(*mcp.CallToolRequest); ok && callReq.Params != nil {
				toolName = callReq.Params.Name
				arguments = callReq.Params.Arguments
			}

			label := toolName
			if label == "" {
				label = "<unknown>"
			}

			if toolName != "" {
				log.Logf("  - Calling tool %s with arguments: %s", toolName, argumentsToString(arguments))
			} else {
				log.Logf("  - Calling tool (unknown) with method: %s", method)
			}

			result, err := next(ctx, method, req)
			duration := time.Since(start)
			if err != nil {
				log.Logf("  ! Tool %s failed after %s: %v", label, duration, err)
				return result, err
			}

			summary := summarizeResult(result)
			log.Logf("  > Tool %s responded in %s: %s", label, duration, summary)

			return result, nil
		}
	}
}

func summarizeResult(result mcp.Result) string {
	if result == nil {
		return "<no result>"
	}
	switch v := result.(type) {
	case *mcp.CallToolResult:
		return summarizeCallToolResult(v)
	default:
		return fmt.Sprintf("%T", result)
	}
}

func summarizeCallToolResult(res *mcp.CallToolResult) string {
	if res == nil {
		return "CallToolResult<nil>"
	}
	status := "ok"
	if res.IsError {
		status = "error"
	}
	parts := make([]string, 0, len(res.Content)+2)
	for _, content := range res.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			text := truncateString(c.Text, 120)
			parts = append(parts, fmt.Sprintf("text(len=%d) %q", len([]rune(c.Text)), text))
		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("image %s (%d bytes)", safeMimeType(c.MIMEType), len(c.Data)))
		case *mcp.AudioContent:
			parts = append(parts, fmt.Sprintf("audio %s (%d bytes)", safeMimeType(c.MIMEType), len(c.Data)))
		case *mcp.ResourceLink:
			parts = append(parts, fmt.Sprintf("resource-link %s", truncateString(c.URI, 80)))
		case *mcp.EmbeddedResource:
			parts = append(parts, fmt.Sprintf("resource %s", summarizeEmbeddedResource(c)))
		default:
			parts = append(parts, fmt.Sprintf("%T", c))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "<no content>")
	}
	if res.StructuredContent != nil {
		structured := truncateString(argumentsToString(res.StructuredContent), 120)
		parts = append(parts, fmt.Sprintf("structured=%s", structured))
	}
	return fmt.Sprintf("status=%s content=[%s]", status, strings.Join(parts, " | "))
}

func summarizeEmbeddedResource(res *mcp.EmbeddedResource) string {
	if res == nil || res.Resource == nil {
		return "<empty>"
	}
	details := res.Resource.URI
	if details == "" {
		details = "<no uri>"
	}
	if res.Resource.Text != "" {
		details = fmt.Sprintf("%s text(len=%d)", details, len([]rune(res.Resource.Text)))
	} else if len(res.Resource.Blob) > 0 {
		details = fmt.Sprintf("%s blob(%d bytes)", details, len(res.Resource.Blob))
	}
	return truncateString(details, 120)
}

func truncateString(in string, limit int) string {
	runes := []rune(in)
	if len(runes) <= limit {
		return in
	}
	return string(runes[:limit]) + "..."
}

func safeMimeType(mime string) string {
	if strings.TrimSpace(mime) == "" {
		return "application/octet-stream"
	}
	return mime
}
