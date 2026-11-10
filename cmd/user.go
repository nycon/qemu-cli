package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/output"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Example: `  qemu-cli user list
  qemu-cli -o json user list`,
}

func init() {
	userCmd.AddCommand(userListCmd)
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/users", nil)
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
			var u map[string]any
			_ = api.Decode(it, &u)
			totp := "no"
			if u["totp_enabled"] == true || u["is_2fa_enabled"] == true {
				totp = "yes"
			}
			active := "yes"
			if u["is_active"] == false {
				active = "no"
			}
			role := "-"
			if u["is_superadmin"] == true {
				role = "superadmin"
			} else if r, ok := u["role"].(string); ok && r != "" {
				role = r
			}
			rows = append(rows, []string{
				shortID(fmt.Sprint(u["id"])),
				fmt.Sprint(nilDash(u["email"])),
				fmt.Sprint(nilDash(u["full_name"])),
				role,
				totp,
				active,
			})
		}
		return output.Table([]string{"ID", "Email", "Name", "Role", "2FA", "Active"}, rows, "Users")
	},
}
