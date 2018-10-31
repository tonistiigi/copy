// +build solaris darwin freebsd

package fs

import (
	"io"
	"os"
	"syscall"

	"github.com/containerd/containerd/sys"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func (c *copier) copyFileInfo(fi os.FileInfo, name string) error {
	st := fi.Sys().(*syscall.Stat_t)
	uid, gid := int(st.Uid), int(st.Gid)
	if c.chown != nil {
		uid, gid = c.chown.Uid, c.chown.Gid
	}
	if err := os.Lchown(name, uid, gid); err != nil {
		return errors.Wrapf(err, "failed to chown %s", name)
	}

	if (fi.Mode() & os.ModeSymlink) != os.ModeSymlink {
		if err := os.Chmod(name, fi.Mode()); err != nil {
			return errors.Wrapf(err, "failed to chmod %s", name)
		}
	}

	timespec := []syscall.Timespec{sys.StatAtime(st), sys.StatMtime(st)}
	if err := syscall.UtimesNano(name, timespec); err != nil {
		return errors.Wrapf(err, "failed to utime %s", name)
	}

	return nil
}

func copyFileContent(dst, src *os.File) error {
	buf := bufferPool.Get().(*[]byte)
	_, err := io.CopyBuffer(dst, src, *buf)
	bufferPool.Put(buf)

	return err
}

func copyDevice(dst string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("unsupported stat type")
	}
	return unix.Mknod(dst, uint32(fi.Mode()), int(st.Rdev))
}
