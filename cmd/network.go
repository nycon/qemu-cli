package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage tenant NAT / Internal networks",
	Example: `  qemu-cli network list
  qemu-cli network create web-dmz --type nat
  qemu-cli network create db --type internal --network 10.42.0.0/24 --gateway 10.42.0.1
  qemu-cli network update <id> --name dmz2 --dhcp=false
  qemu-cli network delete <id> --yes`,
}

var (
	netType      string
	netDesc      string
	netCIDR      string
	netGateway   string
	netDNS1      string
	netDNS2      string
	netDHCP      bool
	netDHCPStart string
	netDHCPEnd   string
	netName      string
	netDeleteYes bool
)

func init() {
	rootCmd.AddCommand(networkCmd)
	networkCmd.AddCommand(networkListCmd)
	networkCmd.AddCommand(networkCreateCmd)
	networkCmd.AddCommand(networkUpdateCmd)
	networkCmd.AddCommand(networkDeleteCmd)

	networkCreateCmd.Flags().StringVar(&netType, "type", "nat", "Network type: nat | internal")
	networkCreateCmd.Flags().StringVar(&netDesc, "description", "", "Description")
	networkCreateCmd.Flags().StringVar(&netCIDR, "network", "", "CIDR (optional; auto if empty)")
	networkCreateCmd.Flags().StringVar(&netGateway, "gateway", "", "Gateway (required with --network)")
	networkCreateCmd.Flags().StringVar(&netDNS1, "dns-primary", "8.8.8.8", "Primary DNS")
	networkCreateCmd.Flags().StringVar(&netDNS2, "dns-secondary", "1.1.1.1", "Secondary DNS")
	networkCreateCmd.Flags().BoolVar(&netDHCP, "dhcp", true, "Enable DHCP")
	networkCreateCmd.Flags().StringVar(&netDHCPStart, "dhcp-start", "", "DHCP start IP")
	networkCreateCmd.Flags().StringVar(&netDHCPEnd, "dhcp-end", "", "DHCP end IP")

	networkUpdateCmd.Flags().StringVar(&netName, "name", "", "New short name")
	networkUpdateCmd.Flags().StringVar(&netDesc, "description", "", "Description")
	networkUpdateCmd.Flags().StringVar(&netDNS1, "dns-primary", "", "Primary DNS")
	networkUpdateCmd.Flags().StringVar(&netDNS2, "dns-secondary", "", "Secondary DNS")
	networkUpdateCmd.Flags().BoolVar(&netDHCP, "dhcp", true, "Enable DHCP (pass explicitly to change)")
	networkUpdateCmd.Flags().StringVar(&netDHCPStart, "dhcp-start", "", "DHCP start IP")
	networkUpdateCmd.Flags().StringVar(&netDHCPEnd, "dhcp-end", "", "DHCP end IP")

	networkDeleteCmd.Flags().BoolVar(&netDeleteYes, "yes", false, "Skip confirmation")
}

var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List NAT/Internal networks for the active tenant",
	Example: `  qemu-cli network list
  qemu-cli -o json network list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		raw, err := c.Get("/tenants/"+tid+"/networks", nil)
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
			var n map[string]any
			_ = api.Decode(it, &n)
			dhcp := "off"
			if b, ok := n["internal_dhcp_enabled"].(bool); ok && b {
				dhcp = "on"
			}
			rows = append(rows, []string{
				shortID(fmt.Sprint(n["id"])),
				fmt.Sprint(n["name"]),
				fmt.Sprint(n["pool_type"]),
				fmt.Sprint(n["network"]),
				fmt.Sprint(n["gateway"]),
				fmt.Sprintf("%v/%v", n["available_ips"], n["total_ips"]),
				dhcp,
			})
		}
		return output.Table([]string{"ID", "Name", "Type", "CIDR", "Gateway", "Free", "DHCP"}, rows, "Networks")
	},
}

var networkCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a NAT or Internal network",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli network create web-dmz --type nat
  qemu-cli network create db --type internal --network 10.42.0.0/24 --gateway 10.42.0.1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		t := strings.ToLower(strings.TrimSpace(netType))
		if t != "nat" && t != "internal" {
			return fmt.Errorf("invalid --type %q (use nat or internal)", netType)
		}
		if (netCIDR == "") != (netGateway == "") {
			return fmt.Errorf("provide both --network and --gateway, or neither for auto CIDR")
		}
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		body := map[string]any{
			"name":                  name,
			"pool_type":             t,
			"dns_primary":           netDNS1,
			"internal_dhcp_enabled": netDHCP,
		}
		if netDesc != "" {
			body["description"] = netDesc
		}
		if netDNS2 != "" {
			body["dns_secondary"] = netDNS2
		}
		if netCIDR != "" {
			body["network"] = netCIDR
			body["gateway"] = netGateway
		}
		if netDHCP && netDHCPStart != "" {
			body["dhcp_start"] = netDHCPStart
		}
		if netDHCP && netDHCPEnd != "" {
			body["dhcp_end"] = netDHCPEnd
		}
		raw, err := c.Post("/tenants/"+tid+"/networks", body)
		if err != nil {
			return err
		}
		var n map[string]any
		_ = api.Decode(raw, &n)
		return output.Success(fmt.Sprintf("Network %q created", n["name"]), map[string]any{
			"id":        n["id"],
			"name":      n["name"],
			"pool_type": n["pool_type"],
			"network":   n["network"],
		})
	},
}

var networkUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a tenant NAT/Internal network",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli network update a1b2c3d4 --name dmz2
  qemu-cli network update a1b2c3d4 --dhcp=false`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		poolID, err := resolveTenantNetwork(c, tid, args[0])
		if err != nil {
			return err
		}
		body := map[string]any{}
		if netName != "" {
			body["name"] = netName
		}
		if cmd.Flags().Changed("description") {
			body["description"] = netDesc
		}
		if netDNS1 != "" {
			body["dns_primary"] = netDNS1
		}
		if cmd.Flags().Changed("dns-secondary") {
			body["dns_secondary"] = netDNS2
		}
		if cmd.Flags().Changed("dhcp") {
			body["internal_dhcp_enabled"] = netDHCP
		}
		if cmd.Flags().Changed("dhcp-start") {
			if netDHCPStart == "" {
				body["dhcp_start"] = nil
			} else {
				body["dhcp_start"] = netDHCPStart
			}
		}
		if cmd.Flags().Changed("dhcp-end") {
			if netDHCPEnd == "" {
				body["dhcp_end"] = nil
			} else {
				body["dhcp_end"] = netDHCPEnd
			}
		}
		if len(body) == 0 {
			return fmt.Errorf("nothing to update — pass --name, --dns-primary, --dhcp, …")
		}
		raw, err := c.Patch("/tenants/"+tid+"/networks/"+poolID, body)
		if err != nil {
			return err
		}
		var n map[string]any
		_ = api.Decode(raw, &n)
		return output.Success("Network updated", map[string]any{
			"id":   n["id"],
			"name": n["name"],
		})
	},
}

var networkDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Short:   "Delete a tenant NAT/Internal network",
	Args:    cobra.ExactArgs(1),
	Example: `  qemu-cli network delete a1b2c3d4 --yes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, tid, err := clientWithTenant()
		if err != nil {
			return err
		}
		poolID, err := resolveTenantNetwork(c, tid, args[0])
		if err != nil {
			return err
		}
		if err := confirm(fmt.Sprintf("Delete network %s?", shortID(poolID)), netDeleteYes); err != nil {
			return err
		}
		_, err = c.Delete("/tenants/" + tid + "/networks/" + poolID)
		if err != nil {
			return err
		}
		return output.Success("Network deleted", map[string]any{"id": poolID})
	},
}

func clientWithTenant() (*api.Client, string, error) {
	c, err := clientFromActive()
	if err != nil {
		return nil, "", err
	}
	tid, err := config.TenantID()
	if err != nil {
		return nil, "", err
	}
	return c, tid, nil
}

func resolveTenantNetwork(c *api.Client, tid, ref string) (string, error) {
	raw, err := c.Get("/tenants/"+tid+"/networks", nil)
	if err != nil {
		return "", err
	}
	items, err := api.AsList(raw)
	if err != nil {
		return "", err
	}
	ref = strings.TrimSpace(ref)
	var matches []string
	for _, it := range items {
		var n map[string]any
		_ = api.Decode(it, &n)
		id := fmt.Sprint(n["id"])
		name := fmt.Sprint(n["name"])
		if id == ref || strings.HasPrefix(id, ref) || strings.EqualFold(name, ref) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("network not found: %s\n  qemu-cli network list", ref)
	}
	return "", fmt.Errorf("ambiguous network %q (%d matches)\n  qemu-cli network list", ref, len(matches))
}
