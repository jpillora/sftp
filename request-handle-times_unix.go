//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos

package sftp

import (
	"errors"
	"time"

	"golang.org/x/sys/unix"
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
	times := []unix.Timeval{
		unix.NsecToTimeval(atime.UnixNano()),
		unix.NsecToTimeval(mtime.UnixNano()),
	}
	return unix.Futimes(int(descriptor.Fd()), times)
}
