//
// git-utime :: utime_windows.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"time"

	"golang.org/x/sys/windows"
)

func lutimes(name string, atime, mtime time.Time) error {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.FILE_WRITE_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	a := windows.NsecToFiletime(atime.UnixNano())
	w := windows.NsecToFiletime(mtime.UnixNano())
	return windows.SetFileTime(h, nil, &a, &w)
}
