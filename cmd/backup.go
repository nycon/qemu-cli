package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
	"github.com/nycon/qemu-cli/internal/resolve"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage VM backups",
	Example: `  qemu-cli backup list my-vm
  qemu-cli backup create my-vm
  qemu-cli backup restore my-vm <backup-id> --yes
  qemu-cli backup delete my-vm <backup-id>`,
}

func init() {
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
}

var backupListCmd = &cobra.Command{
	Use:   "list [vm]",
	Short: "List backups of a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		vmID, err := resolve.VM(c, tid, args[0])
		if err != nil {
			return err
		}
		raw, err := c.Get("/tenants/"+tid+"/vms/"+vmID+"/backups", nil)
		if err != nil {
			return err
		}
		items, err := api.AsList(raw)
		if err != nil {
			return err
		}
		rows := [][]string{}
		for _, it := range items {
			var b map[string]any
			_ = api.Decode(it, &b)
			size := 0.0
			if v, ok := b["file_size_bytes"].(float64); ok {
				size = v / (1024 * 1024)
			}
			rows = append(rows, []string{
				shortID(fmt.Sprint(b["id"])),
				fmt.Sprintf("%.1f MB", size),
				fmt.Sprint(nilDash(b["status"])),
				trunc(fmt.Sprint(b["created_at"]), 19),
			})
		}
		return output.Table([]string{"ID", "Size", "Status", "Created"}, rows, "Backups")
	},
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [vm]",
	Short: "Create a full disk backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		vmID, err := resolve.VM(c, tid, args[0])
		if err != nil {
			return err
		}
		if _, err := c.Post("/tenants/"+tid+"/vms/"+vmID+"/backups", nil); err != nil {
			return err
		}
		return output.Success("Backup task started", map[string]any{"vm_id": vmID})
	},
}

var backupRestoreYes bool

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [vm] [backup-id]",
	Short: "Restore a VM from backup",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := confirm("This will overwrite the current disk. Continue?", backupRestoreYes); err != nil {
			return err
		}
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		vmID, err := resolve.VM(c, tid, args[0])
		if err != nil {
			return err
		}
		if _, err := c.Post("/tenants/"+tid+"/vms/"+vmID+"/backups/"+args[1]+"/restore", nil); err != nil {
			return err
		}
		return output.Success("Restore task started", map[string]any{"vm_id": vmID, "backup_id": args[1]})
	},
}

func init() {
	backupRestoreCmd.Flags().BoolVarP(&backupRestoreYes, "yes", "y", false, "Skip confirmation")
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete [vm] [backup-id]",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		vmID, err := resolve.VM(c, tid, args[0])
		if err != nil {
			return err
		}
		if _, err := c.Delete("/tenants/" + tid + "/vms/" + vmID + "/backups/" + args[1]); err != nil {
			return err
		}
		return output.Success("Backup deleted", map[string]any{"vm_id": vmID, "backup_id": args[1]})
	},
}
