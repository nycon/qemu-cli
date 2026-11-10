package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
	"github.com/nycon/qemu-cli/internal/resolve"
)

var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage virtual machines",
	Long: `Manage VMs on the active tenant.

Examples:
  qemu-cli vm list
  qemu-cli vm create web1 --image "Ubuntu 24.04" --project <id> --ip-pool <id> --random-password
  qemu-cli vm start web1
  qemu-cli vm stop 8b859`,
}

func init() {
	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmDescribeCmd)
	vmCmd.AddCommand(vmStartCmd)
	vmCmd.AddCommand(vmStopCmd)
	vmCmd.AddCommand(vmRebootCmd)
	vmCmd.AddCommand(vmPoweroffCmd)
	vmCmd.AddCommand(vmDeleteCmd)
	vmCmd.AddCommand(vmResizeCmd)
}

var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all VMs",
	Example: `  qemu-cli vm list
  qemu-cli -o json vm list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")
		q := url.Values{}
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("per_page", fmt.Sprintf("%d", perPage))
		raw, err := c.Get("/tenants/"+tid+"/vms", q)
		if err != nil {
			return err
		}
		if handled, err := output.Raw(raw); handled {
			return err
		}
		var wrap struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		}
		if err := api.Decode(raw, &wrap); err != nil {
			return err
		}
		rows := make([][]string, 0, len(wrap.Items))
		for _, vm := range wrap.Items {
			rows = append(rows, []string{
				shortID(fmt.Sprint(vm["id"])),
				fmt.Sprint(vm["name"]),
				fmt.Sprint(vm["status"]),
				fmt.Sprintf("%v CPU / %v MB / %v GB", vm["vcpus"], vm["ram_mb"], vm["disk_size_gb"]),
				trunc(fmt.Sprint(vm["created_at"]), 10),
			})
		}
		title := fmt.Sprintf("VMs (%d)", wrap.Total)
		if wrap.Total == 0 {
			title = fmt.Sprintf("VMs (%d)", len(rows))
		}
		return output.Table([]string{"ID", "Name", "Status", "Resources", "Created"}, rows, title)
	},
}

func init() {
	vmListCmd.Flags().IntP("page", "p", 1, "Page number")
	vmListCmd.Flags().IntP("per-page", "n", 50, "Items per page")
}

var (
	vmImage         string
	vmProject       string
	vmIPPool        string
	vmCPU           int
	vmRAM           int
	vmDisk          int
	vmPassword      string
	vmRandomPass    bool
	vmSSHKey        string
	vmHostname      string
	vmStoragePool   string
	vmDefaultUser   string
	vmDescription   string
)

var vmCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new VM from a cloud image",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli vm create web1 --image "Ubuntu 24.04" --project <uuid> --ip-pool <uuid> --random-password
  qemu-cli vm create db1 --image Debian --project <uuid> --ip-pool <uuid> --cpu 2 --ram 4096 --disk 40 --ssh-key "ssh-ed25519 AAAA..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if vmImage == "" || vmProject == "" || vmIPPool == "" {
			return fmt.Errorf("missing required flags\n  qemu-cli vm create %s --image <name|id> --project <uuid> --ip-pool <uuid>", name)
		}
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		tid, err := config.TenantID()
		if err != nil {
			return err
		}
		imageID, err := resolve.Image(c, vmImage)
		if err != nil {
			return err
		}

		hostname := vmHostname
		if hostname == "" {
			hostname = slugify(name)
		}
		user := vmDefaultUser
		if user == "" {
			user = "cloud"
		}

		password := vmPassword
		if vmRandomPass {
			password, err = randomPassword(20)
			if err != nil {
				return err
			}
			fmt.Printf("Generated password: %s\n", password)
		}

		identity := map[string]any{
			"hostname":               hostname,
			"default_user":           user,
			"ssh_authorized_keys":    []string{},
			"disable_password_login": false,
		}
		if password != "" {
			identity["password"] = password
		}
		if vmSSHKey != "" {
			identity["ssh_authorized_keys"] = []string{vmSSHKey}
		}

		payload := map[string]any{
			"name":                name,
			"description":         vmDescription,
			"project_id":          vmProject,
			"template_id":         imageID,
			"os_profile":          "linux",
			"vcpus":               vmCPU,
			"ram_mb":              vmRAM,
			"disk_size_gb":        vmDisk,
			"ip_pool_id":          vmIPPool,
			"cloud_init_identity": identity,
			"cloud_init_userdata": map[string]any{
				"packages": []string{},
				"runcmd":   []string{},
				"timezone": "Europe/Berlin",
			},
		}
		if vmStoragePool != "" {
			payload["storage_pool_id"] = vmStoragePool
		}

		raw, err := c.Post("/tenants/"+tid+"/vms", payload)
		if err != nil {
			return err
		}
		var vm map[string]any
		_ = api.Decode(raw, &vm)
		id := fmt.Sprint(vm["id"])
		return output.Success(fmt.Sprintf("VM %q created (ID: %s)", name, shortID(id)), map[string]any{
			"id":     id,
			"name":   name,
			"status": vm["status"],
		})
	},
}

func init() {
	vmCreateCmd.Flags().StringVar(&vmImage, "image", "", "Cloud image ID or name (required)")
	vmCreateCmd.Flags().StringVar(&vmProject, "project", "", "Project UUID (required)")
	vmCreateCmd.Flags().StringVar(&vmIPPool, "ip-pool", "", "IP pool UUID (required)")
	vmCreateCmd.Flags().IntVar(&vmCPU, "cpu", 1, "vCPUs")
	vmCreateCmd.Flags().IntVar(&vmRAM, "ram", 1024, "RAM in MB")
	vmCreateCmd.Flags().IntVar(&vmDisk, "disk", 20, "Disk size in GB")
	vmCreateCmd.Flags().StringVar(&vmPassword, "password", "", "Guest password")
	vmCreateCmd.Flags().BoolVar(&vmRandomPass, "random-password", false, "Generate a secure random password")
	vmCreateCmd.Flags().StringVar(&vmSSHKey, "ssh-key", "", "SSH public key")
	vmCreateCmd.Flags().StringVar(&vmHostname, "hostname", "", "Custom hostname (default: from name)")
	vmCreateCmd.Flags().StringVar(&vmStoragePool, "storage-pool", "", "Storage pool UUID")
	vmCreateCmd.Flags().StringVar(&vmDefaultUser, "user", "cloud", "Default cloud-init user")
	vmCreateCmd.Flags().StringVar(&vmDescription, "description", "", "VM description")
}

var vmDescribeCmd = &cobra.Command{
	Use:   "describe [id|name]",
	Short: "Show VM details",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli vm describe my-webserver
  qemu-cli -o json vm describe 8b859`,
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
		raw, err := c.Get("/tenants/"+tid+"/vms/"+vmID, nil)
		if err != nil {
			return err
		}
		var vm map[string]any
		if err := api.Decode(raw, &vm); err != nil {
			return err
		}
		ip := extractIP(vm["cloud_init_network"])
		return output.KV([][2]string{
			{"ID", fmt.Sprint(vm["id"])},
			{"Name", fmt.Sprint(vm["name"])},
			{"Hostname", fmt.Sprint(vm["hostname"])},
			{"Status", fmt.Sprint(vm["status"])},
			{"IP", ip},
			{"vCPUs", fmt.Sprint(vm["vcpus"])},
			{"RAM", fmt.Sprintf("%v MB", vm["ram_mb"])},
			{"Disk", fmt.Sprintf("%v GB (%v)", vm["disk_size_gb"], vm["disk_format"])},
			{"MAC Address", fmt.Sprint(vm["mac_address"])},
			{"Bridge", fmt.Sprint(vm["bridge"])},
			{"VNC Port", fmt.Sprint(nilDash(vm["vnc_port"]))},
			{"Description", fmt.Sprint(nilDash(vm["description"]))},
			{"Created", trunc(fmt.Sprint(vm["created_at"]), 19)},
		}, "")
	},
}

func vmAction(action, idOrName, successMsg string) error {
	c, err := clientFromActive()
	if err != nil {
		return err
	}
	tid, err := config.TenantID()
	if err != nil {
		return err
	}
	vmID, err := resolve.VM(c, tid, idOrName)
	if err != nil {
		return err
	}
	raw, err := c.Post("/tenants/"+tid+"/vms/"+vmID+"/action", map[string]string{"action": action})
	if err != nil {
		return err
	}
	extra := map[string]any{"id": vmID, "action": action}
	var resp map[string]any
	if api.Decode(raw, &resp) == nil {
		if t, ok := resp["task_id"]; ok {
			extra["task_id"] = t
		}
	}
	return output.Success(successMsg, extra)
}

var vmStartCmd = &cobra.Command{
	Use:   "start [id|name]", Short: "Start a VM", Args: cobra.ExactArgs(1),
	Example: `  qemu-cli vm start my-webserver`,
	RunE: func(cmd *cobra.Command, args []string) error { return vmAction("start", args[0], "VM start initiated") },
}
var vmStopCmd = &cobra.Command{
	Use:   "stop [id|name]", Short: "Gracefully stop a VM", Args: cobra.ExactArgs(1),
	Example: `  qemu-cli vm stop my-webserver`,
	RunE: func(cmd *cobra.Command, args []string) error { return vmAction("stop", args[0], "VM stop initiated") },
}
var vmRebootCmd = &cobra.Command{
	Use:   "reboot [id|name]", Short: "Reboot a VM", Args: cobra.ExactArgs(1),
	Example: `  qemu-cli vm reboot my-webserver`,
	RunE: func(cmd *cobra.Command, args []string) error { return vmAction("reboot", args[0], "VM reboot initiated") },
}
var vmPoweroffCmd = &cobra.Command{
	Use:   "poweroff [id|name]", Short: "Force power-off a VM", Args: cobra.ExactArgs(1),
	Example: `  qemu-cli vm poweroff my-webserver`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return vmAction("force_stop", args[0], "VM powered off")
	},
}

var (
	vmDeleteYes    bool
	vmDeleteForce  bool
	vmDeleteDryRun bool
)

var vmDeleteCmd = &cobra.Command{
	Use:   "delete [id|name]",
	Short: "Delete a VM permanently",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli vm delete old-box --yes
  qemu-cli vm delete old-box --dry-run`,
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
		raw, err := c.Get("/tenants/"+tid+"/vms/"+vmID, nil)
		if err != nil {
			return err
		}
		var vm map[string]any
		_ = api.Decode(raw, &vm)
		name := fmt.Sprint(vm["name"])
		if vmDeleteDryRun {
			return output.Success(fmt.Sprintf("dry-run: would delete VM %q (%s)", name, vmID), map[string]any{
				"id": vmID, "name": name, "dry_run": true,
			})
		}
		if err := confirm(fmt.Sprintf("Delete VM %q?", name), vmDeleteYes); err != nil {
			return err
		}
		path := "/tenants/" + tid + "/vms/" + vmID
		if vmDeleteForce {
			path += "?force=true"
		}
		if _, err := c.Delete(path); err != nil {
			return err
		}
		return output.Success("VM deleted", map[string]any{"id": vmID, "name": name})
	},
}

func init() {
	vmDeleteCmd.Flags().BoolVarP(&vmDeleteYes, "yes", "y", false, "Skip confirmation")
	vmDeleteCmd.Flags().BoolVar(&vmDeleteForce, "force", false, "Force delete")
	vmDeleteCmd.Flags().BoolVar(&vmDeleteDryRun, "dry-run", false, "Show what would be deleted")
}

var (
	vmResizeCPU  int
	vmResizeRAM  int
	vmResizeDisk int
)

var vmResizeCmd = &cobra.Command{
	Use:   "resize [id|name]",
	Short: "Resize a stopped VM",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli vm resize web1 --cpu 4 --ram 8192
  qemu-cli vm resize web1 --disk 50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		payload := map[string]any{}
		if cmd.Flags().Changed("cpu") {
			payload["vcpus"] = vmResizeCPU
		}
		if cmd.Flags().Changed("ram") {
			payload["ram_mb"] = vmResizeRAM
		}
		if cmd.Flags().Changed("disk") {
			payload["disk_size_gb"] = vmResizeDisk
		}
		if len(payload) == 0 {
			return fmt.Errorf("specify --cpu, --ram, or --disk\n  qemu-cli vm resize %s --cpu 2 --ram 4096", args[0])
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
		if _, err := c.Patch("/tenants/"+tid+"/vms/"+vmID, payload); err != nil {
			return err
		}
		return output.Success("VM resized", map[string]any{"id": vmID, "changes": payload})
	},
}

func init() {
	vmResizeCmd.Flags().IntVar(&vmResizeCPU, "cpu", 0, "vCPUs")
	vmResizeCmd.Flags().IntVar(&vmResizeRAM, "ram", 0, "RAM in MB")
	vmResizeCmd.Flags().IntVar(&vmResizeDisk, "disk", 0, "Disk in GB (grow only)")
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func nilDash(v any) any {
	if v == nil {
		return "-"
	}
	s := fmt.Sprint(v)
	if s == "" || s == "<nil>" {
		return "-"
	}
	return s
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "vm"
	}
	return s
}

func randomPassword(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*+-_="
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[v.Int64()]
	}
	return string(out), nil
}

func extractIP(netCfg any) string {
	if netCfg == nil {
		return "-"
	}
	b, err := json.Marshal(netCfg)
	if err != nil {
		return "-"
	}
	var cfg struct {
		Ethernets map[string]struct {
			Addresses []string `json:"addresses"`
		} `json:"ethernets"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		return "-"
	}
	for _, iface := range cfg.Ethernets {
		if len(iface.Addresses) > 0 {
			return strings.Split(iface.Addresses[0], "/")[0]
		}
	}
	return "-"
}
