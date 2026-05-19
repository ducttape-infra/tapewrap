package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const version = "0.1.0"

func main() {
	var (
		delete  bool
		restow  bool
		target  string
		verbose bool
		dryRun  bool
		ver     bool
	)

	flag.BoolVar(&delete, "D", false, "Delete (unstow) packages")
	flag.BoolVar(&restow, "R", false, "Restow packages (unstow then stow)")
	flag.StringVar(&target, "t", "", "Target directory (default: parent of stow dir)")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&dryRun, "n", false, "Dry run — show actions without performing them")
	flag.BoolVar(&ver, "version", false, "Print version and exit")
	flag.Parse()

	if ver {
		fmt.Printf("gstow %s\n", version)
		os.Exit(0)
	}

	packages := flag.Args()
	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "error: no packages specified")
		fmt.Fprintln(os.Stderr, "usage: gstow [-D] [-R] [-t TARGET] [-v] [-n] PACKAGE...")
		os.Exit(1)
	}

	stowDir, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if target == "" {
		target = filepath.Dir(stowDir)
	}
	targetDir, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving target: %v\n", err)
		os.Exit(1)
	}

	cfg := &Config{
		StowDir:   stowDir,
		TargetDir: targetDir,
		Verbose:   verbose,
		DryRun:    dryRun,
	}

	if verbose || dryRun {
		fmt.Printf("Stow directory:  %s\n", stowDir)
		fmt.Printf("Target directory: %s\n", targetDir)
		if dryRun {
			fmt.Println("(dry run — no changes will be made)")
		}
	}

	exitCode := 0
	for _, pkg := range packages {
		pkgDir := filepath.Join(stowDir, pkg)
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error: package directory %q does not exist\n", pkgDir)
			exitCode = 1
			continue
		}

		var opErr error
		switch {
		case restow:
			opErr = Restow(cfg, pkg)
		case delete:
			opErr = Unstow(cfg, pkg)
		default:
			opErr = Stow(cfg, pkg)
		}

		if opErr != nil {
			fmt.Fprintf(os.Stderr, "error processing %q: %v\n", pkg, opErr)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
