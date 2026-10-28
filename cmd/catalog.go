package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"yunshu/internal/pkg/permcatalog"

	"github.com/spf13/cobra"
)

func init() {
	catalogCmd.Flags().StringP("out", "o", "docs/permission-catalog.json", "输出文件路径；- 表示只打印到 stdout")
	rootCmd.AddCommand(catalogCmd)
}

var catalogCmd = &cobra.Command{
	Use:   "permission-catalog",
	Short: "Generate the permission catalog snapshot from the single source",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("out")
		data, err := json.MarshalIndent(permcatalog.Snapshot(), "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if out == "-" {
			_, err = os.Stdout.Write(data)
			return err
		}
		if dir := filepath.Dir(out); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d entries)\n", out, len(permcatalog.Entries()))
		return nil
	},
}
