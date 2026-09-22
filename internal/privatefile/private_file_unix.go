//go:build unix

package privatefile

import (
	"os"

	"golang.org/x/sys/unix"
)

func openPrivateFileNoFollow(path string, flag int) (*os.File, error) {
	fd, errOpen := unix.Open(path, flag|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(PrivateFileMode.Perm()))
	if errOpen != nil {
		return nil, errOpen
	}
	return os.NewFile(uintptr(fd), path), nil
}
