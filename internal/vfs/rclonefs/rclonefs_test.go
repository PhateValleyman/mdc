package rclonefs

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fakeBinary(t *testing.T, script string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "rclone")
	content := "#!/bin/sh\nset -eu\n" + script + "\n"
	if err := os.WriteFile(name, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestListRemotesReturnsNamesOnly(t *testing.T) {
	binary := fakeBinary(t, `case "$1" in
listremotes)
  printf 'zeta:\nalpha:\n'
  printf 'api_secret_must_not_escape' >&2
  ;;
*) exit 1 ;;
esac`)
	got, err := listRemotes(binary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "alpha,zeta" {
		t.Fatalf("remote names = %#v, want sorted names only", got)
	}
}

func TestBrowseReadAndWriteRemote(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "calls.log")
	uploadPath := filepath.Join(t.TempDir(), "uploaded")
	t.Setenv("RCLONE_TEST_LOG", logPath)
	t.Setenv("RCLONE_TEST_UPLOAD", uploadPath)
	binary := fakeBinary(t, `case "$1" in
lsjson)
  shift
  if [ "$1" = "--stat" ]; then
    printf '{"Name":"file.txt","Path":"file.txt","Size":5,"IsDir":false,"ModTime":"2024-01-02T03:04:05Z"}'
  else
    printf '[{"Name":"folder","Path":"folder","IsDir":true},{"Name":"file.txt","Path":"file.txt","Size":5,"IsDir":false}]'
  fi
  ;;
cat)
  printf 'hello'
  ;;
copyto)
  printf '%s\\n' "$2|$3" >> "$RCLONE_TEST_LOG"
  cp "$2" "$RCLONE_TEST_UPLOAD"
  ;;
moveto|purge|deletefile|rmdir)
  printf '%s\n' "$1|$2" >> "$RCLONE_TEST_LOG"
  ;;
mkdir)
  printf 'credential_secret_marker' >&2
  exit 7
  ;;
*) exit 1 ;;
esac`)
	fsys, err := newFS("cloud", binary)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := fsys.ReadDir("/")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "folder" || !entries[0].IsDir() {
		t.Fatalf("unexpected directory listing: %#v", entries)
	}

	info, err := fsys.Stat("/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 5 || info.IsDir() {
		t.Fatalf("unexpected stat result: size=%d dir=%v", info.Size(), info.IsDir())
	}

	file, err := fsys.Open("/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("file content = %q, want hello", got)
	}

	out, err := fsys.Create("/upload.mdc-part")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("upload contents")); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	uploaded, err := os.ReadFile(uploadPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(uploaded) != "upload contents" {
		t.Fatalf("uploaded bytes = %q", uploaded)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "cloud:upload.mdc-part") {
		t.Fatalf("upload target was not remote-qualified: %q", calls)
	}

	if err := fsys.Mkdir("/folder", 0o755); err == nil || strings.Contains(err.Error(), "credential_secret_marker") || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("rclone stderr must be discarded and exit status retained; got %v", err)
	}
	if err := fsys.RemoveAll("/"); err == nil {
		t.Fatal("RemoveAll must refuse the remote root")
	}
	if err := fsys.Rename("/file.txt", "/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if err := fsys.RemoveAll("/folder"); err != nil {
		t.Fatal(err)
	}
}

func TestParseURL(t *testing.T) {
	name, remotePath, ok := ParseURL("rclone://Drive/backups/weekly")
	if !ok || name != "Drive" || remotePath != "/backups/weekly" {
		t.Fatalf("parsed rclone URL = %q %q %v", name, remotePath, ok)
	}
	name, remotePath, ok = ParseURL("rclone://Drive")
	if !ok || name != "Drive" || remotePath != "/" {
		t.Fatalf("parsed root URL = %q %q %v", name, remotePath, ok)
	}
	if _, _, ok := ParseURL("rclone://bad:name/path"); ok {
		t.Fatal("remote names containing a colon must be rejected")
	}
}

func TestRootMutationRejected(t *testing.T) {
	binary := fakeBinary(t, `exit 0`)
	fsys, err := newFS("remote:", binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []struct {
		name string
		call func() error
	}{
		{"remove", func() error { return fsys.Remove("/") }},
		{"remove all", func() error { return fsys.RemoveAll("") }},
		{"rename root", func() error { return fsys.Rename("/", "/elsewhere") }},
		{"create root", func() error { _, err := fsys.Create("/"); return err }},
	} {
		t.Run(op.name, func(t *testing.T) {
			if err := op.call(); err == nil {
				t.Errorf("%s must refuse the remote root", op.name)
			}
		})
	}
}

var _ fs.FileInfo = fileInfo{}
var _ fs.DirEntry = dirEntry{}

func TestLocalRcloneRemoteIntegration(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed")
	}
	root := t.TempDir()
	remoteRoot := filepath.Join(root, "remote-data")
	if err := os.MkdirAll(remoteRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "rclone.conf")
	config := "[integration]\ntype = local\nroot = \"" + remoteRoot + "\"\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RCLONE_CONFIG", configPath)

	remotes, err := ListRemotes()
	if err != nil || len(remotes) != 1 || remotes[0] != "integration" {
		t.Fatalf("rclone remotes = %#v, err = %v", remotes, err)
	}
	fsys, err := New("integration")
	if err != nil {
		t.Fatal(err)
	}
	if err := fsys.MkdirAll("/docs", 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := fsys.Create("/docs/readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("rclone integration")); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := fsys.ReadDir("/docs")
	if err != nil || len(entries) != 1 || entries[0].Name() != "readme.txt" {
		t.Fatalf("rclone listing = %#v, err = %v", entries, err)
	}
	file, err := fsys.Open("/docs/readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if string(contents) != "rclone integration" {
		t.Fatalf("remote file contents = %q", contents)
	}
	if err := fsys.Rename("/docs/readme.txt", "/docs/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if err := fsys.Remove("/docs/renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if err := fsys.RemoveAll("/docs"); err != nil {
		t.Fatal(err)
	}
}
