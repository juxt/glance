package main

import (
	"os"
	"path/filepath"
)

func cacheDir() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "glance", "captures")
	}
	d, _ := os.UserCacheDir()
	return filepath.Join(d, "glance", "captures")
}

func configDir() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "glance")
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "glance")
}

func configPath() string {
	return filepath.Join(configDir(), "presets.csv")
}

func capturePath(id string) string {
	return filepath.Join(cacheDir(), id+".txt")
}

func ensureCacheDir() error {
	return os.MkdirAll(cacheDir(), 0o755)
}

func ensureConfigDir() error {
	return os.MkdirAll(configDir(), 0o755)
}
