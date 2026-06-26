# Memory index — clikit (per-repo)

Per-repo memory for the shared CLI toolkit `clikit`, committed with its code (per
the org memory rules). One line per entry; detail in the linked file.

- [CLI discovery UX](cli-discovery-ux.md) — Phase 1 LANDED 2026-06-26 across m/v: styled grouped help (`help.go`, via `kong.Help`) + pager (`pager.go`), two-tier landing/full. Phase 2 LANDED in clikit 2026-06-26: interactive `explore` Bubbletea palette (`palette.go` pure nav + `explore.go` model + `ExploreCmd`) over the `*kong.Node` tree; non-TTY falls back to full help. GOTCHAs: lipgloss strips ANSI off-TTY (test forces `termenv.TrueColor`); driving the alt-screen TUI via `script`+piped-stdin HANGS — test the model directly. NEXT: mount `ExploreCmd` in m/v + repin (clikit v0.3.0). Phase 3 (browser) = sketch. Proposal: `docs/proposals/cli-discovery-ux.md`.
