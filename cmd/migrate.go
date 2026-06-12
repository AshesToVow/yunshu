package cmd

import (
	"yunshu/internal/bootstrap"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.BuildCoreApp(configPath)
		if err != nil {
			return err
		}
		defer app.Close()

		return bootstrap.AutoMigrateModels(app.DB, &app.Config.Plugins)
	},
}
