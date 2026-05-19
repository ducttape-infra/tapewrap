package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the runtime configuration for stow operations.
type Config struct {
	StowDir   string
	TargetDir string
	Verbose   bool
	DryRun    bool
}

func (c *Config) log(format string, args ...any) {
	if c.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}

func (c *Config) warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// Stow symlinks all files/dirs in the package into the target directory,
// using directory folding: if the target path doesn't exist, a symlink to
// the source is created; if it's a real directory, we recurse into it.
func Stow(cfg *Config, pkg string) error {
	pkgDir := filepath.Join(cfg.StowDir, pkg)
	return stowDir(cfg, pkgDir, cfg.TargetDir)
}

func stowDir(cfg *Config, srcDir, tgtDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		tgtPath := filepath.Join(tgtDir, entry.Name())

		if entry.IsDir() {
			if err := stowDirEntry(cfg, srcPath, tgtPath); err != nil {
				return err
			}
		} else {
			if err := stowFileEntry(cfg, srcPath, tgtPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func stowDirEntry(cfg *Config, srcPath, tgtPath string) error {
	info, err := os.Lstat(tgtPath)
	if os.IsNotExist(err) {
		// Target absent: fold by creating a directory symlink
		cfg.log("link dir: %s -> %s", srcPath, tgtPath)
		if cfg.DryRun {
			return nil
		}
		return createDirLink(srcPath, tgtPath)
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", tgtPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// It's a symlink — check if it already points to our source
		existing, err := os.Readlink(tgtPath)
		if err != nil {
			return fmt.Errorf("readlink %s: %w", tgtPath, err)
		}
		if existing == srcPath {
			cfg.log("already linked: %s", tgtPath)
			return nil
		}
		cfg.warn("%s is a symlink to %q, not %q — skipping", tgtPath, existing, srcPath)
		return nil
	}

	if info.IsDir() {
		// Real directory: recurse (can't fold here)
		return stowDir(cfg, srcPath, tgtPath)
	}

	cfg.warn("%s exists as a file, expected directory — skipping", tgtPath)
	return nil
}

func stowFileEntry(cfg *Config, srcPath, tgtPath string) error {
	info, err := os.Lstat(tgtPath)
	if os.IsNotExist(err) {
		cfg.log("link: %s -> %s", srcPath, tgtPath)
		if cfg.DryRun {
			return nil
		}
		return os.Symlink(srcPath, tgtPath)
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", tgtPath, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		existing, err := os.Readlink(tgtPath)
		if err != nil {
			return fmt.Errorf("readlink %s: %w", tgtPath, err)
		}
		if existing == srcPath {
			cfg.log("already linked: %s", tgtPath)
			return nil
		}
		cfg.warn("%s is a symlink to %q, not %q — skipping", tgtPath, existing, srcPath)
		return nil
	}

	cfg.warn("%s already exists and is not a symlink — skipping", tgtPath)
	return nil
}

// Unstow removes symlinks in the target directory that point into the package.
func Unstow(cfg *Config, pkg string) error {
	pkgDir := filepath.Join(cfg.StowDir, pkg)
	return unstowDir(cfg, pkgDir, cfg.TargetDir)
}

func unstowDir(cfg *Config, srcDir, tgtDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		tgtPath := filepath.Join(tgtDir, entry.Name())

		info, err := os.Lstat(tgtPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat %s: %w", tgtPath, err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			existing, err := os.Readlink(tgtPath)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", tgtPath, err)
			}
			// Remove if it points to our source (exact match or under pkgDir prefix)
			if existing == srcPath || strings.HasPrefix(existing, srcPath+string(filepath.Separator)) {
				cfg.log("unlink: %s", tgtPath)
				if !cfg.DryRun {
					if err := removeDirLink(tgtPath, info); err != nil {
						return fmt.Errorf("remove %s: %w", tgtPath, err)
					}
				}
			}
		} else if info.IsDir() && entry.IsDir() {
			// Real directory on both sides: recurse to find individual file links
			if err := unstowDir(cfg, srcPath, tgtPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// Restow unstows then stows a package.
func Restow(cfg *Config, pkg string) error {
	if err := Unstow(cfg, pkg); err != nil {
		return fmt.Errorf("unstow: %w", err)
	}
	return Stow(cfg, pkg)
}
