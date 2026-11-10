package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/config"
	"github.com/nycon/qemu-cli/internal/output"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage server contexts",
	Long: `Manage panel connections (URL + API key).

Examples:
  qemu-cli context create production --url https://panel.example.com --token qiqi_...
  qemu-cli context list
  qemu-cli context use production
  qemu-cli context active`,
}

func init() {
	contextCmd.AddCommand(contextCreateCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextUseCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	contextCmd.AddCommand(contextActiveCmd)
}

var (
	ctxURL      string
	ctxToken    string
	ctxInsecure bool
)

var contextCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new server context",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli context create production --url https://panel.example.com --token qiqi_abc
  qemu-cli context create staging --url https://staging.example.com --token qiqi_... --insecure`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := strings.TrimSpace(ctxURL)
		token := strings.TrimSpace(ctxToken)

		if url == "" {
			fmt.Fprint(os.Stderr, "Server URL: ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			url = strings.TrimSpace(line)
		}
		if token == "" {
			fmt.Fprint(os.Stderr, "API Token (qi_...): ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			token = strings.TrimSpace(line)
		}
		if url == "" || token == "" {
			return fmt.Errorf("url and token are required\n  qemu-cli context create %s --url <url> --token <qi_...>", name)
		}
		url = strings.TrimRight(url, "/")
		insecure := ctxInsecure || insecureFlag

		client := api.New(url, token, insecure)
		raw, err := client.Get("/auth/me", nil)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		var me map[string]any
		_ = api.Decode(raw, &me)
		if email, ok := me["email"].(string); ok {
			fmt.Fprintf(os.Stderr, "Authenticated as: %s\n", email)
		}

		tenantID := ""
		if tRaw, err := client.Get("/tenants", nil); err == nil {
			items, _ := api.AsList(tRaw)
			if len(items) > 0 {
				var t struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				if api.Decode(items[0], &t) == nil {
					tenantID = t.ID
					fmt.Fprintf(os.Stderr, "Tenant: %s\n", t.Name)
				}
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.Contexts[name] = config.Context{
			ServerURL: url,
			Token:     token,
			TenantID:  tenantID,
			Insecure:  insecure,
		}
		cfg.ActiveContext = name
		if err := config.Save(cfg); err != nil {
			return err
		}
		return output.Success(fmt.Sprintf("Context %q created and activated", name), map[string]any{
			"context": name,
			"url":     url,
		})
	},
}

func init() {
	contextCreateCmd.Flags().StringVar(&ctxURL, "url", "", "Panel base URL (e.g. https://panel.example.com)")
	contextCreateCmd.Flags().StringVar(&ctxToken, "token", "", "API key (qi_...)")
	contextCreateCmd.Flags().BoolVar(&ctxInsecure, "insecure", false, "Skip TLS verification for this context")
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	Example: `  qemu-cli context list
  qemu-cli -o json context list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Contexts) == 0 {
			fmt.Fprintln(os.Stderr, "No contexts configured. Run: qemu-cli context create <name> --url <url> --token <qi_...>")
			return nil
		}
		rows := [][]string{}
		for name, ctx := range cfg.Contexts {
			marker := " "
			if name == cfg.ActiveContext {
				marker = "*"
			}
			rows = append(rows, []string{marker + " " + name, ctx.ServerURL})
		}
		return output.Table([]string{"Name", "Server"}, rows, "Contexts")
	},
}

var contextUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Switch the active context",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli context use production`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Contexts[name]; !ok {
			return fmt.Errorf("context %q not found\n  qemu-cli context list", name)
		}
		cfg.ActiveContext = name
		if err := config.Save(cfg); err != nil {
			return err
		}
		return output.Success(fmt.Sprintf("Active context: %s", name), map[string]any{"context": name})
	},
}

var contextDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a context",
	Args:  cobra.ExactArgs(1),
	Example: `  qemu-cli context delete staging`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Contexts[name]; !ok {
			return fmt.Errorf("context %q not found\n  qemu-cli context list", name)
		}
		delete(cfg.Contexts, name)
		if cfg.ActiveContext == name {
			cfg.ActiveContext = ""
			for k := range cfg.Contexts {
				cfg.ActiveContext = k
				break
			}
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		return output.Success(fmt.Sprintf("Context %q deleted", name), map[string]any{"context": name})
	},
}

var contextActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the active context",
	Example: `  qemu-cli context active`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, ctx, err := config.Active()
		if err != nil {
			fmt.Fprintln(os.Stderr, "No active context.")
			return nil
		}
		return output.KV([][2]string{
			{"Name", name},
			{"Server", ctx.ServerURL},
			{"Tenant", ctx.TenantID},
			{"Insecure", fmt.Sprintf("%v", ctx.Insecure)},
		}, "Active Context")
	},
}
