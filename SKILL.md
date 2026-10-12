# Project guidance — Midday Commander (mdc)

## Before changing code
- Read this file and the relevant package implementation/tests before making changes.
- Preserve the existing Go + Bubble Tea architecture and current cross-platform build targets.
- Prefer small, reviewable changes with focused tests; do not claim checks passed unless they were run.
- Keep user-facing errors useful but avoid displaying OAuth tokens, passwords, or raw provider diagnostics.

## Validation
- Run `gofmt` on changed Go files.
- Run `go test ./...`, `go vet ./...`, and `go build ./...` when a Go toolchain is available.
- Check the README and `config.example.toml` when keyboard bindings or user workflows change.
- If tests cannot be run in the available environment, state that explicitly and provide the exact checks still required.

## UI and remote integrations
- Keep slow network/process work outside Bubble Tea's update/render loop.
- For interactive external CLI workflows, suspend the TUI using Bubble Tea's process execution support rather than trying to embed another terminal prompt in an overlay.
- Use rclone's native terminal configuration/authentication commands and token storage; never pass configuration answers or secrets in process arguments or invent a stdin substitute for rclone's interactive flow. Never implement provider-specific OAuth token handling inside MDC unless there is a compelling documented reason.
- Keep key hints in overlays synchronized with actual key handling and README documentation.
- Preserve dual-panel navigation and existing MC-style function-key behavior.
- Use `theme.Theme.OverlayStyles()` for dialogs, pickers, help, and settings; legacy themes derive semantic overlay colors from panel/menu roles, with optional `[overlay]` TOML overrides.
- Keep any trailing menu additions independent of the ten F1–F10 slots, and cover settings routing plus narrow-terminal behavior with focused tests.
- Pass filesystem-, provider-, and device-derived labels through `internal/ui/text.SafeTerminalText` before terminal rendering; keep stored paths and values unchanged for operations.

## Compatibility and security
- Avoid new runtime dependencies unless justified.
- Preserve the ARMv5/uClibc ZyXEL build target and Windows/macOS/Linux support.
- Keep ADB directory metadata lookups bounded; do not fan out unbounded device subprocesses.
- Never commit credentials, access tokens, personal config files, generated binaries, or temporary test data.
