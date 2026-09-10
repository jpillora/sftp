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

func TestDirectoryHandleMetadataUsesListerObject(t *testing.T) {
	lister := &metadataLister{}
	request := &Request{}
	request.setListerAt(lister)

	response := request.fstat(&sshFxpFstatPacket{ID: 1})
	stat, ok := response.(*sshFxpStatResponse)
	if !ok || !stat.info.IsDir() {
		t.Fatalf("FSTAT response = %#v", response)
	}

	response = request.fsetstat(&sshFxpFsetstatPacket{
		ID:    2,
		Flags: sshFileXferAttrPermissions,
		Attrs: &FileStat{Mode: fromFileMode(0o710)},
	})
	status, ok := response.(*sshFxpStatusPacket)
	if !ok || status.Code != sshFxOk || lister.mode.Perm() != 0o710 {
		t.Fatalf("FSETSTAT response = %#v, chmod = %v", response, lister.mode)
	}
}
