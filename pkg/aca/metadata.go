package aca

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ACAMetadata holds discovered Azure environment metadata
type ACAMetadata struct {
	SubscriptionID string // Azure subscription ID
	ResourceGroup  string // Resource group name
	AppName        string // Container App name
	Location       string // Azure region
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

// DiscoverMetadata queries IMDS and parses the metadata
func DiscoverMetadata(ctx context.Context) (*ACAMetadata, error) {
	imdsResp, err := queryIMDS(ctx)
	if err != nil {
		return nil, err
	}

	metadata, err := parseMetadata(imdsResp)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}
