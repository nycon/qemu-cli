// Package resolve maps partial IDs and names to full resource UUIDs.
package resolve

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/nycon/qemu-cli/internal/api"
)

type named struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func match(partial string, items []named) (string, error) {
	partialLower := strings.ToLower(partial)
	for _, it := range items {
		if it.ID == partial || strings.HasPrefix(it.ID, partial) || strings.EqualFold(it.Name, partialLower) {
			return it.ID, nil
		}
	}
	return "", fmt.Errorf("not found: %s", partial)
}

func decodeNamed(raw json.RawMessage) ([]named, error) {
	items, err := api.AsList(raw)
	if err != nil {
		return nil, err
	}
	out := make([]named, 0, len(items))
	for _, it := range items {
		var n named
		if err := json.Unmarshal(it, &n); err != nil {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

// VM resolves a VM by UUID, prefix, or name within a tenant.
func VM(c *api.Client, tenantID, partial string) (string, error) {
	if len(partial) >= 32 {
		return partial, nil
	}
	q := url.Values{"per_page": {"200"}}
	raw, err := c.Get("/tenants/"+tenantID+"/vms", q)
	if err != nil {
		return "", err
	}
	items, err := decodeNamed(raw)
	if err != nil {
		return "", err
	}
	id, err := match(partial, items)
	if err != nil {
		return "", fmt.Errorf("VM not found: %s\n  qemu-cli vm list", partial)
	}
	return id, nil
}

// Image resolves a cloud image by UUID, prefix, or name.
func Image(c *api.Client, partial string) (string, error) {
	if len(partial) >= 32 {
		return partial, nil
	}
	raw, err := c.Get("/cloud-images", nil)
	if err != nil {
		return "", err
	}
	items, err := decodeNamed(raw)
	if err != nil {
		return "", err
	}
	id, err := match(partial, items)
	if err != nil {
		return "", fmt.Errorf("image not found: %s\n  qemu-cli image list", partial)
	}
	return id, nil
}

// Tenant resolves a tenant by UUID, prefix, or name.
func Tenant(c *api.Client, partial string) (id, name string, err error) {
	raw, err := c.Get("/tenants", nil)
	if err != nil {
		return "", "", err
	}
	items, err := decodeNamed(raw)
	if err != nil {
		return "", "", err
	}
	partialLower := strings.ToLower(partial)
	for _, it := range items {
		if it.ID == partial || strings.HasPrefix(it.ID, partial) || strings.EqualFold(it.Name, partialLower) {
			return it.ID, it.Name, nil
		}
	}
	return "", "", fmt.Errorf("tenant not found: %s\n  qemu-cli tenant list", partial)
}
