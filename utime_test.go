//
// git-utime :: utime_test.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const iso8601 = "2006-01-02T15:04:05"

type fileTest struct {
	mtime, path string
}

func init() {
	stdout = ioutil.Discard
	stderr = ioutil.Discard
}

func TestNoRepo(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	if _, err := getwt(); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ls(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmptyRepo(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	init_()

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	files, err := ls(wt)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(files) != 0:
		t.Fatalf("expected empty, got %v", files)
	}
	if err := utime(wt, files); err != nil {
		t.Fatal(err)
	}
}

func TestCommits(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	var log []string
	if err := init_(); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T12:00:00")
	if err := touch("foo"); err != nil {
		t.Fatal(err)
	}
	if err := mkdir("bar"); err != nil {
		t.Fatal(err)
	}
	if err := touch("bar", "foo"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[0]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T13:00:00")
	if err := touch("bar", "bar"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[1]); err != nil {
		t.Fatal(err)
	}

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	files, err := ls(wt)
	if err != nil {
		t.Fatal(err)
	}
	if err := utime(wt, files); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[0], "foo"},
		{log[0], filepath.Join("bar", "foo")},
		{log[1], filepath.Join("bar", "bar")},
		{log[1], "bar"},
		{log[1], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}

	// modify
	now := "2021-07-07T23:59:59"
	tm, err := time.ParseInLocation(iso8601, now, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if err := file("foo", "."); err != nil {
		t.Fatal(err)
	}
	if err := lutimes("foo", tm, tm); err != nil {
		t.Fatal(err)
	}
	if err := mv(filepath.Join("bar", "bar"), filepath.Join("bar", "baz")); err != nil {
		t.Fatal(err)
	}

	files, err = ls(wt)
	if err != nil {
		t.Fatal(err)
	}
	if err := utime(wt, files); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[0], filepath.Join("bar", "foo")},
		{log[0], "bar"},
		{log[0], "."},
		{log[1], filepath.Join("bar", "baz")},
		{now, "foo"},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
}

func TestMergeCommits(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	var log []string
	if err := init_(); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T12:00:00")
	if err := file("foo", "1"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[0]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T13:00:00")
	if err := checkout("-b", "ff"); err != nil {
		t.Fatal(err)
	}
	if err := file("foo", "2"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[1]); err != nil {
		t.Fatal(err)
	}
	// merge
	if err := checkout("master"); err != nil {
		t.Fatal(err)
	}
	if err := merge("ff"); err != nil {
		t.Fatal(err)
	}

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	files, err := ls(wt)
	if err != nil {
		t.Fatal(err)
	}
	if err := utime(wt, files); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[1], "foo"},
		{log[1], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}

	// commit
	log = append(log, "2021-07-07T14:00:00")
	if err := touch("bar"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[2]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T15:00:00")
	if err := file("bar", "1\n2\n3\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[3]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T16:00:00")
	if err := checkout("-b", "no-ff", "@~1"); err != nil {
		t.Fatal(err)
	}
	if err := file("bar", "3\n2\n1\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[4]); err != nil {
		t.Fatal(err)
	}
	// merge
	log = append(log, "2021-07-07T17:00:00")
	if err := checkout("master"); err != nil {
		t.Fatal(err)
	}
	if err := merge("no-ff"); err == nil {
		t.Fatal("expected merge conflict")
	}
	if err := file("bar", "Let's Go!\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[5]); err != nil {
		t.Fatal(err)
	}

	files, err = ls(wt)
	if err != nil {
		t.Fatal(err)
	}
	if err := utime(wt, files); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[1], "foo"},
		{log[4], "bar"},
		{log[4], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
}

func init_() error {
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Utime"},
		{"config", "user.email", "utime@example.com"},
	} {
		if err := exec.Command("git", args...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func checkout(a ...string) error {
	return exec.Command("git", append([]string{"checkout"}, a...)...).Run()
}

func commit(date string) error {
	os.Setenv("GIT_AUTHOR_DATE", date)
	os.Setenv("GIT_COMMITTER_DATE", date)
	defer os.Unsetenv("GIT_AUTHOR_DATE")
	defer os.Unsetenv("GIT_COMMITTER_DATE")

	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "."},
	} {
		if err := exec.Command("git", args...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func merge(name string) error {
	return exec.Command("git", "merge", "--no-ff", name).Run()
}

func mv(oldpath, newpath string) error {
	return exec.Command("git", "mv", oldpath, newpath).Run()
}

func file(name, data string) error {
	return ioutil.WriteFile(name, []byte(data), 0o666)
}

func mkdir(s ...string) error {
	return os.MkdirAll(filepath.Join(s...), 0o777)
}

func pushd(path string) (func() error, error) {
	wd, err := os.Getwd()
	popd := func() error {
		if err != nil {
			return err
		}
		return os.Chdir(wd)
	}
	return popd, os.Chdir(path)
}

func stat(path string) string {
	fi, _ := os.Lstat(path)
	return fi.ModTime().Format(iso8601)
}

func tempDir() (string, error) {
	return ioutil.TempDir("", "git-utime")
}

func touch(s ...string) error {
	return ioutil.WriteFile(filepath.Join(s...), []byte{}, 0666)
}
