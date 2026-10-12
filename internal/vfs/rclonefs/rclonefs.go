// Package rclonefs adapts the rclone command line to mdc's VFS interfaces.
// It never captures or returns rclone's stderr, which can contain provider
// diagnostics and must not become application output or logs.
package rclonefs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kooler/MiddayCommander/internal/vfs"
)

const defaultBinary = "rclone"

// FS is a writable VFS view of one configured rclone remote.
type FS struct {
	remote string
	binary string
}

// New creates a VFS for a configured remote name (with or without a trailing colon).
func New(remoteName string) (*FS, error) {
	return newFS(remoteName, defaultBinary)
}

func newFS(remoteName, binary string) (*FS, error) {
	remoteName = strings.TrimSuffix(remoteName, ":")
	if !validRemoteName(remoteName) {
		return nil, fmt.Errorf("invalid rclone remote name")
	}
	if binary == "" {
		binary = defaultBinary
	}
	return &FS{remote: remoteName, binary: binary}, nil
}

func validRemoteName(name string) bool {
	return strings.TrimSpace(name) != "" && !strings.ContainsAny(name, ":/\\\x00\r\n")
}

// ListRemotes returns names only; it deliberately never reads or emits config contents.
func ListRemotes() ([]string, error) {
	return listRemotes(defaultBinary)
}

func listRemotes(binary string) ([]string, error) {
	cmd := exec.Command(binary, "listremotes")
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError("listremotes", err)
	}
	var remotes []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasSuffix(line, ":") {
			line = strings.TrimSuffix(line, ":")
		}
		if validRemoteName(line) {
			remotes = append(remotes, line)
		}
	}
	sort.Strings(remotes)
	return remotes, nil
}

// ParseURL recognizes mdc's address form rclone://remote/path.
func ParseURL(raw string) (remoteName, remotePath string, ok bool) {
	const prefix = "rclone://"
	if !strings.HasPrefix(raw, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(raw, prefix)
	name, tail, hasPath := strings.Cut(rest, "/")
	if !validRemoteName(name) {
		return "", "", false
	}
	if !hasPath || tail == "" {
		return name, "/", true
	}
	return name, path.Clean("/" + tail), true
}

func (f *FS) spec(name string) string {
	clean := cleanPath(name)
	if clean == "/" {
		return f.remote + ":"
	}
	return f.remote + ":" + strings.TrimPrefix(clean, "/")
}

func cleanPath(name string) string {
	if name == "" || name == "." || name == "/" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(name, "/"))
}

func (f *FS) mutableSpec(name string) (string, error) {
	clean := cleanPath(name)
	if clean == "/" {
		return "", fmt.Errorf("refusing to modify the rclone remote root")
	}
	return f.spec(clean), nil
}

func (f *FS) run(operation string, args ...string) ([]byte, error) {
	allArgs := append([]string{operation}, args...)
	cmd := exec.Command(f.binary, allArgs...)
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError(operation, err)
	}
	return output, nil
}

func commandError(operation string, err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("rclone executable not found")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("rclone %s failed (exit status %d)", operation, exitErr.ExitCode())
	}
	return fmt.Errorf("rclone %s failed", operation)
}

type item struct {
	Name    string `json:"Name"`
	Path    string `json:"Path"`
	Size    int64  `json:"Size"`
	IsDir   bool   `json:"IsDir"`
	ModTime string `json:"ModTime"`
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	output, err := f.run("lsjson", "--no-mimetype", f.spec(name))
	if err != nil {
		return nil, err
	}
	var items []item
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("invalid JSON returned by rclone lsjson")
	}
	entries := make([]fs.DirEntry, 0, len(items))
	for _, it := range items {
		info := makeInfo(it, f.remote)
		if info.Name() == "" || info.Name() == "." || info.Name() == ".." {
			continue
		}
		entries = append(entries, dirEntry{info: info})
	}
	return entries, nil
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	output, err := f.run("lsjson", "--stat", "--no-mimetype", f.spec(name))
	if err != nil {
		return nil, err
	}
	var it item
	if err := json.Unmarshal(output, &it); err != nil {
		return nil, fmt.Errorf("invalid JSON returned by rclone lsjson")
	}
	return makeInfo(it, f.remote), nil
}

func makeInfo(it item, remoteName string) fileInfo {
	name := it.Name
	if name == "" {
		name = path.Base(it.Path)
	}
	if name == "." || name == "/" || name == "" {
		name = remoteName
	}
	mode := fs.FileMode(0o644)
	if it.IsDir {
		mode = fs.ModeDir | 0o755
	}
	modTime, _ := time.Parse(time.RFC3339Nano, it.ModTime)
	return fileInfo{name: name, size: it.Size, mode: mode, modTime: modTime}
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (i fileInfo) Name() string       { return i.name }
func (i fileInfo) Size() int64        { return i.size }
func (i fileInfo) Mode() fs.FileMode  { return i.mode }
func (i fileInfo) ModTime() time.Time { return i.modTime }
func (i fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fileInfo) Sys() any           { return nil }

type dirEntry struct{ info fileInfo }

func (e dirEntry) Name() string               { return e.info.Name() }
func (e dirEntry) IsDir() bool                { return e.info.IsDir() }
func (e dirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e dirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

func (f *FS) Open(name string) (fs.File, error) {
	info, err := f.Stat(name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot open an rclone directory as a file")
	}
	cmd := exec.Command(f.binary, "cat", f.spec(name))
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, commandError("cat", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, commandError("cat", err)
	}
	return &readFile{reader: stdout, cmd: cmd, info: info}, nil
}

type readFile struct {
	reader  io.ReadCloser
	cmd     *exec.Cmd
	info    fs.FileInfo
	waited  bool
	waitErr error
	closed  bool
}

func (f *readFile) Read(p []byte) (int, error) {
	n, err := f.reader.Read(p)
	if err == io.EOF {
		if waitErr := f.wait(); waitErr != nil {
			return n, waitErr
		}
	}
	return n, err
}

func (f *readFile) Close() error {
	if f.closed {
		return f.waitErr
	}
	f.closed = true
	_ = f.reader.Close()
	if !f.waited && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
	}
	return f.wait()
}

func (f *readFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *readFile) wait() error {
	if !f.waited {
		f.waitErr = f.cmd.Wait()
		f.waited = true
		if f.waitErr != nil {
			f.waitErr = commandError("cat", f.waitErr)
		}
	}
	return f.waitErr
}

func (f *FS) Create(name string) (vfs.WriteFile, error) {
	target, err := f.mutableSpec(name)
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "mdc-rclone-*")
	if err != nil {
		return nil, fmt.Errorf("create local staging file for rclone upload")
	}
	return &writeFile{fs: f, target: target, tmp: tmp}, nil
}

type writeFile struct {
	fs       *FS
	target   string
	tmp      *os.File
	closed   bool
	closeErr error
}

func (f *writeFile) Write(p []byte) (int, error) { return f.tmp.Write(p) }
func (f *writeFile) Read(p []byte) (int, error)  { return f.tmp.Read(p) }
func (f *writeFile) Stat() (fs.FileInfo, error)  { return f.tmp.Stat() }
func (f *writeFile) Name() string                { return path.Base(strings.TrimPrefix(f.target, f.fs.remote+":")) }

func (f *writeFile) Close() error {
	if f.closed {
		return f.closeErr
	}
	f.closed = true
	if err := f.tmp.Close(); err != nil {
		f.closeErr = fmt.Errorf("close local rclone staging file")
		_ = os.Remove(f.tmp.Name())
		return f.closeErr
	}
	_, f.closeErr = f.fs.run("copyto", f.tmp.Name(), f.target)
	_ = os.Remove(f.tmp.Name())
	return f.closeErr
}

func (f *FS) Mkdir(name string, _ fs.FileMode) error {
	target, err := f.mutableSpec(name)
	if err != nil {
		return err
	}
	_, err = f.run("mkdir", target)
	return err
}

func (f *FS) MkdirAll(name string, perm fs.FileMode) error { return f.Mkdir(name, perm) }

func (f *FS) Remove(name string) error {
	target, err := f.mutableSpec(name)
	if err != nil {
		return err
	}
	info, err := f.Stat(name)
	if err != nil {
		return err
	}
	operation := "deletefile"
	if info.IsDir() {
		operation = "rmdir"
	}
	_, err = f.run(operation, target)
	return err
}

func (f *FS) RemoveAll(name string) error {
	target, err := f.mutableSpec(name)
	if err != nil {
		return err
	}
	_, err = f.run("purge", target)
	return err
}

func (f *FS) Rename(oldname, newname string) error {
	src, err := f.mutableSpec(oldname)
	if err != nil {
		return err
	}
	dst, err := f.mutableSpec(newname)
	if err != nil {
		return err
	}
	_, err = f.run("moveto", src, dst)
	return err
}

var _ vfs.WritableFS = (*FS)(nil)
