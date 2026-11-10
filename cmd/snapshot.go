package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
	"github.com/nycon/qemu-cli/internal/resolve"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage VM snapshots",
	Example: `  qemu-cli snapshot list my-vm
  qemu-cli snapshot create my-vm --name pre-upgrade
  qemu-cli snapshot restore my-vm pre-upgrade
  qemu-cli snapshot delete my-vm pre-upgrade`,
}

func init() {
	snapshotCmd.AddCommand(snapListCmd)
	snapshotCmd.AddCommand(snapCreateCmd)
	snapshotCmd.AddCommand(snapRestoreCmd)
	snapshotCmd.AddCommand(snapDeleteCmd)
}

var snapListCmd = &cobra.Command{
	Use:   "list [vm]",
	Short: "List snapshots of a VM",
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
		raw, err := c.Get("/tenants/"+tid+"/vms/"+vmID+"/snapshots", nil)
		if err != nil {
			return err
		}
		items, err := api.AsList(raw)
		if err != nil {
			return err
		}
		rows := [][]string{}
		for _, it := range items {
			var s map[string]any
			_ = api.Decode(it, &s)
			rows = append(rows, []string{
				fmt.Sprint(s["name"]),
				fmt.Sprint(nilDash(s["state"])),
				trunc(fmt.Sprint(s["created_at"]), 19),
			})
		}
		return output.Table([]string{"Name", "State", "Created"}, rows, "Snapshots")
	},
}

var snapName string

var snapCreateCmd = &cobra.Command{
	Use:   "create [vm]",
	Short: "Create a snapshot",
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
		body := map[string]any{}
		if snapName != "" {
			body["name"] = snapName
		}
		if _, err := c.Post("/tenants/"+tid+"/vms/"+vmID+"/snapshots", body); err != nil {
			return err
		}
		return output.Success("Snapshot created", map[string]any{"vm_id": vmID, "name": snapName})
	},
}

func init() {
	snapCreateCmd.Flags().StringVarP(&snapName, "name", "n", "", "Snapshot name")
}

var snapRestoreCmd = &cobra.Command{
	Use:   "restore [vm] [name]",
	Short: "Restore a snapshot",
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
		if _, err := c.Post("/tenants/"+tid+"/vms/"+vmID+"/snapshots/"+args[1]+"/restore", nil); err != nil {
			return err
		}
		return output.Success("Snapshot restored", map[string]any{"vm_id": vmID, "name": args[1]})
	},
}

var snapDeleteCmd = &cobra.Command{
	Use:   "delete [vm] [name]",
	Short: "Delete a snapshot",
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
		if _, err := c.Delete("/tenants/" + tid + "/vms/" + vmID + "/snapshots/" + args[1]); err != nil {
			return err
		}
		return output.Success("Snapshot deleted", map[string]any{"vm_id": vmID, "name": args[1]})
	},
}
