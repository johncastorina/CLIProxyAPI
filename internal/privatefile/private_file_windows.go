//go:build windows

package privatefile

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openPrivateFileNoFollow(path string, flag int) (*os.File, error) {
	pathPtr, errPath := windows.UTF16PtrFromString(path)
	if errPath != nil {
		return nil, errPath
	}
	access := uint32(windows.GENERIC_READ)
	if flag&os.O_WRONLY != 0 {
		if flag&os.O_APPEND != 0 {
			access = windows.FILE_APPEND_DATA | windows.FILE_WRITE_ATTRIBUTES
		} else {
			access = windows.GENERIC_WRITE
		}
	} else if flag&os.O_RDWR != 0 {
		if flag&os.O_APPEND != 0 {
			access = windows.GENERIC_READ | windows.FILE_APPEND_DATA | windows.FILE_WRITE_ATTRIBUTES
		} else {
			access = windows.GENERIC_READ | windows.GENERIC_WRITE
		}
	}
	creation := uint32(windows.OPEN_EXISTING)
	if flag&os.O_CREATE != 0 {
		creation = windows.OPEN_ALWAYS
		if flag&os.O_EXCL != 0 {
			creation = windows.CREATE_NEW
		}
	}
	handle, errOpen := windows.CreateFile(
		pathPtr,
		access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		creation,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if errOpen != nil {
		return nil, errOpen
	}
	var info windows.ByHandleFileInformation
	if errInfo := windows.GetFileInformationByHandle(handle, &info); errInfo != nil {
		_ = windows.CloseHandle(handle)
		return nil, errInfo
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("private file %q is a reparse point", path)
	}
	// Windows keeps the destination's inherited ACL. PrivateFileMode only maps
	// to the limited read-only attribute behavior exposed by os.File.Chmod.
	return os.NewFile(uintptr(handle), path), nil
}
