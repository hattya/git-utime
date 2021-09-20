//
// git-utime :: utime_windows_test.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLutimes(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	for _, name := range []string{"name\x00", "_"} {
		if err := lutimes(filepath.Join(dir, name), now, now); err == nil {
			t.Error("expected error")
		}
	}
}
