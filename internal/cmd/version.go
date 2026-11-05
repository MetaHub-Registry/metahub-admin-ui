package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is bumped by .bumpversion.cfg (session `task dev` / `task version:*:*`).
var version = "0.1.2"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of metahub-admin-ui",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("metahub-admin-ui v%s\n", version)
	},
}
