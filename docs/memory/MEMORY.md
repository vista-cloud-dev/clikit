# Memory index — clikit (per-repo)

Per-repo memory for the shared CLI toolkit `clikit`, committed with its code (per
the org memory rules). One line per entry; detail in the linked file.

- [clikit durable gotchas](clikit-durable-gotchas.md) — durable clikit lessons + API invariants from the discovery-UX work: lipgloss strips ANSI off-TTY (tests force `termenv.TrueColor`); help/schema share `schema.go` reflection so they can't drift; `pageThrough` is height-aware + bakes color pre-pipe; palette nav is pure (`palette.go`) vs Bubbletea model (`explore.go`), reads `*kong.Node` not `SchemaDoc`; driving the alt-screen TUI via `script`+piped-stdin HANGS — test the model directly. API: the interactive subcommand is `menu` (`MenuCmd`/`MENU_FAILED`, mount `Menu clikit.MenuCmd`); internal names stay `explore*`. Full narrative + tracker: `docs/proposals/cli-discovery-ux.md` (complete).
