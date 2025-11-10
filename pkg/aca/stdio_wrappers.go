package aca

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed stdio_wrappers.yaml
var embeddedWrappersYAML []byte

//go:embed wrapper_config.yaml
var embeddedConfigYAML []byte

// WrapperEntry represents a single stdio-to-SSE wrapper mapping
type WrapperEntry struct {
	Stdio       string `yaml:"stdio"`
	SSE         string `yaml:"sse"`
	MatchType   string `yaml:"match_type"`   // "exact" or "regex"
	Description string `yaml:"description"`  // optional
	compiledRegex *regexp.Regexp // cached compiled regex for regex match types
}

// WrapperConfig represents the configuration for wrapper mappings
type WrapperConfig struct {
	WrapperURL   string `yaml:"wrapper_url"`
	FetchTimeout int    `yaml:"fetch_timeout"`
}

// WrappersYAML represents the structure of the wrappers YAML file
type WrappersYAML struct {
	Wrappers []WrapperEntry `yaml:"wrappers"`
}

var (
	wrappers     []WrapperEntry
	wrapperMutex sync.RWMutex
	initialized  bool
)

// InitWrappers loads wrapper mappings from online source with fallback to local
func InitWrappers() error {
	wrapperMutex.Lock()
	defer wrapperMutex.Unlock()

	if initialized {
		return nil
	}

	// Load config
	var config WrapperConfig
	if err := yaml.Unmarshal(embeddedConfigYAML, &config); err != nil {
		return fmt.Errorf("failed to parse wrapper config: %w", err)
	}

	// Try to fetch online version first
	var yamlData []byte
	var source string

	if config.WrapperURL != "" {
		fmt.Printf("- Fetching stdio wrapper mappings from: %s\n", config.WrapperURL)
		
		timeout := time.Duration(config.FetchTimeout) * time.Second
		if config.FetchTimeout == 0 {
			timeout = 10 * time.Second
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(config.WrapperURL)
		
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			yamlData, err = io.ReadAll(resp.Body)
			if err == nil {
				source = "online"
				fmt.Println("- Successfully fetched online wrapper mappings")
			}
		} else if err != nil {
			fmt.Printf("- Failed to fetch online wrappers: %v\n", err)
		} else {
			fmt.Printf("- Failed to fetch online wrappers: HTTP %d\n", resp.StatusCode)
		}
	}

	// Fall back to embedded local version
	if yamlData == nil {
		fmt.Println("- Using embedded local wrapper mappings")
		yamlData = embeddedWrappersYAML
		source = "embedded"
	}

	// Parse YAML
	var wrappersData WrappersYAML
	if err := yaml.Unmarshal(yamlData, &wrappersData); err != nil {
		return fmt.Errorf("failed to parse wrappers YAML: %w", err)
	}

	// Compile regex patterns
	for i := range wrappersData.Wrappers {
		entry := &wrappersData.Wrappers[i]
		if entry.MatchType == "regex" {
			compiled, err := regexp.Compile(entry.Stdio)
			if err != nil {
				fmt.Printf("- Warning: invalid regex pattern '%s': %v\n", entry.Stdio, err)
				continue
			}
			entry.compiledRegex = compiled
		}
	}

	wrappers = wrappersData.Wrappers
	initialized = true

	// Print loaded mappings
	fmt.Printf("- Loaded %d stdio-to-SSE wrapper mappings (source: %s):\n", len(wrappers), source)
	for _, entry := range wrappers {
		if entry.MatchType == "regex" {
			fmt.Printf("  [regex] %s -> %s\n", entry.Stdio, entry.SSE)
		} else {
			fmt.Printf("  [exact] %s -> %s\n", entry.Stdio, entry.SSE)
		}
	}

	return nil
}

// GetSSEWrapper returns the SSE wrapper image for a stdio server image, if available
// Returns the wrapper image and true if found, empty string and false otherwise
func GetSSEWrapper(stdioImage string) (string, bool) {
	wrapperMutex.RLock()
	defer wrapperMutex.RUnlock()

	if !initialized {
		// Auto-initialize if not already done
		wrapperMutex.RUnlock()
		if err := InitWrappers(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize wrappers: %v\n", err)
			wrapperMutex.RLock()
			return "", false
		}
		wrapperMutex.RLock()
	}

	// Check exact matches first
	for _, entry := range wrappers {
		if entry.MatchType == "exact" && entry.Stdio == stdioImage {
			return entry.SSE, true
		}
	}

	// Then check regex matches
	for _, entry := range wrappers {
		if entry.MatchType == "regex" && entry.compiledRegex != nil {
			if entry.compiledRegex.MatchString(stdioImage) {
				// Support capture group substitution
				replaced := entry.compiledRegex.ReplaceAllString(stdioImage, entry.SSE)
				return replaced, true
			}
		}
	}

	return "", false
}

// HasSSEWrapper checks if a stdio server has an SSE wrapper available
func HasSSEWrapper(stdioImage string) bool {
	_, ok := GetSSEWrapper(stdioImage)
	return ok
}
