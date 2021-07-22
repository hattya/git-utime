//
// git-utime :: utime.go
//
//   Copyright (c) 2021 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const rfc2822 = "Mon, _2 Jan 2006 15:04:05 -0700"

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func main() {
	wt, err := getwt()
	if err != nil {
		abort(err)
	}
	files, err := ls(wt)
	if err != nil {
		abort(err)
	}
	if err := utime(wt, files); err != nil {
		abort(err)
	}
}

func abort(err error) {
	if err, ok := err.(*exec.ExitError); ok {
		os.Exit(err.ExitCode())
	}
	fmt.Fprintln(stderr, "error:", err)
	os.Exit(1)
}

type fileset map[string]struct{}

func getwt() (wt string, err error) {
	err = git([]string{"rev-parse", "--show-toplevel"}, func(out *bufio.Reader) error {
		wt, err = out.ReadString('\n')
		if err == nil {
			wt = filepath.FromSlash(strings.TrimRight(wt, "\r\n"))
		}
		return err
	})
	return
}

func ls(path string) (fileset, error) {
	files := make(fileset)
	err := git([]string{"-C", path, "ls-files", "-z"}, func(out *bufio.Reader) error {
		for {
			p, err := out.ReadString('\x00')
			if err != nil {
				return err
			}
			files[p[:len(p)-1]] = struct{}{}
		}
	})
	if err != nil {
		return nil, err
	}
	// filter modified
	err = git([]string{"-C", path, "status", "-z", "--porcelain"}, func(out *bufio.Reader) error {
		for {
			s, err := out.ReadString('\x00')
			if err != nil {
				return err
			}
			delete(files, s[3:len(s)-1])
			// renamed or copied
			switch s[0] {
			case 'R', 'C':
				p, err := out.ReadString('\x00')
				if err != nil {
					return err
				}
				delete(files, p[:len(p)-1])
			}
		}
	})
	return files, err
}

func utime(wt string, files fileset) error {
	if len(files) == 0 {
		return nil
	}

	dirs := make(map[string]time.Time)
	err := git([]string{"-C", wt, "log", "--pretty=%n%x00%cD", "-z", "--name-only", "--no-color", "--no-renames"}, func(out *bufio.Reader) error {
		n := len(files)
		defer func() {
			if len(files) < n {
				fmt.Fprintln(stdout)
			}
		}()

		var eof bool
		var tm time.Time
		for !eof && len(files) > 0 {
			l, err := out.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					return err
				}
				eof = true
			}
			l = strings.TrimRight(l, "\r\n")

			switch {
			case l == "":
				continue
			case l[0] == '\x00':
				l = l[1:]
				i := strings.IndexByte(l, '\x00')
				tm, err = time.Parse(rfc2822, l[:i])
				if err != nil {
					return err
				}
				// commit: file names are in the next line
				continue
			}
			for _, p := range strings.Split(l, "\x00") {
				if _, ok := files[p]; !ok {
					continue
				}
				delete(files, p)

				p = filepath.Join(wt, p)
				if err := lutimes(p, tm, tm); err != nil {
					return err
				}
				for p != wt {
					p = filepath.Dir(p)
					if _, ok := dirs[p]; !ok {
						dirs[p] = tm
					}
				}
				fmt.Fprintf(stdout, "\rutime: %3d%% (%d/%d)", (n-len(files))*100/n, n-len(files), n)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	list := make(sort.StringSlice, len(dirs))
	i := 0
	for k := range dirs {
		list[i] = k
		i++
	}
	sort.Sort(sort.Reverse(list))
	for _, p := range list {
		tm := dirs[p]
		if err := lutimes(p, tm, tm); err != nil {
			return err
		}
	}
	return nil
}

func git(args []string, fn func(*bufio.Reader) error) error {
	cmd := exec.Command("git", append([]string{"-c", "core.quotepath=false"}, args...)...)
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return err
	}
	if err := fn(bufio.NewReader(stdout)); err != nil && err != io.EOF {
		cmd.Process.Kill()
		cmd.Wait()
		return err
	}
	stdout.Close()
	return cmd.Wait()
}
