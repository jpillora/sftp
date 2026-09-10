package sftp

import (
	"errors"
	"os"
)

// These operations deliberately use the object returned by the open handler.
// Reopening Request.Filepath is not equivalent: the file may have been renamed
// or unlinked, and a different file may now exist at that path.
func (r *Request) fstat(pkt *sshFxpFstatPacket) responsePacket {
	file, ok := r.openObject().(interface {
		Stat() (os.FileInfo, error)
	})
	if !ok {
		return statusFromError(pkt.ID, ErrSSHFxOpUnsupported)
	}
	info, err := file.Stat()
	if err != nil {
		return statusFromError(pkt.ID, err)
	}
	return &sshFxpStatResponse{ID: pkt.ID, info: info}
}

func (r *Request) fsetstat(pkt *sshFxpFsetstatPacket) responsePacket {
	file := r.openObject()
	if file == nil {
		return statusFromError(pkt.ID, EBADF)
	}
	attrs, err := pkt.unmarshalFileStat(pkt.Flags)
	if err != nil {
		return statusFromError(pkt.ID, err)
	}
	if pkt.Flags&sshFileXferAttrSize != 0 {
		operation, ok := file.(interface{ Truncate(int64) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported)
		}
		err = operation.Truncate(int64(attrs.Size))
	}
	if err == nil && pkt.Flags&sshFileXferAttrPermissions != 0 {
		operation, ok := file.(interface{ Chmod(os.FileMode) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported)
		}
		err = operation.Chmod(attrs.FileMode())
	}
	if err == nil && pkt.Flags&sshFileXferAttrUIDGID != 0 {
		operation, ok := file.(interface{ Chown(int, int) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported)
		}
		err = operation.Chown(int(attrs.UID), int(attrs.GID))
	}
	if err == nil && pkt.Flags&sshFileXferAttrACmodTime != 0 {
		err = setOpenFileTimes(file, attrs.AccessTime(), attrs.ModTime())
	}
	if errors.Is(err, errors.ErrUnsupported) {
		err = ErrSSHFxOpUnsupported
	}
	return statusFromError(pkt.ID, err)
}
