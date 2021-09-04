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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const rfc2822 = "Mon, _2 Jan 2006 15:04:05 -0700"

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr

	diffMerges = "--no-merges"
	recurse    = flag.Bool("r", false, "Recurse into submodules.")
	errParse   = errors.New("parse error")
)

func init() {
	flag.Var(newMergeValue(&diffMerges, "-c"), "c", `Specify the -c option to git log.`)
	flag.Var(newMergeValue(&diffMerges, "-m"), "m", `Specify the -m option to git log.`)
}

func main() {
	flag.Parse()

	wt, err := getwt()
	if err != nil {
		abort(err)
	}
	if err := utimeAll(wt); err != nil {
		abort(err)
	}
}

type mergeValue struct {
	s       *string
	on, off string
}

func newMergeValue(p *string, v string) *mergeValue {
	return &mergeValue{
		s:   p,
		on:  v,
		off: *p,
	}
}

func (m *mergeValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	switch {
	case err != nil:
		err = errParse
	case v:
		*m.s = m.on
	case *m.s == m.on:
		*m.s = m.off
	}
	return err
}

func (m *mergeValue) Get() interface{} { return m.s != nil && *m.s == m.on }
func (m *mergeValue) String() string   { return strconv.FormatBool(m.s != nil && *m.s == m.on) }
func (m *mergeValue) IsBoolFlag() bool { return true }

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

func utimeAll(wt string) error {
	order := []string{wt}
	if *recurse {
		mods, err := submodules(wt)
		if err != nil {
			return err
		}
		order = append(order, mods...)
	}
	dirs := make(map[string]fileset, len(order))
	var m, n int
	for _, p := range order {
		files, err := ls(p)
		if err != nil {
			return err
		}
		dirs[p] = files
		n += len(files)
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		p := order[i]
		m += len(dirs[p])
		if err := utime(p, dirs[p], m, n); err != nil {
			return err
		}
	}
	return nil
}

func submodules(path string) (mods []string, err error) {
	err = git([]string{"-C", path, "submodule", "status", "--recursive"}, func(out *bufio.Reader) error {
		for {
			s, err := out.ReadString('\n')
			switch {
			case err != nil:
				return err
			case s[0] == '-':
				continue
			}
			mods = append(mods, filepath.Join(path, strings.SplitN(s[1:], " ", 3)[1]))
		}
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

func utime(wt string, files fileset, m, n int) error {
	if len(files) == 0 {
		return nil
	}

	dirs := make(map[string]time.Time)
	err := git([]string{"-C", wt, "log", "--pretty=%n%x00%cD", diffMerges, "-z", "--name-only", "--no-color", "--no-renames"}, func(out *bufio.Reader) error {
		var done bool
		defer func() {
			if m == n && done {
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
				switch l[i:] {
				case "\x00":
					// commit: file names are in the next line
					continue
				case "\x00\x00":
					// merge commit: no file names
					continue
				default:
					// merge commit: file names are in the same line
					l = l[i+2:]
				}
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
				fmt.Fprintf(stdout, "\rutime: %3d%% (%d/%d)", (m-len(files))*100/n, m-len(files), n)
				done = true
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
