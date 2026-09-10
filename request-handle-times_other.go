//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows && !zos

package sftp

import (
	"errors"
	"time"
)

func setOpenFileTimes(file any, atime, mtime time.Time) error {
	if operation, ok := file.(interface {
		Chtimes(time.Time, time.Time) error
	}); ok {
		return operation.Chtimes(atime, mtime)
	}
	return errors.ErrUnsupported
}

func supportsOpenFileTimes(file any) bool {
	_, ok := file.(interface {
		Chtimes(time.Time, time.Time) error
	})
	return ok
}
