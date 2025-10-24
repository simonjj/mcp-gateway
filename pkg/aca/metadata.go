package aca

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ACAMetadata holds discovered Azure environment metadata
type ACAMetadata struct {
	SubscriptionID string // Azure subscription ID
	ResourceGroup  string // Resource group name
	AppName        string // Container App name
	Location       string // Azure region
}

// maskValue returns "[set]" if value is non-empty, "[NOT SET]" otherwise
func maskValue(value string) string {
	if value == "" {
		return "[NOT SET]"
	}
	// Show first 8 chars for readability (useful for debugging without exposing full values)
	if len(value) > 12 {
		return value[:8] + "..." + value[len(value)-4:]
	}
	return "[set]"
}

// imdsResponse represents the response from Azure Instance Metadata Service
type imdsResponse struct {
	Compute struct {
		SubscriptionID    string `json:"subscriptionId"`
		ResourceGroupName string `json:"resourceGroupName"`
		Name              string `json:"name"`
		Location          string `json:"location"`
		ResourceID        string `json:"resourceId"`
	} `json:"compute"`
}

// queryIMDS queries the Azure Instance Metadata Service to discover environment information
func queryIMDS(ctx context.Context) (*imdsResponse, error) {
	const imdsURL = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
	const maxRetries = 3
	const initialDelay = 1 * time.Second

	client := &http.Client{
		Timeout: 2 * time.Second, // Short timeout for local IMDS
	}

	var lastErr error
	delay := initialDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", imdsURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create IMDS request: %w", err)
		}

		req.Header.Add("Metadata", "true")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("IMDS request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("IMDS returned status %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, maxRetries, string(body))
			continue
		}

		var imdsResp imdsResponse
		if err := json.NewDecoder(resp.Body).Decode(&imdsResp); err != nil {
			lastErr = fmt.Errorf("failed to decode IMDS response (attempt %d/%d): %w", attempt+1, maxRetries, err)
			continue
		}

		return &imdsResp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("IMDS query failed after %d retries: %w", maxRetries, lastErr)
	}

	return nil, fmt.Errorf("IMDS query failed after %d retries", maxRetries)
}

// parseMetadata extracts metadata from IMDS response
func parseMetadata(imdsResp *imdsResponse) (*ACAMetadata, error) {
	if imdsResp == nil {
		return nil, fmt.Errorf("IMDS response is nil")
	}

	// Validate required fields
	if imdsResp.Compute.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription ID not found in IMDS response")
	}
	if imdsResp.Compute.ResourceGroupName == "" {
		return nil, fmt.Errorf("resource group not found in IMDS response")
	}
	if imdsResp.Compute.Name == "" {
		return nil, fmt.Errorf("app name not found in IMDS response")
	}

	metadata := &ACAMetadata{
		SubscriptionID: imdsResp.Compute.SubscriptionID,
		ResourceGroup:  imdsResp.Compute.ResourceGroupName,
		AppName:        imdsResp.Compute.Name,
		Location:       imdsResp.Compute.Location,
	}

	return metadata, nil
}

// DiscoverMetadata discovers Azure environment metadata from environment variables or IMDS
// Priority: 1) Environment variables (ACA standard), 2) IMDS (Azure VMs)
func DiscoverMetadata(ctx context.Context) (*ACAMetadata, error) {
	// Try environment variables first (ACA standard approach)
	metadata, err := discoverFromEnvVars()
	if err == nil {
		return metadata, nil
	}

	// Fallback to IMDS for Azure VMs (not typically available in ACA)
	imdsResp, err := queryIMDS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover metadata from environment variables and IMDS is not accessible: %w", err)
	}

	return parseMetadata(imdsResp)
}

// discoverFromEnvVars reads metadata from environment variables
// ACA automatically injects these, or they can be set manually
func discoverFromEnvVars() (*ACAMetadata, error) {
	// Check for explicit environment variables first
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP")
	appName := os.Getenv("AZURE_APP_NAME")

	// Log what we found (or didn't find)
	fmt.Fprintf(os.Stderr, "ACA Metadata Discovery: Checking environment variables...\n")
	fmt.Fprintf(os.Stderr, "  AZURE_SUBSCRIPTION_ID: %s\n", maskValue(subscriptionID))
	fmt.Fprintf(os.Stderr, "  AZURE_RESOURCE_GROUP: %s\n", maskValue(resourceGroup))
	fmt.Fprintf(os.Stderr, "  AZURE_APP_NAME: %s\n", maskValue(appName))

	// If not set explicitly, try to derive from CONTAINER_APP_NAME and other ACA env vars
	if appName == "" {
		// ACA sets CONTAINER_APP_NAME automatically
		appName = os.Getenv("CONTAINER_APP_NAME")
		fmt.Fprintf(os.Stderr, "  CONTAINER_APP_NAME: %s\n", maskValue(appName))
	}

	// Validate we have the minimum required fields
	var missingVars []string
	if subscriptionID == "" {
		missingVars = append(missingVars, "AZURE_SUBSCRIPTION_ID")
	}
	if resourceGroup == "" {
		missingVars = append(missingVars, "AZURE_RESOURCE_GROUP")
	}
	if appName == "" {
		missingVars = append(missingVars, "AZURE_APP_NAME or CONTAINER_APP_NAME")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	metadata := &ACAMetadata{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		AppName:        appName,
		Location:       os.Getenv("AZURE_LOCATION"), // Optional, may be empty
	}

	return metadata, nil
}
