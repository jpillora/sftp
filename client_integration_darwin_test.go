package sftp

import (
	"syscall"
	"testing"
)

func TestClientStatVFS(t *testing.T) {
	if *testServerImpl {
		t.Skipf("go server does not support FXP_EXTENDED")
	}
	sftp, cmd := testClient(t, READWRITE, NODELAY)
	defer cmd.Wait()
	defer sftp.Close()

	vfs, err := sftp.StatVFS("/")
	if err != nil {
		t.Fatal(err)
	}

	// get system stats
	s := syscall.Statfs_t{}
	err = syscall.Statfs("/", &s)
	if err != nil {
		t.Fatal(err)
	}

	// Compare stable filesystem properties. Free blocks and inodes can change
	// between the remote and local statfs calls on a busy system.
	if vfs.Frsize != uint64(s.Bsize) {
		t.Fatalf("fr_size does not match, expected: %v, got: %v", s.Bsize, vfs.Frsize)
	}

	if vfs.Bsize != uint64(s.Iosize) {
		t.Fatalf("f_bsize does not match, expected: %v, got: %v", s.Iosize, vfs.Bsize)
	}

}
