package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kooler/MiddayCommander/internal/app"
	"github.com/kooler/MiddayCommander/internal/cli"
	"github.com/kooler/MiddayCommander/internal/platform"
	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	opts, err := cli.Parse(os.Args[1:])
	if err != nil {
		prog := "mdc"
		if len(os.Args) > 0 && os.Args[0] != "" {
			prog = filepath.Base(os.Args[0])
		}
		fmt.Fprintf(os.Stderr, "Error: %v\nRun '%s --help' for usage.\n", err, prog)
		os.Exit(1)
	}

	if opts.ShowHelp {
		fmt.Print(cli.HelpText())
		os.Exit(0)
	}

	if opts.ShowVersion {
		fmt.Printf("mdc %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	// With -r, fd 1 is a pipe to the shell (`cd "$(mdc -r)"`). Route the TUI
	// through a separately-opened /dev/tty so fd 1 stays clean for the final
	// path on exit.
	var ttyFile *os.File
	if opts.ReturnPath {
		var err error
		ttyFile, err = openControllingTTY()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer ttyFile.Close()
	}

	// Setup color profile according to user flags and environment detection.
	// This ensures that when running on servers (over SSH with TERM=xterm, etc.),
	// colors are properly enabled rather than falling back to monochrome Ascii.
	theme.SetupColor(opts.ColorMode, opts.ColorProfile, ttyFile)

	// Enable Kitty keyboard protocol (flag 1: disambiguate) so the terminal
	// reports modifier-only key presses (e.g. bare Shift). Terminals that
	// don't support the protocol silently ignore this sequence.
	uiOut := os.Stdout
	if ttyFile != nil {
		uiOut = ttyFile
	}
	_, _ = uiOut.WriteString("\x1b[>1u")
	defer func() { _, _ = uiOut.WriteString("\x1b[<u") }() // disable on exit

	programOpts := []tea.ProgramOption{
		tea.WithAltScreen(),
		// Cell motion, not all motion: only clicks, wheel and drags are used,
		// so bare pointer movement must not cost a full re-render per cell.
		tea.WithMouseCellMotion(),
		// The UI has no animation beyond a 100ms spinner; 30fps halves the
		// renderer's idle wakeups, which matters on low-power hardware.
		tea.WithFPS(30),
		tea.WithFilter(app.KittyFilter),
	}
	if ttyFile != nil {
		programOpts = append(programOpts, tea.WithInput(ttyFile), tea.WithOutput(ttyFile))
	}

	appOpts := app.Options{
		Version:   version,
		LeftPath:  opts.LeftPath,
		RightPath: opts.RightPath,
		Theme:     opts.Theme,
	}

	p := tea.NewProgram(app.NewWithOptions(appOpts), programOpts...)

	// Poll OS-level shift key state and send messages to the Bubble Tea program.
	// Skipped where IsShiftPressed is a stub, so the ticker never runs for nothing.
	if platform.ShiftPollingSupported {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go pollShift(ctx, p)
	}

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.ReturnPath {
		if m, ok := final.(app.Model); ok {
			fmt.Println(m.ActivePanelPath())
		}
	}
}

// pollShift checks the OS modifier state periodically and sends
// ShiftPressMsg / ShiftReleaseMsg when the state changes.
func pollShift(ctx context.Context, p *tea.Program) {
	var wasShift bool
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pressed := platform.IsShiftPressed()
			if pressed != wasShift {
				wasShift = pressed
				if pressed {
					p.Send(app.ShiftPressMsg{})
				} else {
					p.Send(app.ShiftReleaseMsg{})
				}
			}
		}
	}
}
