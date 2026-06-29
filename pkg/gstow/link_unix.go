//go:build !windows

package gstow

import "os"

// createDirLink creates a symlink for a directory on Unix.
func createDirLink(src, tgt string) error {
	return os.Symlink(src, tgt)
}

// removeDirLink removes a symlink (or symlinked directory) on Unix.
func removeDirLink(path string, _ os.FileInfo) error {
	return os.Remove(path)
}
