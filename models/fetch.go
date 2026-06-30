package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Data source URLs.
const (
	modelsDevURL = "https://models.dev/api.json"
	litellmURL   = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
)

// fetchOptions controls how source data is retrieved and cached.
type fetchOptions struct {
	refresh bool          // ignore any cached copy and re-download
	noCache bool          // do not read or write the cache at all
	ttl     time.Duration // maximum age of a cached copy before it is refreshed
}

func defaultFetchOptions() fetchOptions {
	return fetchOptions{ttl: 24 * time.Hour}
}

// cacheDir returns the directory used to cache downloaded source data.
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "models")
}

// fetchJSON returns the raw bytes for url, named name in the cache. Fresh cache
// entries (younger than opts.ttl) are returned without a network request. On a
// network failure, a stale cache entry is used as a fallback when available.
func fetchJSON(url, name string, opts fetchOptions) ([]byte, error) {
	cachePath := filepath.Join(cacheDir(), name)

	if !opts.noCache && !opts.refresh {
		if data, ok := readFreshCache(cachePath, opts.ttl); ok {
			return data, nil
		}
	}

	data, err := download(url)
	if err != nil {
		if !opts.noCache {
			if cached, rerr := os.ReadFile(cachePath); rerr == nil {
				fmt.Fprintf(os.Stderr, "warning: %s download failed (%v); using cached copy\n", name, err)
				return cached, nil
			}
		}
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}

	if !opts.noCache {
		writeCache(cachePath, data)
	}
	return data, nil
}

func readFreshCache(path string, ttl time.Duration) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func writeCache(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	// Write atomically so a partial write never corrupts the cache.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
