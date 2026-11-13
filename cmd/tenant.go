package cmd

import (
	"fmt"
	"strings"

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
  qemu-cli tenant public-ips
  qemu-cli tenant set-email ops@example.com
  qemu-cli tenant assign-public-ips Default --allocation-id <uuid> --allocation-id <uuid>`,
}

var (
	tenantEmail          string
	tenantAllocIDs       []string
	tenantClearPublicIPs bool
)

func init() {
	tenantCmd.AddCommand(tenantListCmd)
	tenantCmd.AddCommand(tenantUseCmd)
	tenantCmd.AddCommand(tenantPublicIPsCmd)
	tenantCmd.AddCommand(tenantSetEmailCmd)
	tenantCmd.AddCommand(tenantAssignPublicIPsCmd)

	tenantSetEmailCmd.Flags().StringVar(&tenantEmail, "email", "", "Notification email (empty clears)")
	tenantAssignPublicIPsCmd.Flags().StringArrayVar(&tenantAllocIDs, "allocation-id", nil, "Public IP allocation UUID (repeatable)")
	tenantAssignPublicIPsCmd.Flags().BoolVar(&tenantClearPublicIPs, "clear", false, "Unassign all free public IPs from the tenant")
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
			email := "—"
			if t["notification_email"] != nil && fmt.Sprint(t["notification_email"]) != "" && fmt.Sprint(t["notification_email"]) != "<nil>" {
				email = fmt.Sprint(t["notification_email"])
			}
			rows = append(rows, []string{
				marker + " " + shortID(id),
				fmt.Sprint(t["name"]),
				fmt.Sprint(t["slug"]),
				fmt.Sprintf("%v", t["max_ips"]),
				email,
			})
		}
		return output.Table([]string{"ID", "Name", "Slug", "Max IPs", "Notify email"}, rows, "Tenants")
	},
}

var tenantUseCmd = &cobra.Command{
	Use:     "use [id|name]",
	Short:   "Select the active tenant",
	Args:    cobra.ExactArgs(1),
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

var tenantPublicIPsCmd = &cobra.Command{
	Use:   "public-ips",
	Short: "List public IPs assigned to the active tenant",
	Example: `  qemu-cli tenant public-ips
  qemu-cli -o json tenant public-ips`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		raw, err := c.Get("/tenants/"+tid+"/public-ips", nil)
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
			vm := "—"
			if a["vm_id"] != nil && fmt.Sprint(a["vm_id"]) != "" && fmt.Sprint(a["vm_id"]) != "<nil>" {
				vm = shortID(fmt.Sprint(a["vm_id"]))
			}
			rows = append(rows, []string{
				fmt.Sprint(a["id"]),
				fmt.Sprint(a["ip_address"]),
				fmt.Sprint(a["status"]),
				vm,
			})
		}
		return output.Table([]string{"Allocation ID", "IP", "Status", "VM"}, rows, "Assigned public IPs")
	},
}

var tenantSetEmailCmd = &cobra.Command{
	Use:   "set-email [email]",
	Short: "Set the tenant notification email for the active tenant",
	Args:  cobra.MaximumNArgs(1),
	Example: `  qemu-cli tenant set-email ops@example.com
  qemu-cli tenant set-email --email ops@example.com
  qemu-cli tenant set-email ""`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email := tenantEmail
		if len(args) == 1 {
			email = args[0]
		}
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		var bodyEmail any
		if strings.TrimSpace(email) == "" {
			bodyEmail = nil
		} else {
			bodyEmail = strings.TrimSpace(email)
		}
		raw, err := c.Patch("/tenants/"+tid+"/notification-email", map[string]any{
			"notification_email": bodyEmail,
		})
		if err != nil {
			return err
		}
		var t map[string]any
		_ = api.Decode(raw, &t)
		return output.Success("Tenant notification email updated", map[string]any{
			"tenant_id":          tid,
			"notification_email": t["notification_email"],
		})
	},
}

var tenantAssignPublicIPsCmd = &cobra.Command{
	Use:   "assign-public-ips [tenant]",
	Short: "Assign concrete public IPs to a tenant (superadmin)",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli tenant assign-public-ips Default --allocation-id <uuid> --allocation-id <uuid>
  qemu-cli tenant assign-public-ips Default --clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !tenantClearPublicIPs && len(tenantAllocIDs) == 0 {
			return fmt.Errorf("pass --allocation-id (repeatable) or --clear\n  qemu-cli tenant assign-public-ips %s --allocation-id <uuid>", args[0])
		}
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		id, name, err := resolve.Tenant(c, args[0])
		if err != nil {
			return err
		}
		// Keep IPs already on VMs; sync refuses to drop allocated addresses.
		currentRaw, err := c.Get("/tenants/"+id+"/public-ips", nil)
		if err != nil {
			return err
		}
		currentItems, err := api.AsList(currentRaw)
		if err != nil {
			return err
		}
		ids := []string{}
		seen := map[string]bool{}
		for _, it := range currentItems {
			var a map[string]any
			_ = api.Decode(it, &a)
			aid := fmt.Sprint(a["id"])
			if fmt.Sprint(a["status"]) == "allocated" {
				ids = append(ids, aid)
				seen[aid] = true
			}
		}
		if tenantClearPublicIPs {
			// only keep in-use allocations
		} else {
			for _, aid := range tenantAllocIDs {
				if !seen[aid] {
					ids = append(ids, aid)
					seen[aid] = true
				}
			}
		}
		raw, err := c.Patch("/tenants/"+id, map[string]any{
			"public_ip_allocation_ids": ids,
			"max_ips":                  len(ids),
		})
		if err != nil {
			return err
		}
		var t map[string]any
		_ = api.Decode(raw, &t)
		return output.Success(fmt.Sprintf("Public IPs synced for %s (%d)", name, len(ids)), map[string]any{
			"tenant_id":                id,
			"name":                     name,
			"public_ip_allocation_ids": ids,
			"max_ips":                  t["max_ips"],
		})
	},
}
