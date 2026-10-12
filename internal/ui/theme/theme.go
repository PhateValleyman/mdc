package theme

import "github.com/charmbracelet/lipgloss"

// OverlayPalette contains semantic colors shared by dialogs, pickers, help and settings.
// Theme files may override these in [overlay]; legacy themes derive them from panel colors.
type OverlayPalette struct {
	Background lipgloss.Color
	Foreground lipgloss.Color
	Subtle     lipgloss.Color
	Accent     lipgloss.Color
	Highlight  lipgloss.Color
	CursorBG   lipgloss.Color
	Success    lipgloss.Color
	Danger     lipgloss.Color
	Border     lipgloss.Color
}

// OverlayStyles is the canonical style set for modal and picker content.
type OverlayStyles struct {
	Base        lipgloss.Style
	Dim         lipgloss.Style
	Prompt      lipgloss.Style
	Highlight   lipgloss.Style
	Selected    lipgloss.Style
	Input       lipgloss.Style
	InputCursor lipgloss.Style
	Success     lipgloss.Style
	Danger      lipgloss.Style
	Border      lipgloss.Style
	Title       lipgloss.Style
}

// Theme holds all styles used by the application.
type Theme struct {
	// Panel
	PanelBorder       lipgloss.Style
	PanelBorderActive lipgloss.Style
	PanelHeader       lipgloss.Style
	PanelHeaderActive lipgloss.Style

	// File list
	FileNormal            lipgloss.Style
	FileDir               lipgloss.Style
	FileExec              lipgloss.Style
	FileSymlink           lipgloss.Style
	FileCursor            lipgloss.Style
	FileCursorDir         lipgloss.Style
	FileSelected          lipgloss.Style
	FileCursorSelected    lipgloss.Style
	FileCursorDirSelected lipgloss.Style

	// Status bar and menu bar
	StatusBar lipgloss.Style
	MenuBar   lipgloss.Style
	FKeyHint  lipgloss.Style
	FKeyLabel lipgloss.Style

	// Dialogs and overlays
	Overlay OverlayPalette

	// Command line
	CmdLine lipgloss.Style
}

// OverlayStyles returns the semantic overlay styles for this theme, filling any
// missing colors from the MC Classic palette for zero-value or legacy Theme values.
func (t Theme) OverlayStyles() OverlayStyles {
	fallback := defaultOverlayPalette()
	p := t.Overlay
	if p.Background == "" {
		p.Background = fallback.Background
	}
	if p.Foreground == "" {
		p.Foreground = fallback.Foreground
	}
	if p.Subtle == "" {
		p.Subtle = fallback.Subtle
	}
	if p.Accent == "" {
		p.Accent = fallback.Accent
	}
	if p.Highlight == "" {
		p.Highlight = fallback.Highlight
	}
	if p.CursorBG == "" {
		p.CursorBG = fallback.CursorBG
	}
	if p.Success == "" {
		p.Success = fallback.Success
	}
	if p.Danger == "" {
		p.Danger = fallback.Danger
	}
	if p.Border == "" {
		p.Border = fallback.Border
	}

	base := lipgloss.NewStyle().Background(p.Background).Foreground(p.Foreground)
	return OverlayStyles{
		Base:        base,
		Dim:         lipgloss.NewStyle().Background(p.Background).Foreground(p.Subtle),
		Prompt:      lipgloss.NewStyle().Background(p.Background).Foreground(p.Accent).Bold(true),
		Highlight:   lipgloss.NewStyle().Background(p.Background).Foreground(p.Highlight).Bold(true),
		Selected:    lipgloss.NewStyle().Background(p.CursorBG).Foreground(p.Foreground),
		Input:       lipgloss.NewStyle().Background(p.Background).Foreground(p.Highlight),
		InputCursor: lipgloss.NewStyle().Background(p.Highlight).Foreground(p.Background),
		Success:     lipgloss.NewStyle().Background(p.Background).Foreground(p.Success),
		Danger:      lipgloss.NewStyle().Background(p.Background).Foreground(p.Danger),
		Border:      lipgloss.NewStyle().Background(p.Background).Foreground(p.Border),
		Title:       lipgloss.NewStyle().Background(p.Background).Foreground(p.Accent).Bold(true),
	}
}
