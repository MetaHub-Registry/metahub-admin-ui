package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/metahub-registry/metahub-admin-ui/internal/server"
)

// uiAssets is injected by Execute() — package main owns the go:embed of
// dist/ and hands the SPA filesystem over to the server.
var uiAssets fs.FS

var (
	port     int
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "metahub-admin-ui",
	Short: "MetaHub admin UI service: inventory lookup API + embedded SPA",
	Long: `metahub-admin-ui serves the ops-facing admin UI and the versioned
inventory API (/api/v1/...). First capability: profile resolution +
inventory lookup (see openspec/changes/add-inventory-service/).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		lvl, err := log.ParseLevel(logLevel)
		if err != nil {
			return fmt.Errorf("invalid --log-level %q: %w", logLevel, err)
		}
		log.SetLevel(lvl)

		srv := server.New(server.Options{
			Port:    port,
			Assets:  uiAssets,
			Version: version,
		})
		return srv.ListenAndServe()
	},
}

// Execute wires the embedded SPA assets in and runs the root command.
func Execute(assets fs.FS) {
	uiAssets = assets
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", envPort(),
		"listen port (env: METAHUB_ADMIN_UI_PORT)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info",
		"log level (debug|info|warn|error)")
	rootCmd.AddCommand(versionCmd)
}

func envPort() int {
	if v := os.Getenv("METAHUB_ADMIN_UI_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 8080
}
