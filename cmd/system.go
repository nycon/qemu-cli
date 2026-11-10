package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/output"
)

var serverInfoCmd = &cobra.Command{
	Use:   "server-info",
	Short: "Show host system status (CPU, RAM, Disk)",
	Example: `  qemu-cli server-info
  qemu-cli -o json server-info`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/system/metrics", nil)
		if err != nil {
			return err
		}
		if handled, err := output.Raw(raw); handled {
			return err
		}
		var data map[string]any
		if err := api.Decode(raw, &data); err != nil {
			return err
		}
		cpu, _ := data["cpu"].(map[string]any)
		mem, _ := data["memory"].(map[string]any)
		disk, _ := data["disk"].(map[string]any)
		cpuPct := cpu["percent"]
		if arr, ok := cpuPct.([]any); ok && len(arr) > 0 {
			sum := 0.0
			for _, v := range arr {
				if f, ok := v.(float64); ok {
					sum += f
				}
			}
			cpuPct = fmt.Sprintf("%.1f", sum/float64(len(arr)))
		}
		return output.KV([][2]string{
			{"CPU Cores", fmt.Sprint(nilDash(cpu["count"]))},
			{"CPU Usage", fmt.Sprintf("%v%%", cpuPct)},
			{"RAM Total", fmt.Sprintf("%v MB", mem["total_mb"])},
			{"RAM Used", fmt.Sprintf("%v MB (%v%%)", mem["used_mb"], mem["percent"])},
			{"RAM Available", fmt.Sprintf("%v MB", mem["available_mb"])},
			{"Disk Total", fmt.Sprintf("%v GB", disk["total_gb"])},
			{"Disk Used", fmt.Sprintf("%v GB (%v%%)", disk["used_gb"], disk["percent"])},
			{"Disk Free", fmt.Sprintf("%v GB", disk["free_gb"])},
		}, "Host System")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return output.KV([][2]string{{"Version", Version}}, "")
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate completion script for your shell.

Examples:
  qemu-cli completion zsh
  qemu-cli completion bash > /etc/bash_completion.d/qemu-cli
  qemu-cli completion fish > ~/.config/fish/completions/qemu-cli.fish`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell %q\n  qemu-cli completion zsh", args[0])
		}
	},
}
