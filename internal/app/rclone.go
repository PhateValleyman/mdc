package app

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	adbui "github.com/kooler/MiddayCommander/internal/ui/adb"
	rcloneui "github.com/kooler/MiddayCommander/internal/ui/rclone"
	"github.com/kooler/MiddayCommander/internal/vfs"
	"github.com/kooler/MiddayCommander/internal/vfs/adbfs"
	"github.com/kooler/MiddayCommander/internal/vfs/rclonefs"
)

func (m Model) startRcloneRemotes() (tea.Model, tea.Cmd) {
	picker := rcloneui.New(m.width, m.height)
	m.rclonePicker = &picker
	return m, rcloneui.LoadRemotesCmd()
}

func rcloneConfigCommand() *exec.Cmd {
	return exec.Command("rclone", "config")
}

func (m Model) startRcloneConfig() (tea.Model, tea.Cmd) {
	cmd := rcloneConfigCommand()
	return m, tea.ExecProcess(cmd, func(_ error) tea.Msg {
		return rcloneui.ConfigCommandDoneMsg{}
	})
}

func (m Model) openRcloneTarget(raw string) (tea.Model, tea.Cmd, bool) {
	remoteName, remotePath, ok := rclonefs.ParseURL(raw)
	if !ok {
		return m, nil, false
	}
	next, cmd := m.openRcloneRemote(remoteName, remotePath)
	return next, cmd, true
}

func (m Model) openRcloneRemote(remoteName, remotePath string) (tea.Model, tea.Cmd) {
	fsys := m.rcloneFileSystems[remoteName]
	if fsys == nil {
		var err error
		fsys, err = rclonefs.New(remoteName)
		if err != nil {
			return m.showError("Invalid rclone remote", err)
		}
		if m.rcloneFileSystems == nil {
			m.rcloneFileSystems = make(map[string]*rclonefs.FS)
		}
		m.rcloneFileSystems[remoteName] = fsys
	}
	if remotePath == "" {
		remotePath = "/"
	}
	m.activePanel().SetLocation(vfs.NewLocation(fsys, remotePath, vfs.KindRclone, "rclone://"+remoteName))
	m.releaseUnusedConnections()
	return m, m.activePanel().LoadDir()
}

func (m Model) startADBDevices() (tea.Model, tea.Cmd) {
	picker := adbui.New(m.width, m.height)
	m.adbPicker = &picker
	return m, adbui.LoadDevicesCmd()
}

func (m Model) openADBTarget(raw string) (tea.Model, tea.Cmd, bool) {
	serial, devicePath, ok := adbfs.ParseURL(raw)
	if !ok {
		return m, nil, false
	}
	next, cmd := m.openADBDevice(serial, devicePath)
	return next, cmd, true
}

func (m Model) openADBDevice(serial, devicePath string) (tea.Model, tea.Cmd) {
	fsys := m.adbFileSystems[serial]
	if fsys == nil {
		var err error
		fsys, err = adbfs.New(serial)
		if err != nil {
			return m.showError("Invalid adb device", err)
		}
		if m.adbFileSystems == nil {
			m.adbFileSystems = make(map[string]*adbfs.FS)
		}
		m.adbFileSystems[serial] = fsys
	}
	if devicePath == "" || devicePath == "/" {
		devicePath = "/sdcard"
	}
	m.activePanel().SetLocation(vfs.NewLocation(fsys, devicePath, vfs.KindADB, fsys.Label()))
	m.releaseUnusedConnections()
	return m, m.activePanel().LoadDir()
}
