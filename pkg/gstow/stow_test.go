package gstow

import (
	"os"
	"path/filepath"
	"testing"
)

// makeTree creates a directory tree from a map of relative paths to content.
// Paths ending in "/" are created as directories; others as files with given content.
func makeTree(t *testing.T, base string, tree map[string]string) {
	t.Helper()
	for path, content := range tree {
		full := filepath.Join(base, filepath.FromSlash(path))
		if len(path) > 0 && path[len(path)-1] == '/' {
			if err := os.MkdirAll(full, 0755); err != nil {
				t.Fatalf("makeTree mkdir %s: %v", full, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				t.Fatalf("makeTree mkdirall %s: %v", filepath.Dir(full), err)
			}
			if err := os.WriteFile(full, []byte(content), 0644); err != nil {
				t.Fatalf("makeTree write %s: %v", full, err)
			}
		}
	}
}

// assertSymlink checks that path is a symlink pointing to wantTarget.
func assertSymlink(t *testing.T, path, wantTarget string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Errorf("assertSymlink: lstat %s: %v", path, err)
		return
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("assertSymlink: %s is not a symlink (mode %v)", path, info.Mode())
		return
	}
	got, err := os.Readlink(path)
	if err != nil {
		t.Errorf("assertSymlink: readlink %s: %v", path, err)
		return
	}
	if got != wantTarget {
		t.Errorf("assertSymlink: %s -> %q, want %q", path, got, wantTarget)
	}
}

// assertAbsent checks that path does not exist (not even as a broken symlink).
func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("assertAbsent: %s should not exist", path)
	}
}

// assertNotSymlink checks that path exists as a real file/dir (not a symlink).
func assertNotSymlink(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Errorf("assertNotSymlink: lstat %s: %v", path, err)
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("assertNotSymlink: %s is a symlink (should be real)", path)
	}
}

func newCfg(stowDir, targetDir string) *Config {
	return &Config{StowDir: stowDir, TargetDir: targetDir}
}

// ---------------------------------------------------------------------------
// Stow: stow links package CONTENTS into the target directory.
// e.g. stow "zsh" links items inside <stow>/zsh/ into <target>/
// ---------------------------------------------------------------------------

func TestStow_FileInPackage(t *testing.T) {
	// Package zsh has .zshrc → <tgt>/.zshrc symlink created.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"zsh/.zshrc": "# zsh config"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "zsh"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	assertSymlink(t,
		filepath.Join(tgtDir, ".zshrc"),
		filepath.Join(stowDir, "zsh", ".zshrc"),
	)
}

func TestStow_FoldsSubdirectory(t *testing.T) {
	// Package zsh has .zshrc.d/a.zsh; .zshrc.d doesn't exist in target.
	// Fold: create <tgt>/.zshrc.d → <stow>/zsh/.zshrc.d (dir symlink).
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"zsh/.zshrc":           "# rc",
		"zsh/.zshrc.d/a.zsh":  "alias ll='ls -la'",
	})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "zsh"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	// File linked directly
	assertSymlink(t,
		filepath.Join(tgtDir, ".zshrc"),
		filepath.Join(stowDir, "zsh", ".zshrc"),
	)
	// Directory folded
	assertSymlink(t,
		filepath.Join(tgtDir, ".zshrc.d"),
		filepath.Join(stowDir, "zsh", ".zshrc.d"),
	)
}

func TestStow_DirectoryFolding_NestedRealDir(t *testing.T) {
	// Package config has .config/dotfiles/dotfiles.ini.
	// .config exists as a real dir → recurse.
	// .config/dotfiles doesn't exist → fold as dir symlink.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"config/.config/dotfiles/dotfiles.ini": "[dotfiles]",
	})
	if err := os.MkdirAll(filepath.Join(tgtDir, ".config"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "config"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	// .config exists (real dir) → recursed into it
	// .config/dotfiles absent → folded
	assertSymlink(t,
		filepath.Join(tgtDir, ".config", "dotfiles"),
		filepath.Join(stowDir, "config", ".config", "dotfiles"),
	)
}

func TestStow_RecursesIntoRealDir(t *testing.T) {
	// Package pkg has .config/zsh/rc; both .config and .config/zsh exist as real dirs.
	// Should create individual file symlink .config/zsh/rc.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"pkg/.config/zsh/rc":      "# zsh",
		"pkg/.config/zsh/env":     "export X=1",
	})
	if err := os.MkdirAll(filepath.Join(tgtDir, ".config", "zsh"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "pkg"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	assertSymlink(t,
		filepath.Join(tgtDir, ".config", "zsh", "rc"),
		filepath.Join(stowDir, "pkg", ".config", "zsh", "rc"),
	)
	assertSymlink(t,
		filepath.Join(tgtDir, ".config", "zsh", "env"),
		filepath.Join(stowDir, "pkg", ".config", "zsh", "env"),
	)
}

func TestStow_AlreadyLinked_Idempotent(t *testing.T) {
	// Stowing twice should not error and links stay correct.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"vim/.vimrc": "set nu"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("first Stow: %v", err)
	}
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("second Stow (idempotent): %v", err)
	}

	assertSymlink(t,
		filepath.Join(tgtDir, ".vimrc"),
		filepath.Join(stowDir, "vim", ".vimrc"),
	)
}

func TestStow_ExistingRealFile_NoOverwrite(t *testing.T) {
	// A real file at the target location must not be replaced.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"vim/.vimrc": "set nu"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tgtDir, ".vimrc"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	assertNotSymlink(t, filepath.Join(tgtDir, ".vimrc"))
}

// ---------------------------------------------------------------------------
// Unstow tests
// ---------------------------------------------------------------------------

func TestUnstow_RemovesFileLink(t *testing.T) {
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"zsh/.zshrc": "# rc"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "zsh"); err != nil {
		t.Fatalf("Stow: %v", err)
	}
	if err := Unstow(cfg, "zsh"); err != nil {
		t.Fatalf("Unstow: %v", err)
	}

	assertAbsent(t, filepath.Join(tgtDir, ".zshrc"))
}

func TestUnstow_RemovesDirLink(t *testing.T) {
	// Folded directory symlink gets removed on unstow.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"config/.config/dotfiles/dotfiles.ini": "[dotfiles]",
	})
	if err := os.MkdirAll(filepath.Join(tgtDir, ".config"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "config"); err != nil {
		t.Fatalf("Stow: %v", err)
	}
	if err := Unstow(cfg, "config"); err != nil {
		t.Fatalf("Unstow: %v", err)
	}

	assertAbsent(t, filepath.Join(tgtDir, ".config", "dotfiles"))
}

func TestUnstow_LeavesOtherPackageLinks(t *testing.T) {
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"zsh/.zshrc": "# rc",
		"vim/.vimrc": "set nu",
	})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "zsh"); err != nil {
		t.Fatalf("Stow zsh: %v", err)
	}
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("Stow vim: %v", err)
	}
	if err := Unstow(cfg, "zsh"); err != nil {
		t.Fatalf("Unstow zsh: %v", err)
	}

	assertAbsent(t, filepath.Join(tgtDir, ".zshrc"))
	assertSymlink(t,
		filepath.Join(tgtDir, ".vimrc"),
		filepath.Join(stowDir, "vim", ".vimrc"),
	)
}

func TestUnstow_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"vim/.vimrc": "set nu"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("Stow: %v", err)
	}
	if err := Unstow(cfg, "vim"); err != nil {
		t.Fatalf("first Unstow: %v", err)
	}
	if err := Unstow(cfg, "vim"); err != nil {
		t.Fatalf("second Unstow (idempotent): %v", err)
	}
}

func TestUnstow_DeepFileLinks(t *testing.T) {
	// Individual file links inside real dirs get removed.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"pkg/.config/zsh/rc":  "# zsh",
		"pkg/.config/zsh/env": "export X=1",
	})
	if err := os.MkdirAll(filepath.Join(tgtDir, ".config", "zsh"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "pkg"); err != nil {
		t.Fatalf("Stow: %v", err)
	}
	if err := Unstow(cfg, "pkg"); err != nil {
		t.Fatalf("Unstow: %v", err)
	}

	assertAbsent(t, filepath.Join(tgtDir, ".config", "zsh", "rc"))
	assertAbsent(t, filepath.Join(tgtDir, ".config", "zsh", "env"))
}

// ---------------------------------------------------------------------------
// Restow tests
// ---------------------------------------------------------------------------

func TestRestow_LinksNewFile(t *testing.T) {
	// After adding a file to the package, restow picks it up.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"vim/.vimrc": "set nu"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "vim"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	// Add a new file to the package
	makeTree(t, stowDir, map[string]string{"vim/.vim/colors/monokai.vim": "hi!"})

	if err := Restow(cfg, "vim"); err != nil {
		t.Fatalf("Restow: %v", err)
	}

	assertSymlink(t,
		filepath.Join(tgtDir, ".vimrc"),
		filepath.Join(stowDir, "vim", ".vimrc"),
	)
	// New subdirectory folded
	assertSymlink(t,
		filepath.Join(tgtDir, ".vim"),
		filepath.Join(stowDir, "vim", ".vim"),
	)
}

// ---------------------------------------------------------------------------
// Dry-run tests
// ---------------------------------------------------------------------------

func TestDryRun_NoChanges(t *testing.T) {
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{"zsh/.zshrc": "# rc"})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{StowDir: stowDir, TargetDir: tgtDir, DryRun: true}
	if err := Stow(cfg, "zsh"); err != nil {
		t.Fatalf("Stow (dry run): %v", err)
	}

	assertAbsent(t, filepath.Join(tgtDir, ".zshrc"))
}

// ---------------------------------------------------------------------------
// Multi-package tests
// ---------------------------------------------------------------------------

func TestStow_MultiplePackages(t *testing.T) {
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"zsh/.zshrc":      "# rc",
		"vim/.vimrc":      "set nu",
		"tmux/.tmux.conf": "set -g default-terminal screen-256color",
	})
	if err := os.MkdirAll(tgtDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	for _, pkg := range []string{"zsh", "vim", "tmux"} {
		if err := Stow(cfg, pkg); err != nil {
			t.Fatalf("Stow %s: %v", pkg, err)
		}
	}

	assertSymlink(t, filepath.Join(tgtDir, ".zshrc"), filepath.Join(stowDir, "zsh", ".zshrc"))
	assertSymlink(t, filepath.Join(tgtDir, ".vimrc"), filepath.Join(stowDir, "vim", ".vimrc"))
	assertSymlink(t, filepath.Join(tgtDir, ".tmux.conf"), filepath.Join(stowDir, "tmux", ".tmux.conf"))
}

func TestStow_DeeplyNested(t *testing.T) {
	// .config and .config/dotfiles both exist as real dirs → recurse to file level.
	tmp := t.TempDir()
	stowDir := filepath.Join(tmp, "dotfiles")
	tgtDir := filepath.Join(tmp, "home")
	makeTree(t, stowDir, map[string]string{
		"config/.config/dotfiles/dotfiles.ini": "[dotfiles]",
		"config/.config/dotfiles/machine.ini":  "[machine]",
	})
	if err := os.MkdirAll(filepath.Join(tgtDir, ".config", "dotfiles"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := newCfg(stowDir, tgtDir)
	if err := Stow(cfg, "config"); err != nil {
		t.Fatalf("Stow: %v", err)
	}

	assertSymlink(t,
		filepath.Join(tgtDir, ".config", "dotfiles", "dotfiles.ini"),
		filepath.Join(stowDir, "config", ".config", "dotfiles", "dotfiles.ini"),
	)
	assertSymlink(t,
		filepath.Join(tgtDir, ".config", "dotfiles", "machine.ini"),
		filepath.Join(stowDir, "config", ".config", "dotfiles", "machine.ini"),
	)
}
