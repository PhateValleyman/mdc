package rclonefs

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Provider is the public, non-secret part of an rclone backend definition.
type Provider struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Hide        bool   `json:"Hide"`
}

// ConfigExample is one suggested value for an interactive backend option.
type ConfigExample struct {
	Value any    `json:"Value"`
	Help  string `json:"Help"`
}

// ConfigOption describes a question emitted by rclone's non-interactive
// configuration protocol. Value may contain an existing secret; callers must
// keep password options masked and never log the value.
type ConfigOption struct {
	Name       string          `json:"Name"`
	Help       string          `json:"Help"`
	Default    any             `json:"Default"`
	Value      any             `json:"Value"`
	DefaultStr string          `json:"DefaultStr"`
	ValueStr   string          `json:"ValueStr"`
	Examples   []ConfigExample `json:"Examples"`
	Required   bool            `json:"Required"`
	IsPassword bool            `json:"IsPassword"`
	Sensitive  bool            `json:"Sensitive"`
	Exclusive  bool            `json:"Exclusive"`
	Type       string          `json:"Type"`
}

// ConfigQuestion is one step from rclone's --non-interactive config wizard.
type ConfigQuestion struct {
	State  string        `json:"State"`
	Option *ConfigOption `json:"Option"`
	Error  string        `json:"Error"`
	Result string        `json:"Result"`
}

// ListProviders returns rclone's provider names and descriptions, not config values.
func ListProviders() ([]Provider, error) { return listProviders(defaultBinary) }

func listProviders(binary string) ([]Provider, error) {
	output, err := runConfig(binary, "providers")
	if err != nil {
		return nil, err
	}
	var all []Provider
	if err := json.Unmarshal(output, &all); err != nil {
		return nil, fmt.Errorf("invalid provider list returned by rclone")
	}
	providers := make([]Provider, 0, len(all))
	for _, provider := range all {
		if provider.Name != "" && !provider.Hide {
			providers = append(providers, provider)
		}
	}
	return providers, nil
}

// StartConfigCreate starts rclone's in-process, non-interactive config flow.
func StartConfigCreate(name, provider string) (ConfigQuestion, error) {
	return startConfigCreate(defaultBinary, name, provider)
}

func startConfigCreate(binary, name, provider string) (ConfigQuestion, error) {
	if !validRemoteName(name) || strings.TrimSpace(provider) == "" || strings.ContainsAny(provider, " \t\r\n") {
		return ConfigQuestion{}, fmt.Errorf("invalid remote name or provider")
	}
	remotes, err := listRemotes(binary)
	if err != nil {
		return ConfigQuestion{}, err
	}
	for _, remote := range remotes {
		if strings.EqualFold(remote, name) {
			return ConfigQuestion{}, fmt.Errorf("an rclone remote with that name already exists")
		}
	}
	return runQuestion(binary, "create", name, provider, "--all", "--non-interactive")
}

// StartConfigUpdate starts the guided edit flow for an existing remote.
func StartConfigUpdate(name string) (ConfigQuestion, error) {
	return startConfigUpdate(defaultBinary, name)
}

func startConfigUpdate(binary, name string) (ConfigQuestion, error) {
	if !validRemoteName(name) {
		return ConfigQuestion{}, fmt.Errorf("invalid rclone remote name")
	}
	return runQuestion(binary, "update", name, "--all", "--non-interactive")
}

// DeleteConfig removes a remote created by a config flow that the user cancelled.
func DeleteConfig(name string) error { return deleteConfig(defaultBinary, name) }

func deleteConfig(binary, name string) error {
	if !validRemoteName(name) {
		return fmt.Errorf("invalid rclone remote name")
	}
	_, err := runConfig(binary, "delete", name)
	return err
}

func runQuestion(binary string, args ...string) (ConfigQuestion, error) {
	allArgs := append([]string{"config"}, args...)
	return runQuestionArgs(binary, allArgs...)
}

func runQuestionArgs(binary string, args ...string) (ConfigQuestion, error) {
	output, err := run(binary, args...)
	if err != nil {
		return ConfigQuestion{}, err
	}
	var question ConfigQuestion
	if err := json.Unmarshal(output, &question); err != nil {
		return ConfigQuestion{}, fmt.Errorf("invalid configuration prompt returned by rclone")
	}
	return question, nil
}

func runConfig(binary string, args ...string) ([]byte, error) {
	allArgs := append([]string{"config"}, args...)
	return run(binary, allArgs...)
}

func run(binary string, args ...string) ([]byte, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		// Do not expose raw rclone diagnostics, which may contain backend details.
		return nil, commandError("config", err)
	}
	return output, nil
}
