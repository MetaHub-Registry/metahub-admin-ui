package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/metahub-registry/metahub-admin-ui/internal/inventory"
	"github.com/metahub-registry/metahub-admin-ui/internal/server"
)

// uiAssets is injected by Execute() — package main owns the go:embed of
// dist/ and hands the SPA filesystem over to the server.
var uiAssets fs.FS

var (
	port     int
	logLevel string
	cfgFile  string
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

		// Inventory sources: config block + fail-fast validation. A missing
		// FILE means "no sources configured" (empty inventory ⇒ every lookup
		// is the documented 404 no-match); a present-but-invalid file is a
		// startup failure naming the offending entry.
		cfg, err := inventory.LoadConfig(cfgFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				log.WithField("config", cfgFile).Warn("inventory config not found — starting with an empty inventory")
				cfg = inventory.Config{}
			} else {
				return fmt.Errorf("load inventory config %s: %w", cfgFile, err)
			}
		} else if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid inventory config %s: %w", cfgFile, err)
		}
		engine, err := inventory.Build(cfg, inventory.NetPTRLookup)
		if err != nil {
			return fmt.Errorf("build inventory engine: %w", err)
		}
		log.WithFields(log.Fields{
			"config":      cfgFile,
			"entries":     len(cfg.Entries),
			"rules":       len(cfg.Hostname.Rules),
			"lookup_ttl":  engine.TTL(),
		}).Info("inventory sources loaded")

		srv := server.New(server.Options{
			Port:           port,
			Assets:         uiAssets,
			Version:        version,
			Engine:         engine,
			InventoryToken: os.Getenv("INVENTORY_TOKEN"),
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
	rootCmd.Flags().StringVar(&cfgFile, "config", envOr("CFG_FILE", "/etc/metahub-admin-ui/config.yaml"),
		"inventory config file (env: CFG_FILE)")
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
