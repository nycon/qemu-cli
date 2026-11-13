package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
)

var ipPoolCmd = &cobra.Command{
	Use:   "ip-pool",
	Short: "List IP pools and free public IPs",
	Example: `  qemu-cli ip-pool list
  qemu-cli ip-pool available
  qemu-cli -o json ip-pool available`,
}

func init() {
	rootCmd.AddCommand(ipPoolCmd)
	ipPoolCmd.AddCommand(ipPoolListCmd)
	ipPoolCmd.AddCommand(ipPoolAvailableCmd)
}

var ipPoolListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List IP pools visible for the active tenant (or all for superadmin)",
	Example: `  qemu-cli ip-pool list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		q := url.Values{}
		if tid, err := config.TenantID(); err == nil && tid != "" {
			q.Set("tenant_id", tid)
		}
		raw, err := c.Get("/ip-pools", q)
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
		rows := [][]string{}
		for _, it := range items {
			var p map[string]any
			_ = api.Decode(it, &p)
			net := fmt.Sprint(p["network"])
			if net == "" {
				net = "—"
			}
			rows = append(rows, []string{
				shortID(fmt.Sprint(p["id"])),
				fmt.Sprint(p["name"]),
				fmt.Sprint(p["pool_type"]),
				net,
				fmt.Sprintf("%v", p["available_ips"]),
				fmt.Sprintf("%v", p["total_ips"]),
			})
		}
		return output.Table([]string{"ID", "Name", "Type", "Network", "Free", "Total"}, rows, "IP pools")
	},
}

var ipPoolAvailableCmd = &cobra.Command{
	Use:   "available",
	Short: "List free public IPs assignable to tenants (superadmin)",
	Example: `  qemu-cli ip-pool available
  qemu-cli -o json ip-pool available`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/ip-pools/public/available", nil)
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
		rows := [][]string{}
		for _, it := range items {
			var a map[string]any
			_ = api.Decode(it, &a)
			rows = append(rows, []string{
				fmt.Sprint(a["id"]),
				fmt.Sprint(a["ip_address"]),
				shortID(fmt.Sprint(a["pool_id"])),
				fmt.Sprint(a["status"]),
			})
		}
		return output.Table([]string{"Allocation ID", "IP", "Pool", "Status"}, rows, "Free public IPs")
	},
}
