package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/output"
)

var firewallCmd = &cobra.Command{
	Use:   "firewall",
	Short: "IP blacklist & Fail2Ban",
	Example: `  qemu-cli firewall list
  qemu-cli firewall ban 1.2.3.4 --reason "Portscan"
  qemu-cli firewall unban 1.2.3.4
  qemu-cli firewall set-jail sshd --bantime 86400`,
}

func init() {
	firewallCmd.AddCommand(fwListCmd)
	firewallCmd.AddCommand(fwBanCmd)
	firewallCmd.AddCommand(fwUnbanCmd)
	firewallCmd.AddCommand(fwCheckCmd)
	firewallCmd.AddCommand(fwJailsCmd)
	firewallCmd.AddCommand(fwSetJailCmd)
	firewallCmd.AddCommand(fwLogCmd)
}

var fwListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all banned IPs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/system/security/blacklist", nil)
		if err != nil {
			return err
		}
		var data struct {
			ManualBans  []map[string]any `json:"manual_bans"`
			Fail2BanBans []map[string]any `json:"fail2ban_bans"`
		}
		if err := api.Decode(raw, &data); err != nil {
			return err
		}
		rows := [][]string{}
		for _, e := range data.ManualBans {
			ptype := "Temporary"
			if e["permanent"] == true {
				ptype = "Permanent"
			}
			rows = append(rows, []string{
				fmt.Sprint(e["ip"]),
				"Manual (" + ptype + ")",
				fmt.Sprint(nilDash(e["reason"])),
				trunc(fmt.Sprint(e["banned_at"]), 19),
			})
		}
		for _, e := range data.Fail2BanBans {
			rows = append(rows, []string{
				fmt.Sprint(e["ip"]),
				fmt.Sprintf("Fail2Ban (%v)", e["jail"]),
				"auto-banned",
				"-",
			})
		}
		return output.Table([]string{"IP", "Source", "Reason", "Banned At"}, rows, fmt.Sprintf("Banned IPs (%d)", len(rows)))
	},
}

var (
	fwBanReason    string
	fwBanPermanent bool
)

var fwBanCmd = &cobra.Command{
	Use:   "ban [ip]",
	Short: "Ban an IP address",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli firewall ban 1.2.3.4 --reason "Port scanning"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Post("/system/security/blacklist/ban", map[string]any{
			"ip":        args[0],
			"reason":    fwBanReason,
			"permanent": fwBanPermanent,
		})
		if err != nil {
			return err
		}
		var resp map[string]any
		_ = api.Decode(raw, &resp)
		msg := fmt.Sprintf("%s banned", args[0])
		if m, ok := resp["message"].(string); ok && m != "" {
			msg = m
		}
		return output.Success(msg, map[string]any{"ip": args[0]})
	},
}

func init() {
	fwBanCmd.Flags().StringVarP(&fwBanReason, "reason", "r", "Banned via CLI", "Ban reason")
	fwBanCmd.Flags().BoolVar(&fwBanPermanent, "permanent", true, "Permanent ban")
}

var fwUnbanCmd = &cobra.Command{
	Use:   "unban [ip]",
	Short: "Unban an IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Post("/system/security/blacklist/unban", map[string]any{"ip": args[0]})
		if err != nil {
			return err
		}
		var resp map[string]any
		_ = api.Decode(raw, &resp)
		msg := fmt.Sprintf("%s unbanned", args[0])
		if m, ok := resp["message"].(string); ok && m != "" {
			msg = m
		}
		return output.Success(msg, map[string]any{"ip": args[0]})
	},
}

var fwCheckCmd = &cobra.Command{
	Use:   "check [ip]",
	Short: "Check if an IP is currently banned",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/system/security/blacklist/check/"+args[0], nil)
		if err != nil {
			return err
		}
		var data map[string]any
		if err := api.Decode(raw, &data); err != nil {
			return err
		}
		status := "not banned"
		if data["banned"] == true {
			status = "BANNED"
		}
		return output.KV([][2]string{
			{"IP", args[0]},
			{"Status", status},
			{"iptables", fmt.Sprint(data["in_iptables"])},
			{"fail2ban", fmt.Sprint(data["in_fail2ban"])},
			{"jail", fmt.Sprint(nilDash(data["fail2ban_jail"]))},
		}, "")
	},
}

var fwJailsCmd = &cobra.Command{
	Use:   "jails",
	Short: "Show Fail2Ban jail configuration and stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/system/security/fail2ban/config", nil)
		if err != nil {
			return err
		}
		var data struct {
			Jails map[string]map[string]any `json:"jails"`
		}
		if err := api.Decode(raw, &data); err != nil {
			return err
		}
		rows := [][]string{}
		for name, cfg := range data.Jails {
			bt := fmt.Sprint(cfg["bantime"])
			if cfg["bantime"] == float64(-1) || cfg["bantime"] == -1 {
				bt = "permanent"
			} else if cfg["bantime"] != nil {
				bt = fmt.Sprintf("%vs", cfg["bantime"])
			}
			rows = append(rows, []string{
				name,
				bt,
				fmt.Sprintf("%vs", cfg["findtime"]),
				fmt.Sprint(cfg["maxretry"]),
				fmt.Sprint(cfg["currently_banned"]),
				fmt.Sprint(cfg["total_banned"]),
			})
		}
		return output.Table([]string{"Jail", "Ban", "Find", "MaxRetry", "Banned", "Total"}, rows, "Fail2Ban Jails")
	},
}

var (
	fwJailBantime  int
	fwJailFindtime int
	fwJailMaxretry int
)

var fwSetJailCmd = &cobra.Command{
	Use:   "set-jail [jail]",
	Short: "Update Fail2Ban jail settings",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli firewall set-jail sshd --bantime 86400`,
	RunE: func(cmd *cobra.Command, args []string) error {
		payload := map[string]any{"jail": args[0]}
		changed := false
		if cmd.Flags().Changed("bantime") {
			payload["bantime"] = fwJailBantime
			changed = true
		}
		if cmd.Flags().Changed("findtime") {
			payload["findtime"] = fwJailFindtime
			changed = true
		}
		if cmd.Flags().Changed("maxretry") {
			payload["maxretry"] = fwJailMaxretry
			changed = true
		}
		if !changed {
			return fmt.Errorf("specify --bantime, --findtime, or --maxretry\n  qemu-cli firewall set-jail %s --bantime 86400", args[0])
		}
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Put("/system/security/fail2ban/config", payload)
		if err != nil {
			return err
		}
		var resp map[string]any
		_ = api.Decode(raw, &resp)
		msg := "Updated"
		if m, ok := resp["message"].(string); ok && m != "" {
			msg = m
		}
		return output.Success(msg, map[string]any{"jail": args[0]})
	},
}

func init() {
	fwSetJailCmd.Flags().IntVar(&fwJailBantime, "bantime", 0, "Seconds (-1=permanent)")
	fwSetJailCmd.Flags().IntVar(&fwJailFindtime, "findtime", 0, "Detection window (seconds)")
	fwSetJailCmd.Flags().IntVar(&fwJailMaxretry, "maxretry", 0, "Max failed attempts")
}

var fwLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent Fail2Ban log",
	RunE: func(cmd *cobra.Command, args []string) error {
		lines, _ := cmd.Flags().GetInt("lines")
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		q := url.Values{}
		q.Set("lines", fmt.Sprintf("%d", lines))
		raw, err := c.Get("/system/security/fail2ban/log", q)
		if err != nil {
			return err
		}
		var data struct {
			Entries []string `json:"entries"`
		}
		if err := api.Decode(raw, &data); err != nil {
			return err
		}
		if output.GetFormat() != output.FormatTable {
			return output.Table([]string{"Entry"}, func() [][]string {
				rows := make([][]string, 0, len(data.Entries))
				for _, e := range data.Entries {
					rows = append(rows, []string{e})
				}
				return rows
			}(), "")
		}
		for _, e := range data.Entries {
			fmt.Println(e)
		}
		return nil
	},
}

func init() {
	fwLogCmd.Flags().IntP("lines", "n", 30, "Number of log lines")
}
