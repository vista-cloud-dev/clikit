# Memory index — clikit (per-repo)

Per-repo memory for the shared CLI toolkit `clikit`, committed with its code (per
the org memory rules). One line per entry; detail in the linked file.

- [CLI discovery UX](cli-discovery-ux.md) — Phase 1 LANDED 2026-06-26: styled, curated, grouped help renderer (`help.go`, via `kong.Help`) + pager (`pager.go`), two-tier landing/full surface, replacing Kong's stock help for the whole m/v suite. GOTCHA: lipgloss strips ANSI off-TTY (test forces `termenv.TrueColor`). NEXT: tag clikit release → migrate m-cli off its vendored copy → add `group:""` tags in m/v. Proposal: `docs/proposals/cli-discovery-ux.md`.
