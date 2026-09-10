//go:build windows

package sftp

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

func setOpenFileTimes(file any, atime, mtime time.Time) error {
	if operation, ok := file.(interface {
		Chtimes(time.Time, time.Time) error
	}); ok {
		return operation.Chtimes(atime, mtime)
	}
	descriptor, ok := file.(interface{ Fd() uintptr })
	if !ok {
		return errors.ErrUnsupported
	}
	access := windows.NsecToFiletime(atime.UnixNano())
	write := windows.NsecToFiletime(mtime.UnixNano())
	return windows.SetFileTime(windows.Handle(descriptor.Fd()), nil, &access, &write)
}
