package sftp

import (
	"errors"
	"os"
)

// These operations prefer the object returned by the open handler. Reopening
// Request.Filepath is not equivalent: the file may have been renamed or
// unlinked, and a different file may now exist at that path. The bool reports
// whether the live object supports the operation; RequestServer otherwise
// falls back to its historical FileList/FileCmd dispatch.
func (r *Request) fstat(pkt *sshFxpFstatPacket) (responsePacket, bool) {
	file, ok := r.openObject().(interface {
		Stat() (os.FileInfo, error)
	})
	if !ok {
		return nil, false
	}
	info, err := file.Stat()
	if err != nil {
		return statusFromError(pkt.ID, err), true
	}
	return &sshFxpStatResponse{ID: pkt.ID, info: info}, true
}

func (r *Request) fsetstat(pkt *sshFxpFsetstatPacket) (responsePacket, bool) {
	file := r.openObject()
	complete, partial := openFileSetstatSupport(file, pkt.Flags)
	if partial {
		return statusFromError(pkt.ID, ErrSSHFxOpUnsupported), true
	}
	if !complete {
		return nil, false
	}
	attrs, err := pkt.unmarshalFileStat(pkt.Flags)
	if err != nil {
		return statusFromError(pkt.ID, err), true
	}
	if pkt.Flags&sshFileXferAttrSize != 0 {
		operation, ok := file.(interface{ Truncate(int64) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported), true
		}
		err = operation.Truncate(int64(attrs.Size))
	}
	if err == nil && pkt.Flags&sshFileXferAttrPermissions != 0 {
		operation, ok := file.(interface{ Chmod(os.FileMode) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported), true
		}
		err = operation.Chmod(attrs.FileMode())
	}
	if err == nil && pkt.Flags&sshFileXferAttrUIDGID != 0 {
		operation, ok := file.(interface{ Chown(int, int) error })
		if !ok {
			return statusFromError(pkt.ID, ErrSSHFxOpUnsupported), true
		}
		err = operation.Chown(int(attrs.UID), int(attrs.GID))
	}
	if err == nil && pkt.Flags&sshFileXferAttrACmodTime != 0 {
		err = setOpenFileTimes(file, attrs.AccessTime(), attrs.ModTime())
	}
	if errors.Is(err, errors.ErrUnsupported) {
		err = ErrSSHFxOpUnsupported
	}
	return statusFromError(pkt.ID, err), true
}

func openFileSetstatSupport(file any, flags uint32) (complete, partial bool) {
	const supportedFlags = sshFileXferAttrSize |
		sshFileXferAttrUIDGID |
		sshFileXferAttrPermissions |
		sshFileXferAttrACmodTime

	if file == nil || flags == 0 {
		return false, false
	}
	anySupported := false
	anyUnsupported := flags&^supportedFlags != 0
	if flags&sshFileXferAttrSize != 0 {
		_, ok := file.(interface{ Truncate(int64) error })
		anySupported = anySupported || ok
		anyUnsupported = anyUnsupported || !ok
	}
	if flags&sshFileXferAttrPermissions != 0 {
		_, ok := file.(interface{ Chmod(os.FileMode) error })
		anySupported = anySupported || ok
		anyUnsupported = anyUnsupported || !ok
	}
	if flags&sshFileXferAttrUIDGID != 0 {
		_, ok := file.(interface{ Chown(int, int) error })
		anySupported = anySupported || ok
		anyUnsupported = anyUnsupported || !ok
	}
	if flags&sshFileXferAttrACmodTime != 0 {
		ok := supportsOpenFileTimes(file)
		anySupported = anySupported || ok
		anyUnsupported = anyUnsupported || !ok
	}
	return anySupported && !anyUnsupported, anySupported && anyUnsupported
}
