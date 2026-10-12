package adbfs

import (
	"strings"
	"testing"
)

func TestNormalizeConnectTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
		bad   bool
	}{
		{"192.168.1.20", "192.168.1.20:5555", false},
		{"phone.local:5566", "phone.local:5566", false},
		{"2001:db8::1", "[2001:db8::1]:5555", false},
		{"[2001:db8::1]:37000", "[2001:db8::1]:37000", false},
		{"host:0", "", true},
		{"host:65536", "", true},
		{"host;touch /tmp/x", "", true},
		{"host name", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := normalizeConnectTarget(tc.input)
			if tc.bad {
				if err == nil {
					t.Fatalf("normalizeConnectTarget(%q) = %q, want error", tc.input, got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("normalizeConnectTarget(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
		})
	}
}

func TestConnectRecognizesADBSuccess(t *testing.T) {
	binary := fakeADB(t, `if [ "$1" = "connect" ] && [ "$2" = "phone.local:5555" ]; then printf 'connected to %s\n' "$2"; else exit 4; fi`)
	got, err := connect(binary, "phone.local")
	if err != nil || got != "phone.local:5555" {
		t.Fatalf("connect = %q, %v", got, err)
	}
}

func TestConnectRejectsADBFailureOutput(t *testing.T) {
	binary := fakeADB(t, `printf 'unable to connect to %s\n' "$2"`)
	_, err := connect(binary, "phone.local")
	if err == nil || !strings.Contains(err.Error(), "could not connect") {
		t.Fatalf("connect error = %v", err)
	}
}
