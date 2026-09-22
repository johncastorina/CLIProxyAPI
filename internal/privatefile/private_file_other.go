//go:build !unix && !windows

package privatefile

import (
	"fmt"
	"os"
)

func openPrivateFileNoFollow(path string, flag int) (*os.File, error) {
	// These platforms do not expose an atomic no-follow open through the Go
	// standard library. This check is best effort; Unix and Windows use
	// descriptor-based final-component rejection instead.
	info, errLstat := os.Lstat(path)
	if errLstat == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("private file %q is a symlink", path)
	}
	if errLstat != nil && !os.IsNotExist(errLstat) {
		return nil, errLstat
	}
	return os.OpenFile(path, flag, PrivateFileMode)
}
