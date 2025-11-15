package aca

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGetSSEWrapper(t *testing.T) {
	remoteWrappers := fetchWrappersFromConfig(t)

	resetWrappersForTest(t)
	if err := InitWrappers(); err != nil {
		t.Fatalf("Failed to initialize wrappers: %v", err)
	}

	for _, entry := range remoteWrappers {
		stdioImage, ok := sampleStdioForEntry(entry)
		if !ok {
			t.Logf("skipping entry %q (unsupported pattern)", entry.Stdio)
			continue
		}

		entryCopy := entry
		stdioCopy := stdioImage

		t.Run(stdioCopy, func(t *testing.T) {
			expected := expectedSSEFromEntry(entryCopy, stdioCopy)

			got, found := GetSSEWrapper(stdioCopy)
			if !found {
				t.Fatalf("expected wrapper for %s, got none", stdioCopy)
			}
			if got != expected {
				t.Fatalf("wrapper mismatch for %s: got %s, want %s", stdioCopy, got, expected)
			}
		})
	}

	t.Run("missing entry", func(t *testing.T) {
		missing := "mcp/unknown@sha256:" + strings.Repeat("0", 64)
		if got, found := GetSSEWrapper(missing); found || got != "" {
			t.Fatalf("expected no wrapper for %s, got %q (found=%v)", missing, got, found)
		}
	})
}

func TestHasSSEWrapper(t *testing.T) {
	remoteWrappers := fetchWrappersFromConfig(t)

	resetWrappersForTest(t)
	if err := InitWrappers(); err != nil {
		t.Fatalf("Failed to initialize wrappers: %v", err)
	}

	for _, entry := range remoteWrappers {
		stdioImage, ok := sampleStdioForEntry(entry)
		if !ok {
			t.Logf("skipping entry %q (unsupported pattern)", entry.Stdio)
			continue
		}

		stdioCopy := stdioImage
		t.Run(stdioCopy, func(t *testing.T) {
			if !HasSSEWrapper(stdioCopy) {
				t.Fatalf("expected HasSSEWrapper() to be true for %s", stdioCopy)
			}
		})
	}

	t.Run("missing entry", func(t *testing.T) {
		missing := "mcp/unknown@sha256:" + strings.Repeat("1", 64)
		if HasSSEWrapper(missing) {
			t.Fatalf("expected HasSSEWrapper() to be false for %s", missing)
		}
	})
}

func TestInitWrappers(t *testing.T) {
	resetWrappersForTest(t)

	if err := InitWrappers(); err != nil {
		t.Fatalf("InitWrappers() error = %v", err)
	}

	if err := InitWrappers(); err != nil {
		t.Fatalf("InitWrappers() second call error = %v", err)
	}

	wrapperMutex.RLock()
	count := len(wrappers)
	wrapperMutex.RUnlock()

	if count == 0 {
		t.Fatal("InitWrappers() loaded 0 wrappers, expected at least 1")
	}

	t.Logf("Loaded %d wrappers", count)
}

func fetchWrappersFromConfig(t *testing.T) []WrapperEntry {
	t.Helper()

	var config WrapperConfig
	if err := yaml.Unmarshal(embeddedConfigYAML, &config); err != nil {
		t.Fatalf("failed to parse wrapper config: %v", err)
	}

	resp, err := http.Get(config.WrapperURL)
	if err != nil {
		t.Skipf("unable to fetch wrappers from %s: %v", config.WrapperURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("unable to fetch wrappers from %s: HTTP %d", config.WrapperURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read wrapper response: %v", err)
	}

	var wrappersData WrappersYAML
	if err := yaml.Unmarshal(body, &wrappersData); err != nil {
		t.Fatalf("failed to parse wrappers YAML: %v", err)
	}

	return compileWrapperEntries(t, wrappersData.Wrappers)
}

func compileWrapperEntries(t *testing.T, entries []WrapperEntry) []WrapperEntry {
	t.Helper()

	for i := range entries {
		matchType := entries[i].MatchType
		if matchType == "" {
			matchType = "exact"
			entries[i].MatchType = matchType
		}
		if matchType == "regex" {
			compiled, err := regexp.Compile(entries[i].Stdio)
			if err != nil {
				t.Fatalf("failed to compile regex %q: %v", entries[i].Stdio, err)
			}
			entries[i].compiledRegex = compiled
		}
	}
	return entries
}

func sampleStdioForEntry(entry WrapperEntry) (string, bool) {
	matchType := entry.MatchType
	if matchType == "" {
		matchType = "exact"
	}

	if matchType == "exact" {
		return entry.Stdio, true
	}

	if matchType != "regex" || entry.compiledRegex == nil {
		return "", false
	}

	raw := strings.TrimPrefix(entry.Stdio, "^")
	raw = strings.TrimSuffix(raw, "$")

	classQuantifier := regexp.MustCompile(`\[[^\]]+\]\{\d+(,\d+)?\}`)
	sample := classQuantifier.ReplaceAllStringFunc(raw, func(segment string) string {
		closing := strings.Index(segment, "]")
		if closing < 0 {
			return segment
		}

		class := segment[1:closing]
		quantifier := strings.Trim(segment[closing+1:], "{}")
		parts := strings.SplitN(quantifier, ",", 2)

		count, err := strconv.Atoi(parts[0])
		if err != nil || count <= 0 {
			return segment
		}

		return strings.Repeat(sampleCharFromClass(class), count)
	})

	sample = strings.ReplaceAll(sample, `\.`, ".")
	sample = strings.ReplaceAll(sample, `\-`, "-")

	if !entry.compiledRegex.MatchString(sample) {
		return "", false
	}

	return sample, true
}

func sampleCharFromClass(class string) string {
	if class == "" {
		return "a"
	}

	if strings.HasPrefix(class, "\\") && len(class) >= 2 {
		return class[1:2]
	}

	if dash := strings.Index(class, "-"); dash > 0 {
		return class[:1]
	}

	return class[:1]
}

func expectedSSEFromEntry(entry WrapperEntry, stdio string) string {
	matchType := entry.MatchType
	if matchType == "" {
		matchType = "exact"
	}
	if matchType == "regex" && entry.compiledRegex != nil {
		return entry.compiledRegex.ReplaceAllString(stdio, entry.SSE)
	}
	return entry.SSE
}

func resetWrappersForTest(t *testing.T) {
	t.Helper()

	wrapperMutex.Lock()
	wrappers = nil
	initialized = false
	wrapperMutex.Unlock()
}
