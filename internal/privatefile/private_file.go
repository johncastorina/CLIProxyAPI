package privatefile

import (
	"fmt"
	"io"
	"os"
)

const (
	PrivateFileMode os.FileMode = 0o600
	PrivateDirMode  os.FileMode = 0o700
)

// MkdirAll creates missing directories with permissions suitable for
// secrets. Existing directories are left unchanged.
func MkdirAll(path string) error {
	return os.MkdirAll(path, PrivateDirMode)
}

// PrepareExisting rejects a final symlink and tightens an existing regular
// file without creating a missing destination.
func PrepareExisting(path string) error {
	file, errOpen := Open(path, os.O_WRONLY)
	if os.IsNotExist(errOpen) {
		return nil
	}
	if errOpen != nil {
		return errOpen
	}
	return file.Close()
}

// Open opens a regular file without following a final symlink. It
// tightens the descriptor permissions before truncating or writing existing
// content.
func Open(path string, flag int) (*os.File, error) {
	openFlag := flag &^ os.O_TRUNC
	file, errOpen := openPrivateFileNoFollow(path, openFlag)
	if errOpen != nil {
		return nil, errOpen
	}
	closeOnError := func(err error) (*os.File, error) {
		_ = file.Close()
		return nil, err
	}

	info, errStat := file.Stat()
	if errStat != nil {
		return closeOnError(errStat)
	}
	if !info.Mode().IsRegular() {
		return closeOnError(fmt.Errorf("private file %q is not a regular file", path))
	}
	if errChmod := file.Chmod(PrivateFileMode); errChmod != nil {
		return closeOnError(errChmod)
	}
	if flag&os.O_TRUNC != 0 {
		if errTruncate := file.Truncate(0); errTruncate != nil {
			return closeOnError(errTruncate)
		}
	}
	if flag&os.O_APPEND != 0 {
		if _, errSeek := file.Seek(0, io.SeekEnd); errSeek != nil {
			return closeOnError(errSeek)
		}
	}
	return file, nil
}

// WriteFile writes data to a private regular file.
func WriteFile(path string, data []byte) error {
	file, errOpen := Open(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if errOpen != nil {
		return errOpen
	}
	if _, errWrite := file.Write(data); errWrite != nil {
		_ = file.Close()
		return errWrite
	}
	return file.Close()
}
