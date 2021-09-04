//
// git-utime :: utime_test.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeValue(t *testing.T) {
	var s string
	c := newMergeValue(&s, "-c")
	m := newMergeValue(&s, "-m")

	if !c.IsBoolFlag() {
		t.Fatal("expected true, got false")
	}
	if !m.IsBoolFlag() {
		t.Fatal("expected true, got false")
	}

	if c.Set("_") == nil {
		t.Fatal("expected error")
	}
	if m.Set("_") == nil {
		t.Fatal("expected error")
	}

	// set as "-c"
	if err := c.Set("true"); err != nil {
		t.Fatal(err)
	}
	if err := m.Set("false"); err != nil {
		t.Fatal(err)
	}
	if g, e := s, "-c"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
	if g, e := c.Get(), true; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := m.Get(), false; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := c.String(), "true"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := m.String(), "false"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}

	// set as "-m"
	if err := c.Set("false"); err != nil {
		t.Fatal(err)
	}
	if err := m.Set("true"); err != nil {
		t.Fatal(err)
	}
	if g, e := s, "-m"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
	if g, e := c.Get(), false; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := m.Get(), true; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := c.String(), "false"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := m.String(), "true"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}

	// set as ""
	if err := m.Set("false"); err != nil {
		t.Fatal(err)
	}
	if g, e := s, ""; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
	if g, e := m.Get(), false; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
	if g, e := m.String(), "false"; g != e {
		t.Errorf("expected %v, got %v", e, g)
	}
}

const iso8601 = "2006-01-02T15:04:05"

type fileTest struct {
	mtime, path string
}

func init() {
	stdout = ioutil.Discard
	stderr = ioutil.Discard
	*recurse = true
}

func TestNoRepo(t *testing.T) {
	dir := t.TempDir()
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	if _, err := getwt(); err == nil {
		t.Fatal("expected error")
	}
	if _, err := submodules(dir); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ls(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestEmptyRepo(t *testing.T) {
	popd, err := pushd(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	init_()

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	mods, err := submodules(wt)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(mods) != 0:
		t.Fatalf("expected empty, got %v", mods)
	}
	files, err := ls(wt)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(files) != 0:
		t.Fatalf("expected empty, got %v", files)
	}
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
}

func TestCommits(t *testing.T) {
	popd, err := pushd(t.TempDir())
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
	if err := utimeAll(wt); err != nil {
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

	if err := utimeAll(wt); err != nil {
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
	popd, err := pushd(t.TempDir())
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
	log = append(log, "2021-07-07T14:00:00")
	if err := checkout("master"); err != nil {
		t.Fatal(err)
	}
	if err := merge("ff", log[2]); err != nil {
		t.Fatal(err)
	}

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	if err := utimeAll(wt); err != nil {
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
	// specify -c
	flag.Set("c", "true")
	if err := utimeAll(wt); err != nil {
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
	// specify -m
	flag.Set("m", "true")
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[2], "foo"},
		{log[2], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
	// reset
	flag.Set("c", "false")
	flag.Set("m", "false")

	// commit
	log = append(log, "2021-07-07T15:00:00")
	if err := touch("bar"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[3]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T16:00:00")
	if err := file("bar", "1\n2\n3\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[4]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T17:00:00")
	if err := checkout("-b", "no-ff", "@~1"); err != nil {
		t.Fatal(err)
	}
	if err := file("bar", "3\n2\n1\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[5]); err != nil {
		t.Fatal(err)
	}
	// merge
	log = append(log, "2021-07-07T18:00:00")
	if err := checkout("master"); err != nil {
		t.Fatal(err)
	}
	if err := merge("no-ff", log[6]); err == nil {
		t.Fatal("expected merge conflict")
	}
	if err := file("bar", "Let's Go!\n"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[6]); err != nil {
		t.Fatal(err)
	}

	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[1], "foo"},
		{log[5], "bar"},
		{log[5], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
	// specify -c
	flag.Set("c", "true")
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[1], "foo"},
		{log[6], "bar"},
		{log[6], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
	// specify -m
	flag.Set("m", "true")
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[2], "foo"},
		{log[6], "bar"},
		{log[6], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}
	// reset
	flag.Set("c", "false")
	flag.Set("m", "false")
}

func TestSubmodule(t *testing.T) {
	popd, err := pushd(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer popd()

	var log []string
	// repository: baz
	if err := mkdir("baz"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("baz"); err != nil {
		t.Fatal(err)
	}
	if err := init_(); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T12:00:00")
	if err := touch("file"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[0]); err != nil {
		t.Fatal(err)
	}
	// popd
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}

	// repository: bar
	if err := mkdir("bar"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("bar"); err != nil {
		t.Fatal(err)
	}
	if err := init_(); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T13:00:00")
	if err := touch("file"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[1]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T14:00:00")
	if err := addSubmodule("../baz", "baz"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[2]); err != nil {
		t.Fatal(err)
	}
	// popd
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}

	// repository foo
	if err := mkdir("foo"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("foo"); err != nil {
		t.Fatal(err)
	}
	if err := init_(); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T15:00:00")
	if err := touch("file"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[3]); err != nil {
		t.Fatal(err)
	}
	// commit
	log = append(log, "2021-07-07T16:00:00")
	if err := addSubmodule("../bar", "bar"); err != nil {
		t.Fatal(err)
	}
	if err := commit(log[4]); err != nil {
		t.Fatal(err)
	}

	wt, err := getwt()
	if err != nil {
		t.Fatal(err)
	}
	mods, err := submodules(wt)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(mods) != 1:
		t.Errorf("expected 1, got %v", mods)
	}
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[1], filepath.Join("bar", "file")},
		{log[2], filepath.Join("bar", "baz")},
		{log[3], "file"},
		{log[4], "bar"},
		{log[4], "."},
	} {
		if mtime := stat(tt.path); mtime != tt.mtime {
			t.Errorf("%v: expected %v, got %v", tt.path, tt.mtime, mtime)
		}
	}

	if err := initSubmodules(); err != nil {
		t.Fatal(err)
	}

	mods, err = submodules(wt)
	switch {
	case err != nil:
		t.Fatal(err)
	case len(mods) != 2:
		t.Errorf("expected 1, got %v", mods)
	}
	if err := utimeAll(wt); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []fileTest{
		{log[0], filepath.Join("bar", "baz", "file")},
		{log[1], filepath.Join("bar", "file")},
		{log[2], filepath.Join("bar", "baz")},
		{log[3], "file"},
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

func merge(name, date string) error {
	os.Setenv("GIT_AUTHOR_DATE", date)
	os.Setenv("GIT_COMMITTER_DATE", date)
	defer os.Unsetenv("GIT_AUTHOR_DATE")
	defer os.Unsetenv("GIT_COMMITTER_DATE")

	return exec.Command("git", "merge", "--no-ff", name).Run()
}

func mv(oldpath, newpath string) error {
	return exec.Command("git", "mv", oldpath, newpath).Run()
}

func addSubmodule(repo, path string) error {
	return exec.Command("git", "submodule", "add", repo, path).Run()
}

func initSubmodules() error {
	return exec.Command("git", "submodule", "update", "--init", "--recursive").Run()
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

func touch(s ...string) error {
	return ioutil.WriteFile(filepath.Join(s...), []byte{}, 0666)
}
