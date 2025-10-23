package aca

import (
	"fmt"
	"strconv"
	"strings"
)

// ACAResources holds discovered resource allocation
type ACAResources struct {
	CPU    string // e.g., "0.5" or "500m"
	Memory string // e.g., "1Gi" or "512Mi"
}

// parseResourceString validates and normalizes resource strings
func parseResourceString(resource, resourceType string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("%s resource is empty", resourceType)
	}

	// Resource is already in string format from ACA API
	// Just validate it's not obviously invalid
	if resourceType == "CPU" {
		// CPU can be "0.5" or "500m" format
		if strings.HasSuffix(resource, "m") {
			// Millicores format
			milli := strings.TrimSuffix(resource, "m")
			if _, err := strconv.Atoi(milli); err != nil {
				return "", fmt.Errorf("invalid CPU millicores format: %s", resource)
			}
		} else {
			// Cores format
			if _, err := strconv.ParseFloat(resource, 64); err != nil {
				return "", fmt.Errorf("invalid CPU cores format: %s", resource)
			}
		}
	} else if resourceType == "Memory" {
		// Memory should have unit suffix like "Gi" or "Mi"
		if !strings.HasSuffix(resource, "Gi") && !strings.HasSuffix(resource, "Mi") &&
			!strings.HasSuffix(resource, "G") && !strings.HasSuffix(resource, "M") {
			return "", fmt.Errorf("invalid memory format (missing unit): %s", resource)
		}
	}

	return resource, nil
}

// ParseResources validates resource limits
func ParseResources(cpu, memory string) (*ACAResources, error) {
	validCPU, err := parseResourceString(cpu, "CPU")
	if err != nil {
		return nil, err
	}

	validMemory, err := parseResourceString(memory, "Memory")
	if err != nil {
		return nil, err
	}

	return &ACAResources{
		CPU:    validCPU,
		Memory: validMemory,
	}, nil
}

// Default resource limits for MCP servers if discovery fails
func DefaultResources() *ACAResources {
	return &ACAResources{
		CPU:    "0.25",
		Memory: "0.5Gi",
	}
}
