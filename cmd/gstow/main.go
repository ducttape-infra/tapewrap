package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ducttape-infra/gstow/pkg/gstow"
	"github.com/spf13/cobra"
)

var (
	delete  bool
	restow  bool
	target  string
	verbose bool
	dryRun  bool
	force   bool
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:          "gstow [packages...]",
		Short:        "Manage symlink farms for dotfiles",
		SilenceUsage: true,
		Version:      version,
		Args:         cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			cfg := buildConfig()
			for _, pkg := range args {
				var err error
				switch {
				case restow:
					err = gstow.Restow(cfg, pkg)
				case delete:
					err = gstow.Unstow(cfg, pkg)
				default:
					err = gstow.Stow(cfg, pkg)
				}
				if err != nil {
					return fmt.Errorf("%s: %w", pkg, err)
				}
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVarP(&delete, "delete", "D", false, "Delete (unstow) packages")
	rootCmd.Flags().BoolVarP(&restow, "restow", "R", false, "Restow packages (unstow then stow)")
	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "", "Target directory (default: parent of stow dir)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "Dry run -- show actions without performing them")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "Force -- overwrite existing real files/dirs blocking symlinks")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "stow [packages...]",
		Short: "Stow packages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig()
			for _, pkg := range args {
				if err := gstow.Stow(cfg, pkg); err != nil {
					return fmt.Errorf("%s: %w", pkg, err)
				}
			}
			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "unstow [packages...]",
		Short: "Unstow packages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig()
			for _, pkg := range args {
				if err := gstow.Unstow(cfg, pkg); err != nil {
					return fmt.Errorf("%s: %w", pkg, err)
				}
			}
			return nil
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "restow [packages...]",
		Short: "Restow packages (unstow then stow)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := buildConfig()
			for _, pkg := range args {
				if err := gstow.Restow(cfg, pkg); err != nil {
					return fmt.Errorf("%s: %w", pkg, err)
				}
			}
			return nil
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildConfig() *gstow.Config {
	stowDir, _ := filepath.Abs(".")
	tgt := target
	if tgt == "" {
		tgt = filepath.Dir(stowDir)
	}
	targetDir, _ := filepath.Abs(tgt)
	return &gstow.Config{
		StowDir:   stowDir,
		TargetDir: targetDir,
		Verbose:   verbose,
		DryRun:    dryRun,
		Force:     force,
	}
}
