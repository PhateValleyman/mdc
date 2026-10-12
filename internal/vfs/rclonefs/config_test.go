package rclonefs

import (
	"os"
	"os/exec"
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

func TestConfigCommandErrorsNeverIncludeAnswers(t *testing.T) {
	binary := fakeConfigBinary(t, "exit 17")
	const secret = "credential-with-sensitive-content"
	_, err := continueConfig(binary, "safe", "local", true, "state", secret)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("error = %v; it must not expose the answer", err)
	}
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

func TestConfigCreateUpdateFlowWithRclone(t *testing.T) {
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed")
	}
	providers, err := ListProviders()
	if err != nil || len(providers) == 0 {
		t.Fatalf("provider list has %d entries, err=%v", len(providers), err)
	}
	configPath := filepath.Join(t.TempDir(), "rclone.conf")
	t.Setenv("RCLONE_CONFIG", configPath)

	q, err := StartConfigCreate("mdc-test", "local")
	if err != nil || q.State == "" || q.Option == nil {
		t.Fatalf("start create: question=%+v err=%v", q, err)
	}
	q, err = ContinueConfig("mdc-test", "local", true, q.State, "false")
	if err != nil || q.State != "" {
		t.Fatalf("finish create: question=%+v err=%v", q, err)
	}
	remotes, err := listRemotes("rclone")
	if err != nil || len(remotes) != 1 || remotes[0] != "mdc-test" {
		t.Fatalf("remotes = %#v, err=%v", remotes, err)
	}

	q, err = StartConfigUpdate("mdc-test")
	if err != nil || q.State == "" || q.Option == nil {
		t.Fatalf("start update: question=%+v err=%v", q, err)
	}
	q, err = ContinueConfig("mdc-test", "", false, q.State, "false")
	if err != nil || q.State != "" {
		t.Fatalf("finish update: question=%+v err=%v", q, err)
	}
	if err := deleteConfig("rclone", "mdc-test"); err != nil {
		t.Fatalf("delete cancelled remote: %v", err)
	}
	remotes, err = listRemotes("rclone")
	if err != nil || len(remotes) != 0 {
		t.Fatalf("remotes after cleanup = %#v, err=%v", remotes, err)
	}
}
