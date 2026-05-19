//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procCreateJunction   = modkernel32.NewProc("CreateDirectoryW") // placeholder; junctions via DeviceIoControl
	procCreateSymbolicLink = modkernel32.NewProc("CreateSymbolicLinkW")
)

const (
	symlinkFlagDirectory = 0x1
	symlinkFlagAllowUnprivileged = 0x2 // requires Developer Mode (Windows 10 1703+)
)

// createDirLink attempts to create a directory symlink on Windows.
// Falls back to an NTFS junction point if symlink creation is denied.
func createDirLink(src, tgt string) error {
	// Try symlink first (works with Developer Mode or elevated privileges)
	err := os.Symlink(src, tgt)
	if err == nil {
		return nil
	}

	// Fall back to NTFS junction if symlink creation is not permitted
	if isPrivilegeError(err) {
		return createJunction(src, tgt)
	}
	return err
}

// removeDirLink removes a symlink or junction on Windows.
func removeDirLink(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	// Junction points appear as directories; use RemoveAll-equivalent for single entry
	return syscall.RemoveDirectory(syscall.StringToUTF16Ptr(path))
}

func isPrivilegeError(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		// ERROR_PRIVILEGE_NOT_HELD or ERROR_ACCESS_DENIED
		return errno == 1314 || errno == 5
	}
	return false
}

// createJunction creates an NTFS junction point (directory reparse point).
// Junctions don't require elevated privileges and work on all Windows versions.
func createJunction(src, tgt string) error {
	// Create the target directory first
	if err := os.MkdirAll(tgt, 0755); err != nil {
		return err
	}

	tgtPtr, err := syscall.UTF16PtrFromString(`\??\` + src)
	if err != nil {
		return err
	}

	// REPARSE_DATA_BUFFER layout for junction points
	type reparseDataBuffer struct {
		ReparseTag        uint32
		ReparseDataLength uint16
		Reserved          uint16
		SubstituteNameOffset uint16
		SubstituteNameLength uint16
		PrintNameOffset   uint16
		PrintNameLength   uint16
		PathBuffer        [1]uint16
	}

	srcUTF16 := syscall.StringToUTF16(`\??\` + src)
	srcLen := len(srcUTF16)*2 - 2 // bytes, excluding null terminator

	bufSize := int(unsafe.Sizeof(reparseDataBuffer{})) + srcLen*2
	buf := make([]byte, bufSize)

	rdb := (*reparseDataBuffer)(unsafe.Pointer(&buf[0]))
	rdb.ReparseTag = 0xA0000003 // IO_REPARSE_TAG_MOUNT_POINT
	rdb.ReparseDataLength = uint16(8 + srcLen*2 + 2)
	rdb.SubstituteNameOffset = 0
	rdb.SubstituteNameLength = uint16(srcLen)
	rdb.PrintNameOffset = uint16(srcLen + 2)
	rdb.PrintNameLength = 0

	copy(buf[unsafe.Sizeof(reparseDataBuffer{}):], (*[1 << 20]byte)(unsafe.Pointer(&srcUTF16[0]))[:srcLen])
	_ = tgtPtr

	handle, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr(tgt),
		syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	var bytesReturned uint32
	return syscall.DeviceIoControl(
		handle,
		0x000900A4, // FSCTL_SET_REPARSE_POINT
		&buf[0],
		uint32(bufSize),
		nil,
		0,
		&bytesReturned,
		nil,
	)
}
