// Package adbfs adapts the Android Debug Bridge to mdc's VFS interfaces.
package adbfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kooler/MiddayCommander/internal/vfs"
)

const defaultBinary = "adb"

// Device describes one row reported by `adb devices -l`.
type Device struct {
	Serial  string
	State   string
	Model   string
	Product string
}

// FS is a writable virtual filesystem for one connected Android device.
type FS struct {
	serial string
	binary string
}

// New creates a filesystem bound to an adb device serial.
func New(serial string) (*FS, error) { return newFS(serial, defaultBinary) }

func newFS(serial, binary string) (*FS, error) {
	if !validSerial(serial) {
		return nil, fmt.Errorf("invalid adb device serial")
	}
	if binary == "" {
		binary = defaultBinary
	}
	return &FS{serial: serial, binary: binary}, nil
}

func validSerial(serial string) bool {
	if serial == "" || strings.TrimSpace(serial) != serial {
		return false
	}
	for _, r := range serial {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:-", r)) {
			return false
		}
	}
	return true
}

// ListDevices returns connected and unavailable adb targets without exposing
// adb's diagnostics, which can include implementation-specific information.
func ListDevices() ([]Device, error) { return listDevices(defaultBinary) }

func listDevices(binary string) ([]Device, error) {
	cmd := exec.Command(binary, "devices", "-l")
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError("devices", err)
	}
	var devices []Device
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 || fields[0] == "List" || !validSerial(fields[0]) {
			continue
		}
		device := Device{Serial: fields[0], State: fields[1]}
		for _, field := range fields[2:] {
			key, value, ok := strings.Cut(field, ":")
			if !ok {
				continue
			}
			switch key {
			case "model":
				device.Model = strings.ReplaceAll(value, "_", " ")
			case "product":
				device.Product = value
			}
		}
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Serial < devices[j].Serial })
	return devices, nil
}

// ParseURL recognizes mdc's address form adb://serial/path.
func ParseURL(raw string) (serial, devicePath string, ok bool) {
	const prefix = "adb://"
	if !strings.HasPrefix(raw, prefix) {
		return "", "", false
	}
	serial, tail, hasPath := strings.Cut(strings.TrimPrefix(raw, prefix), "/")
	if !validSerial(serial) {
		return "", "", false
	}
	if !hasPath || tail == "" {
		return serial, "/sdcard", true
	}
	return serial, cleanPath("/" + tail), true
}

func cleanPath(name string) string {
	if name == "" || name == "." || name == "/" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(name, "/"))
}

func (f *FS) Label() string { return "adb://" + f.serial }

func (f *FS) shell(command string) ([]byte, error) {
	cmd := exec.Command(f.binary, "-s", f.serial, "shell", command)
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return nil, commandError("shell", err)
	}
	return output, nil
}

func commandError(operation string, err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("adb executable not found")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("adb %s failed (exit status %d)", operation, exitErr.ExitCode())
	}
	return fmt.Errorf("adb %s failed", operation)
}

// shellQuote quotes a path for Android's /system/bin/sh. Paths are always
// passed as one shell command, so quoting is required even when adb shell v2 is
// available: this also works with older adb clients.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	p := cleanPath(name)
	format := shellQuote("%F\\t%s\\t%Y\\t%a")
	output, err := f.shell("stat -L -c " + format + " " + shellQuote(p))
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(fields) != 4 {
		return nil, fmt.Errorf("invalid stat response from adb device")
	}
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stat response from adb device")
	}
	unix, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stat response from adb device")
	}
	perm, err := strconv.ParseUint(fields[3], 8, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid stat response from adb device")
	}
	mode := fs.FileMode(perm) & fs.ModePerm
	switch strings.ToLower(fields[0]) {
	case "directory":
		mode |= fs.ModeDir
	case "symbolic link":
		mode |= fs.ModeSymlink
	case "character special file":
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case "block special file":
		mode |= fs.ModeDevice
	case "fifo":
		mode |= fs.ModeNamedPipe
	case "socket":
		mode |= fs.ModeSocket
	}
	return fileInfo{name: path.Base(p), size: size, mode: mode, modTime: time.Unix(unix, 0)}, nil
}

func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	p := cleanPath(name)
	output, err := f.shell("ls -1A " + shellQuote(p))
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for _, name := range strings.Split(strings.TrimSuffix(string(output), "\n"), "\n") {
		name = strings.TrimSuffix(name, "\r")
		if name == "" || name == "." || name == ".." {
			continue
		}
		child := path.Join(p, name)
		info, err := f.Stat(child)
		if err != nil {
			return nil, err
		}
		entries = append(entries, dirEntry{info: info})
	}
	return entries, nil
}

func (f *FS) Open(name string) (fs.File, error) {
	p := cleanPath(name)
	info, err := f.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot open an adb directory as a file")
	}
	cmd := exec.Command(f.binary, "-s", f.serial, "exec-out", "cat "+shellQuote(p))
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, commandError("read", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, commandError("read", err)
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
			f.waitErr = commandError("read", f.waitErr)
		}
	}
	return f.waitErr
}

func (f *FS) Create(name string) (vfs.WriteFile, error) {
	p, err := f.mutablePath(name)
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "mdc-adb-*")
	if err != nil {
		return nil, fmt.Errorf("create local adb staging file")
	}
	return &writeFile{fs: f, target: p, tmp: tmp}, nil
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
func (f *writeFile) Name() string                { return path.Base(f.target) }
func (f *writeFile) Close() error {
	if f.closed {
		return f.closeErr
	}
	f.closed = true
	if err := f.tmp.Close(); err != nil {
		f.closeErr = fmt.Errorf("close local adb staging file")
		_ = os.Remove(f.tmp.Name())
		return f.closeErr
	}
	cmd := exec.Command(f.fs.binary, "-s", f.fs.serial, "push", f.tmp.Name(), f.target)
	cmd.Stderr = io.Discard
	if _, err := cmd.Output(); err != nil {
		f.closeErr = commandError("push", err)
	}
	_ = os.Remove(f.tmp.Name())
	return f.closeErr
}

func (f *FS) mutablePath(name string) (string, error) {
	p := cleanPath(name)
	if p == "/" {
		return "", fmt.Errorf("refusing to modify the adb device root")
	}
	return p, nil
}
func (f *FS) Mkdir(name string, perm fs.FileMode) error {
	p, err := f.mutablePath(name)
	if err != nil {
		return err
	}
	_, err = f.shell("mkdir " + shellQuote(p) + " && chmod " + strconv.FormatUint(uint64(perm.Perm()), 8) + " " + shellQuote(p))
	return err
}
func (f *FS) MkdirAll(name string, perm fs.FileMode) error {
	p, err := f.mutablePath(name)
	if err != nil {
		return err
	}
	_, err = f.shell("mkdir -p " + shellQuote(p) + " && chmod " + strconv.FormatUint(uint64(perm.Perm()), 8) + " " + shellQuote(p))
	return err
}
func (f *FS) Remove(name string) error {
	p, err := f.mutablePath(name)
	if err != nil {
		return err
	}
	_, err = f.shell("rm -f " + shellQuote(p))
	return err
}
func (f *FS) RemoveAll(name string) error {
	p, err := f.mutablePath(name)
	if err != nil {
		return err
	}
	_, err = f.shell("rm -rf " + shellQuote(p))
	return err
}
func (f *FS) Rename(oldname, newname string) error {
	oldPath, err := f.mutablePath(oldname)
	if err != nil {
		return err
	}
	newPath, err := f.mutablePath(newname)
	if err != nil {
		return err
	}
	_, err = f.shell("mv " + shellQuote(oldPath) + " " + shellQuote(newPath))
	return err
}
func (f *FS) Chmod(name string, mode fs.FileMode) error {
	p, err := f.mutablePath(name)
	if err != nil {
		return err
	}
	_, err = f.shell("chmod " + strconv.FormatUint(uint64(mode.Perm()), 8) + " " + shellQuote(p))
	return err
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

type dirEntry struct{ info fs.FileInfo }

func (e dirEntry) Name() string               { return e.info.Name() }
func (e dirEntry) IsDir() bool                { return e.info.IsDir() }
func (e dirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e dirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

var _ vfs.WritableFS = (*FS)(nil)
var _ vfs.Chmoder = (*FS)(nil)
