package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
	"github.com/nycon/qemu-cli/internal/resolve"
)

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage tenants",
	Example: `  qemu-cli tenant list
  qemu-cli tenant use Default
  qemu-cli tenant use 10c72d24`,
}

func init() {
	tenantCmd.AddCommand(tenantListCmd)
	tenantCmd.AddCommand(tenantUseCmd)
}

var tenantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/tenants", nil)
		if err != nil {
			return err
		}
		if handled, err := output.Raw(raw); handled {
			return err
		}
		items, err := api.AsList(raw)
		if err != nil {
			return err
		}
		current, _ := config.TenantID()
		rows := [][]string{}
		for _, it := range items {
			var t map[string]any
			_ = api.Decode(it, &t)
			id := fmt.Sprint(t["id"])
			marker := " "
			if id == current {
				marker = "*"
			}
			rows = append(rows, []string{marker + " " + shortID(id), fmt.Sprint(t["name"])})
		}
		return output.Table([]string{"ID", "Name"}, rows, "Tenants")
	},
}

var tenantUseCmd = &cobra.Command{
	Use:   "use [id|name]",
	Short: "Select the active tenant",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli tenant use Default`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		id, name, err := resolve.Tenant(c, args[0])
		if err != nil {
			return err
		}
		if err := config.SetTenantID(id); err != nil {
			return err
		}
		return output.Success(fmt.Sprintf("Active tenant: %s", name), map[string]any{
			"tenant_id": id,
			"name":      name,
		})
	},
}
