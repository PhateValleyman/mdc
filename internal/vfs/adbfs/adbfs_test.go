package adbfs

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeADB(t *testing.T, body string) string {
	t.Helper()
	if os.PathSeparator == '\\' {
		t.Skip("fake adb executable uses a POSIX shell")
	}
	binary := filepath.Join(t.TempDir(), "adb")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return binary
}

func TestListDevices(t *testing.T) {
	binary := fakeADB(t, `printf 'List of devices attached\nserial-b device product:foo model:Pixel_8 device:oriole\nserial-a unauthorized usb:1-1\n'`)
	devices, err := listDevices(binary)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 || devices[0].Serial != "serial-a" || devices[1].Model != "Pixel 8" {
		t.Fatalf("devices = %#v", devices)
	}
}

func TestParseURL(t *testing.T) {
	serial, devicePath, ok := ParseURL("adb://192.168.1.4:5555/storage/emulated/0/DCIM")
	if !ok || serial != "192.168.1.4:5555" || devicePath != "/storage/emulated/0/DCIM" {
		t.Fatalf("parsed URL = %q %q %v", serial, devicePath, ok)
	}
	serial, devicePath, ok = ParseURL("adb://emulator-5554")
	if !ok || serial != "emulator-5554" || devicePath != "/sdcard" {
		t.Fatalf("root URL = %q %q %v", serial, devicePath, ok)
	}
	if _, _, ok := ParseURL("adb://bad serial/path"); ok {
		t.Fatal("serial containing spaces must be rejected")
	}
}

func TestShellQuoteAndRootProtection(t *testing.T) {
	if got := shellQuote("/sdcard/O'Brien's file"); got != `'/sdcard/O'\''Brien'\''s file'` {
		t.Fatalf("shellQuote = %q", got)
	}
	fsys, err := newFS("serial", "adb")
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range []struct {
		name string
		call func(string) error
	}{
		{"Remove", fsys.Remove},
		{"RemoveAll", fsys.RemoveAll},
		{"Chmod", func(name string) error { return fsys.Chmod(name, 0o600) }},
	} {
		if err := op.call("/"); err == nil || !strings.Contains(err.Error(), "root") {
			t.Errorf("%s on root = %v, want refusal", op.name, err)
		}
	}
}

func TestStatAndReadDir(t *testing.T) {
	binary := fakeADB(t, `
if [ "$3" = "shell" ]; then
  case "$4" in
	    ls*) printf 'notes.txt\nsecond.txt\nthird.txt\n' ;;
    stat*) printf 'regular file\t12\t1700000000\t644\n' ;;
    *) exit 2 ;;
  esac
else
  exit 2
fi`)
	fsys, err := newFS("serial", binary)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fsys.ReadDir("/sdcard")
	if err != nil || len(entries) != 3 || entries[0].Name() != "notes.txt" || entries[2].Name() != "third.txt" {
		t.Fatalf("ReadDir = %#v, %v", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil || info.Size() != 12 || info.Mode().Perm() != 0o644 {
		t.Fatalf("Info = %#v, %v", info, err)
	}
}

func TestOpenStreamsDeviceFile(t *testing.T) {
	binary := fakeADB(t, `
if [ "$3" = "shell" ]; then
  printf 'regular file\t7\t1700000000\t644\n'
elif [ "$3" = "exec-out" ]; then
  printf 'content'
else
  exit 2
fi`)
	fsys, err := newFS("serial", binary)
	if err != nil {
		t.Fatal(err)
	}
	file, err := fsys.Open("/sdcard/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	content, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || string(content) != "content" {
		t.Fatalf("content=%q readErr=%v closeErr=%v", content, readErr, closeErr)
	}
}

func TestCreateUploadsEditedContent(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "uploaded.txt")
	t.Setenv("ADB_PUSH_DEST", dest)
	binary := fakeADB(t, `if [ "$1" = "-s" ] && [ "$3" = "push" ]; then cp "$4" "$ADB_PUSH_DEST"; else exit 2; fi`)
	fsys, err := newFS("serial", binary)
	if err != nil {
		t.Fatal(err)
	}
	file, err := fsys.Create("/sdcard/edited.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(file, "edited contents"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "edited contents" {
		t.Fatalf("uploaded content=%q err=%v", got, err)
	}
}
