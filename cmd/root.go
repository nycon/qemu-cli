package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/output"
)

// Version is set at build time via -ldflags.
var Version = "1.1.0"

var (
	outputFormat string
	insecureFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "qemu-cli",
	Short: "QEMU Interface CLI – manage VMs, snapshots, backups, and security",
	Long: `qemu-cli talks to a QEMU Interface panel using an API key.

Examples:
  qemu-cli context create production --url https://panel.example.com --token qi_...
  qemu-cli vm list
  qemu-cli -o json vm list
  qemu-cli vm start my-webserver`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if outputFormat != "" {
			if err := output.SetFormat(outputFormat); err != nil {
				return err
			}
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		output.Errorf("%s", err.Error())
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolVar(&insecureFlag, "insecure", false, "Skip TLS certificate verification (self-signed panels)")

	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(vmCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(storagePoolCmd)
	rootCmd.AddCommand(imageCmd)
	rootCmd.AddCommand(firewallCmd)
	rootCmd.AddCommand(tenantCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(serverInfoCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}

func die(err error) error {
	if err == nil {
		return nil
	}
	return err
}

func confirm(prompt string, yes bool) error {
	if yes {
		return nil
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	var ans string
	_, _ = fmt.Scanln(&ans)
	if ans != "y" && ans != "Y" && ans != "yes" {
		return fmt.Errorf("aborted")
	}
	return nil
}
