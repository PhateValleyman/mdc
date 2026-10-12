package adbfs

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// Connect initiates an adb TCP/IP connection. A missing port defaults to 5555.
// Android Wireless Debugging pairing must be completed separately before calling
// this for devices that require pairing.
func Connect(address string) (string, error) { return connect(defaultBinary, address) }

func connect(binary, address string) (string, error) {
	target, err := normalizeConnectTarget(address)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(binary, "connect", target)
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return "", commandError("connect", err)
	}
	response := strings.ToLower(strings.TrimSpace(string(output)))
	if strings.HasPrefix(response, "connected to ") || strings.HasPrefix(response, "already connected to ") {
		return target, nil
	}
	return "", fmt.Errorf("adb could not connect to the target")
}

func normalizeConnectTarget(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" || strings.ContainsAny(address, "\x00\r\n\t ") {
		return "", fmt.Errorf("enter a host name or IP address, optionally followed by :port")
	}
	if strings.HasPrefix(address, "[") && strings.HasSuffix(address, "]") {
		host := strings.TrimSuffix(strings.TrimPrefix(address, "["), "]")
		if net.ParseIP(host) == nil {
			return "", fmt.Errorf("invalid IPv6 address")
		}
		return net.JoinHostPort(host, "5555"), nil
	}
	if ip := net.ParseIP(address); ip != nil {
		return net.JoinHostPort(address, "5555"), nil
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		if strings.Count(address, ":") != 0 {
			return "", fmt.Errorf("use host:port, or bracket IPv6 addresses")
		}
		host, portText = address, "5555"
	}
	if !validConnectHost(host) {
		return "", fmt.Errorf("invalid adb host name or IP address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("adb port must be between 1 and 65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func validConnectHost(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}
	return true
}
