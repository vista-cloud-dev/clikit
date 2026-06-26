# Memory index — clikit (per-repo)

Per-repo memory for the shared CLI toolkit `clikit`, committed with its code (per
the org memory rules). One line per entry; detail in the linked file.

- [CLI discovery UX](cli-discovery-ux.md) — Phase 1 LANDED 2026-06-26 across m/v: styled grouped help (`help.go`, via `kong.Help`) + pager (`pager.go`), two-tier landing/full. Phase 2 LANDED in clikit 2026-06-26: interactive `explore` Bubbletea palette (`palette.go` pure nav + `explore.go` model + `ExploreCmd`) over the `*kong.Node` tree; non-TTY falls back to full help. GOTCHAs: lipgloss strips ANSI off-TTY (test forces `termenv.TrueColor`); driving the alt-screen TUI via `script`+piped-stdin HANGS — test the model directly. Phase 2 ROLLED OUT 2026-06-26: `m explore` / `v-pkg explore` / `v explore`, all clikit v0.3.2 (palette restyled: bold-white groups, green commands, yellow detail, plain `[status]` text, one-line bottom bar). COMPLETE — Phase 3 browser/Miller-columns REJECTED (out of scope: Miller columns browse data, not a fixed command tree). Proposal: `docs/proposals/cli-discovery-ux.md`.
