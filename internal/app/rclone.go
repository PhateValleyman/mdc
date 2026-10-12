package app

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	rcloneui "github.com/kooler/MiddayCommander/internal/ui/rclone"
	"github.com/kooler/MiddayCommander/internal/vfs"
	"github.com/kooler/MiddayCommander/internal/vfs/rclonefs"
)

func (m Model) startRcloneRemotes() (tea.Model, tea.Cmd) {
	picker := rcloneui.New(m.width, m.height)
	m.rclonePicker = &picker
	return m, rcloneui.LoadRemotesCmd()
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

func (m Model) rcloneConfigEditCmd() tea.Cmd {
	cmd := exec.Command("rclone", "config", "edit")
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return rcloneui.ConfigEditFinishedMsg{Err: err}
	})
}

func (m Model) handleRcloneConfigEdited(err error) (tea.Model, tea.Cmd) {
	if err != nil {
		m.rclonePicker = nil
		return m.showError("rclone config edit", fmt.Errorf("%w", err))
	}
	if m.rclonePicker != nil {
		m.rclonePicker.BeginRefresh()
	}
	return m, tea.Batch(rcloneui.LoadRemotesCmd(), m.refreshBothPanels())
}
