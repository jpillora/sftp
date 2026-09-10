package sftp

import (
	"io"
	"os"
	"testing"
	"time"
)

type metadataLister struct {
	mode os.FileMode
}

func (*metadataLister) ListAt([]os.FileInfo, int64) (int, error) { return 0, io.EOF }
func (*metadataLister) Stat() (os.FileInfo, error)               { return staticFileInfo{}, nil }
func (l *metadataLister) Chmod(mode os.FileMode) error {
	l.mode = mode
	return nil
}

type staticFileInfo struct{}

func (staticFileInfo) Name() string       { return "directory" }
func (staticFileInfo) Size() int64        { return 0 }
func (staticFileInfo) Mode() os.FileMode  { return os.ModeDir | 0o755 }
func (staticFileInfo) ModTime() time.Time { return time.Unix(1, 0) }
func (staticFileInfo) IsDir() bool        { return true }
func (staticFileInfo) Sys() any           { return nil }

type truncateOnlyFile struct {
	metadataLister
	truncates int
}

func (*truncateOnlyFile) WriteAt([]byte, int64) (int, error) { return 0, nil }
func (f *truncateOnlyFile) Truncate(int64) error {
	f.truncates++
	return nil
}

func TestDirectoryHandleMetadataUsesListerObject(t *testing.T) {
	lister := &metadataLister{}
	request := &Request{}
	request.setListerAt(lister)

	response, handled := request.fstat(&sshFxpFstatPacket{ID: 1})
	if !handled {
		t.Fatal("FSTAT was not handled through the directory object")
	}
	stat, ok := response.(*sshFxpStatResponse)
	if !ok || !stat.info.IsDir() {
		t.Fatalf("FSTAT response = %#v", response)
	}

	response, handled = request.fsetstat(&sshFxpFsetstatPacket{
		ID:    2,
		Flags: sshFileXferAttrPermissions,
		Attrs: &FileStat{Mode: fromFileMode(0o710)},
	})
	if !handled {
		t.Fatal("FSETSTAT was not handled through the directory object")
	}
	status, ok := response.(*sshFxpStatusPacket)
	if !ok || status.Code != sshFxOk || lister.mode.Perm() != 0o710 {
		t.Fatalf("FSETSTAT response = %#v, chmod = %v", response, lister.mode)
	}
}

func TestPartialHandleMetadataSupportDoesNotFallBackOrMutate(t *testing.T) {
	file := &truncateOnlyFile{}
	request := &Request{}
	request.setWriterAt(file)

	response, handled := request.fsetstat(&sshFxpFsetstatPacket{
		ID:    3,
		Flags: sshFileXferAttrSize | sshFileXferAttrUIDGID,
		Attrs: &FileStat{Size: 5, UID: 1, GID: 1},
	})
	if !handled {
		t.Fatal("partial live-object support must not fall back to a pathname")
	}
	status, ok := response.(*sshFxpStatusPacket)
	if !ok || status.Code != sshFxOPUnsupported {
		t.Fatalf("FSETSTAT response = %#v", response)
	}
	if file.truncates != 0 {
		t.Fatalf("partial FSETSTAT mutated the live object %d times", file.truncates)
	}
}
