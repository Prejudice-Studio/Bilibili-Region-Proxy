package proxy

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const configFileName = "Bilibili-Region-Proxy.json"

type proxyConfig struct {
	forceIntlBangumiSearch bool
	injectionEnabled       bool
	injectionPosition      string
	injectionItems         []any
}

type configSignature struct {
	valid bool
	mtime int64
	size  int64
}

var (
	cfgMu        sync.Mutex
	cfgSignature configSignature
	cfgCache     = proxyConfig{injectionPosition: "tail"}

	// Set before the server starts accepting requests.
	settingsConfigPath = defaultConfigPath()
)

func configDirectory() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if containsConfig(dir) {
			return dir
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if containsConfig(wd) {
			return wd
		}
		return wd
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func containsConfig(dir string) bool {
	for _, name := range []string{configFileName, "config.json"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func defaultConfigPath() string {
	dir := configDirectory()
	for _, name := range []string{configFileName, "config.json"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return filepath.Join(dir, configFileName)
}

// SetConfigFile overrides the JSON config location. Call before serving requests.
func SetConfigFile(path string) {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	settingsConfigPath = path
}

// ConfigFile reports the JSON config path currently in use.
func ConfigFile() string { return settingsConfigPath }

func configPath() string { return settingsConfigPath }

func loadProxyConfig() proxyConfig {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	sig := configSignature{}
	info, err := os.Stat(configPath())
	if err != nil {
		cfgSignature = sig
		cfgCache = proxyConfig{injectionPosition: "tail"}
		return cfgCache
	}
	sig = configSignature{valid: true, mtime: info.ModTime().UnixNano(), size: info.Size()}
	if sig == cfgSignature {
		return cfgCache
	}

	cfgSignature = sig
	raw, err := os.ReadFile(configPath())
	if err != nil {
		cfgCache = proxyConfig{injectionPosition: "tail"}
		return cfgCache
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		log.Println("Proxy JSON config is invalid; optional search features are disabled")
		cfgCache = proxyConfig{injectionPosition: "tail"}
		return cfgCache
	}

	forceIntlBangumiSearch := root["force_intl_bangumi_search"] == true
	position := "tail"
	enabled := false
	var items []any
	if section, ok := root["injection"].(map[string]any); ok {
		if p, ok := section["position"].(string); ok && (strings.EqualFold(p, "head") || strings.EqualFold(p, "tail")) {
			position = strings.ToLower(p)
		}
		enabled = section["enabled"] == true
		if list, ok := section["items"].([]any); ok {
			items = list
		}
	}
	cfgCache = proxyConfig{
		forceIntlBangumiSearch: forceIntlBangumiSearch,
		injectionEnabled:       enabled,
		injectionPosition:      position,
		injectionItems:         items,
	}
	return cfgCache
}
