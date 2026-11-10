// Package config stores multi-server contexts in ~/.qemu-cli/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirName  = ".qemu-cli"
	FileName = "config.json"
)

// Context is one panel connection (URL + API key + optional tenant).
type Context struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	TenantID  string `json:"tenant_id,omitempty"`
	Insecure  bool   `json:"insecure,omitempty"`
}

// File is the on-disk configuration.
type File struct {
	ActiveContext string             `json:"active_context"`
	Contexts      map[string]Context `json:"contexts"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName, FileName), nil
}

func ensureDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, DirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// Load reads ~/.qemu-cli/config.json (empty config on missing/invalid file).
func Load() (*File, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{Contexts: map[string]Context{}}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return &File{Contexts: map[string]Context{}}, nil
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &File{Contexts: map[string]Context{}}, nil
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]Context{}
	}
	return &cfg, nil
}

// Save writes config with mode 0600.
func Save(cfg *File) error {
	if _, err := ensureDir(); err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

// Active returns the active context or an actionable error.
func Active() (name string, ctx Context, err error) {
	cfg, err := Load()
	if err != nil {
		return "", Context{}, err
	}
	name = cfg.ActiveContext
	if name == "" {
		return "", Context{}, fmt.Errorf("no active context\n  qemu-cli context create <name> --url <url> --token <qi_...>")
	}
	ctx, ok := cfg.Contexts[name]
	if !ok {
		return "", Context{}, fmt.Errorf("context %q not found\n  qemu-cli context list", name)
	}
	return name, ctx, nil
}

// ActiveName returns the active context name (may be empty).
func ActiveName() string {
	cfg, err := Load()
	if err != nil {
		return ""
	}
	return cfg.ActiveContext
}

// TenantID returns the tenant for the active context.
func TenantID() (string, error) {
	_, ctx, err := Active()
	if err != nil {
		return "", err
	}
	if ctx.TenantID == "" {
		return "", fmt.Errorf("no tenant selected\n  qemu-cli tenant list\n  qemu-cli tenant use <id|name>")
	}
	return ctx.TenantID, nil
}

// SetTenantID updates tenant_id on the active context.
func SetTenantID(tenantID string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	name := cfg.ActiveContext
	if name == "" {
		return fmt.Errorf("no active context\n  qemu-cli context create <name> --url <url> --token <qi_...>")
	}
	ctx, ok := cfg.Contexts[name]
	if !ok {
		return fmt.Errorf("context %q not found", name)
	}
	ctx.TenantID = tenantID
	cfg.Contexts[name] = ctx
	return Save(cfg)
}

// SetInsecure updates insecure TLS flag on the active context.
func SetInsecure(insecure bool) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	name := cfg.ActiveContext
	if name == "" {
		return fmt.Errorf("no active context")
	}
	ctx, ok := cfg.Contexts[name]
	if !ok {
		return fmt.Errorf("context %q not found", name)
	}
	ctx.Insecure = insecure
	cfg.Contexts[name] = ctx
	return Save(cfg)
}
