package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nycon/qemu-cli/internal/api"
	"github.com/nycon/qemu-cli/internal/output"
)

var storagePoolCmd = &cobra.Command{
	Use:   "storage-pool",
	Short: "Manage storage pools",
	Example: `  qemu-cli storage-pool list`,
}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage cloud images",
	Example: `  qemu-cli image list
  qemu-cli -o json image list`,
}

func init() {
	storagePoolCmd.AddCommand(storagePoolListCmd)
	imageCmd.AddCommand(imageListCmd)
}

var storagePoolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List storage pools",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/storage-pools", nil)
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
			var p map[string]any
			_ = api.Decode(it, &p)
			status := "Inactive"
			if p["is_active"] == true {
				status = "Active"
			}
			if p["is_default"] == true {
				status += " (Default)"
			}
			rows = append(rows, []string{
				fmt.Sprint(p["name"]),
				fmt.Sprint(nilDash(p["path"])),
				fmt.Sprintf("%v GB", p["used_size_gb"]),
				fmt.Sprintf("%v GB", p["free_size_gb"]),
				fmt.Sprintf("%v GB", p["total_size_gb"]),
				status,
			})
		}
		return output.Table([]string{"Name", "Path", "VM Usage", "Free", "Total", "Status"}, rows, "Storage Pools")
	},
}

var imageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cloud images",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := clientFromActive()
		if err != nil {
			return err
		}
		raw, err := c.Get("/cloud-images", nil)
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
			var img map[string]any
			_ = api.Decode(it, &img)
			sizeMB := "-"
			if v, ok := img["file_size_bytes"].(float64); ok && v > 0 {
				sizeMB = fmt.Sprintf("%.0f MB", v/(1024*1024))
			}
			rows = append(rows, []string{
				shortID(fmt.Sprint(img["id"])),
				fmt.Sprint(img["name"]),
				fmt.Sprintf("%v %v", img["os_type"], img["os_version"]),
				fmt.Sprint(nilDash(img["format"])),
				sizeMB,
				fmt.Sprint(nilDash(img["download_status"])),
			})
		}
		return output.Table([]string{"ID", "Name", "OS", "Format", "Size", "Status"}, rows, "Cloud Images")
	},
}
