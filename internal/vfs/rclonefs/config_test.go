package rclonefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeConfigBinary(t *testing.T, body string) string {
	t.Helper()
	if os.PathSeparator == '\\' {
		t.Skip("fake executable uses a POSIX shell")
	}
	binary := filepath.Join(t.TempDir(), "rclone")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return binary
}

func TestCreateRefusesExistingRemoteBeforeCallingRcloneCreate(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "create-called")
	t.Setenv("RCLONE_CREATE_MARKER", marker)
	binary := fakeConfigBinary(t, `
if [ "$1" = "listremotes" ]; then
	  printf '%s\n' 'existing:'
elif [ "$1" = "config" ]; then
  : > "$RCLONE_CREATE_MARKER"
  printf '{"State":"unexpected"}'
else
  exit 3
fi`)
	_, err := startConfigCreate(binary, "Existing", "local")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create error = %v, want duplicate refusal", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("rclone config create ran for duplicate remote: stat err=%v", err)
	}
}
