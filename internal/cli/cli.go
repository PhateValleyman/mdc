package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kooler/MiddayCommander/internal/ui/theme"
)

// Options holds the parsed command-line flags and arguments.
type Options struct {
	ShowHelp     bool
	ShowVersion  bool
	ReturnPath   bool
	Theme        string
	ColorMode    theme.ColorMode
	ColorProfile string
	LeftPath     string
	RightPath    string
}

// Parse parses command-line arguments (excluding os.Args[0]).
func Parse(args []string) (Options, error) {
	opts := Options{
		ColorMode: theme.ColorAuto,
	}

	var positional []string
	inFlags := true

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if inFlags && arg == "--" {
			inFlags = false
			continue
		}

		if inFlags && strings.HasPrefix(arg, "-") && arg != "-" {
			// Long flags: --flag or --flag=value
			if strings.HasPrefix(arg, "--") {
				flagName := arg[2:]
				flagVal := ""
				hasVal := false

				if idx := strings.IndexByte(flagName, '='); idx >= 0 {
					flagVal = flagName[idx+1:]
					flagName = flagName[:idx]
					hasVal = true
				}

				switch flagName {
				case "help":
					opts.ShowHelp = true
				case "version":
					opts.ShowVersion = true
				case "return-path":
					opts.ReturnPath = true
				case "color":
					if hasVal {
						switch strings.ToLower(flagVal) {
						case "always", "yes", "true", "1":
							opts.ColorMode = theme.ColorAlways
						case "never", "no", "false", "0":
							opts.ColorMode = theme.ColorNever
						case "auto":
							opts.ColorMode = theme.ColorAuto
						default:
							return opts, fmt.Errorf("invalid value for --color: %q (expected always, auto, or never)", flagVal)
						}
					} else {
						opts.ColorMode = theme.ColorAlways
					}
				case "no-color", "black-white", "bw":
					opts.ColorMode = theme.ColorNever
				case "color-profile":
					if !hasVal {
						if i+1 >= len(args) {
							return opts, fmt.Errorf("--color-profile requires an argument")
						}
						i++
						flagVal = args[i]
					}
					if _, ok := theme.ParseColorProfile(flagVal); !ok {
						return opts, fmt.Errorf("invalid value for --color-profile: %q (expected truecolor, 256, ansi, or ascii)", flagVal)
					}
					opts.ColorProfile = flagVal
				case "theme":
					if !hasVal {
						if i+1 >= len(args) {
							return opts, fmt.Errorf("--theme requires an argument")
						}
						i++
						flagVal = args[i]
					}
					if flagVal == "" {
						return opts, fmt.Errorf("--theme requires a non-empty name")
					}
					opts.Theme = flagVal
				default:
					return opts, fmt.Errorf("unknown flag: --%s", flagName)
				}
				continue
			}

			// Short flags: -h, -v, -r, -c, -b, -t <theme>, or combined like -rc
			flagGroup := arg[1:]
			if flagGroup == "?" {
				opts.ShowHelp = true
				continue
			}

			skipFlagGroup := false
			for j := 0; j < len(flagGroup); j++ {
				ch := flagGroup[j]
				switch ch {
				case 'h', '?':
					opts.ShowHelp = true
				case 'v':
					opts.ShowVersion = true
				case 'r':
					opts.ReturnPath = true
				case 'c':
					opts.ColorMode = theme.ColorAlways
				case 'b':
					opts.ColorMode = theme.ColorNever
				case 't':
					// -t can take value immediately (-tName) or as next arg (-t Name)
					rest := flagGroup[j+1:]
					if rest != "" {
						if strings.HasPrefix(rest, "=") {
							rest = rest[1:]
						}
						opts.Theme = rest
					} else {
						if i+1 >= len(args) {
							return opts, fmt.Errorf("-t requires an argument")
						}
						i++
						opts.Theme = args[i]
					}
					skipFlagGroup = true
				default:
					return opts, fmt.Errorf("unknown flag: -%c", ch)
				}
				if skipFlagGroup {
					break
				}
			}
			continue
		}

		// Positional argument
		positional = append(positional, arg)
	}

	if len(positional) > 0 {
		opts.LeftPath = positional[0]
	}
	if len(positional) > 1 {
		opts.RightPath = positional[1]
	}
	if len(positional) > 2 {
		return opts, fmt.Errorf("too many positional arguments (maximum 2 paths allowed, got %d)", len(positional))
	}

	return opts, nil
}

// HelpText returns the formatted help string.
func HelpText() string {
	prog := "mdc"
	if len(os.Args) > 0 && os.Args[0] != "" {
		prog = filepath.Base(os.Args[0])
	}

	return fmt.Sprintf(`MiddayCommander (%s) - terminal file manager

Usage:
  %s [options] [path1] [path2]

Arguments:
  path1                   Initial directory for the active (left) panel
  path2                   Initial directory for the inactive (right) panel

Options:
  -h, --help              Show this help message and exit
  -v, --version           Show version information and exit
  -r, --return-path       Print active panel path on exit (for shell 'cd' wrapper)
  -t, --theme <name>      Override theme (e.g. catppuccin-mocha, mc-classic, tokyo-night)
  -c, --color             Force colored output (ANSI 256 / TrueColor)
  -b, --no-color          Disable colors (monochrome mode with reverse cursor)
      --color=<when>      When to use color: 'always', 'auto', or 'never' (default: auto)
      --color-profile=<p> Force specific color profile:
                            'truecolor' (24-bit RGB)
                            '256'       (8-bit ANSI 256)
                            'ansi'      (4-bit ANSI 16 colors)
                            'ascii'     (0-bit monochrome)

Shell Integration:
  To change directory on exit, add this function to your shell config (~/.bashrc or ~/.zshrc):
    mc() {
      local dir
      dir="$(%s -r "$@")" && [ -n "$dir" ] && cd "$dir"
    }

Key Bindings:
  Tab                     Switch between left and right panels
  F3                      View file
  F4                      Edit file
  F5                      Copy selected files
  F6                      Move / rename selected files
  F7                      Create new directory
  F8                      Delete selected files
  F10, Ctrl+C             Quit MiddayCommander

Examples:
  %s                      Open current directory and home directory
  %s /var/log             Open /var/log in active panel
  %s /etc /var            Open /etc in left panel and /var in right panel
  %s -c                   Force colors when connecting over SSH
  %s -t mc-classic        Start with classic Midnight Commander blue theme
`, prog, prog, prog, prog, prog, prog, prog, prog)
}
