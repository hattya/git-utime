//
// git-utime :: utime_unix.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

//go:build !plan9 && !windows
// +build !plan9,!windows

package main

import (
	"time"

	"golang.org/x/sys/unix"
)

func lutimes(name string, atime, mtime time.Time) error {
	return unix.Lutimes(name, []unix.Timeval{
		unix.NsecToTimeval(atime.UnixNano()),
		unix.NsecToTimeval(mtime.UnixNano()),
	})
}
