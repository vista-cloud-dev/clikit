---
name: cli-discovery-ux
description: clikit discovery UX — Phase 1 (styled grouped help + pager) and Phase 2 (interactive `explore` Bubbletea palette), both landed across m/v 2026-06-26. COMPLETE — a Phase 3 browser/Miller-columns face is rejected as out of scope (Miller columns browse data, not a fixed command tree).
metadata:
  type: project
---

**clikit command-discovery UX — Phase 1 landed (2026-06-26).** Gave the m/v CLI
suite vista-info-hub-style discoverability by adding a styled help renderer to the
**shared** clikit (one change upgrades m, v, and every future domain). Proposal:
`clikit/docs/proposals/cli-discovery-ux.md`.

**What shipped in clikit (this increment):**
- `help.go` — custom Kong help printer installed via `kong.Help(helpPrinter)` in
  `run.go`. Renders through the existing `*Context` style primitives (so it
  degrades to clean plain text off a TTY / under `NO_COLOR` automatically).
  - **Curated category groups** from Kong-native `group:""` tags
    (`groupsFrom` buckets `node.Children` by `Node.Group`, first-seen order;
    untagged → trailing **"Commands"** bucket). Title defaults to the tag key.
  - **Two-tier disclosure**: bare `<tool>` → compact **landing page**
    (no global flags, "run --help" hint); `<tool> help` / `--help` → **full**
    grouped surface + Global flags. Both intercepted in `run.go` (`os.Args[1:]`
    empty → landing; `["help"]` → full); `--help` flows through Kong's
    `helpFlag.BeforeReset → printHelp`, which then calls `os.Exit(0)`.
  - Leaf command help: usage (`app.Name + node.Summary()`), help/detail,
    Arguments, Flags (command flags minus globals/`--help`).
- `pager.go` — `pageThrough` pipes long output through `$PAGER` (default
  `less -FRX`) on a TTY, direct-writes otherwise. Color is baked into the buffer
  **before** paging (decided by the real TTY, not the pipe), so it survives.
  Overrides: `--no-pager` (caller), `CLIKIT_NO_PAGER` env.

**Non-obvious gotchas:**
- **lipgloss strips ANSI when stdout isn't a TTY** — even with our `Color=true`.
  So under `go test` (no TTY) color output is blank; the color test forces it with
  `lipgloss.SetColorProfile(termenv.TrueColor)`. Production on a real TTY is fine.
- `newRenderContext(w, color)` is the testable Context constructor (writes to any
  writer, color chosen explicitly) vs `NewContext` which hard-wires `os.Stdout` +
  TTY detection. Help color resolves from env+TTY (`helpColorEnabled`) because the
  help flag fires *before* `--no-color` is bound.
- Reused `flagSchema`/`valueType`/`splitEnum` from `schema.go` for flag/arg
  extraction — same reflection source as `m schema`, so help and schema can't
  drift.

**Gates (all green):** gofmt, `go vet`, golangci-lint, `go test -race` (new tests:
`help_test.go`, `pager_test.go`; clikit had none before — 22.5% cov), govulncheck.
Smoke-tested end-to-end via a throwaway binary (replace → local clikit): landing /
full / leaf surfaces, color on pty (16 ESC) vs `NO_COLOR=1` (0), normal command +
error paths unaffected.

**Next (per proposal §5, NOT yet done):**
1. Tag a clikit release (e.g. `v0.2.0`) so consumers can pin it.
2. **Migrate m-cli to the shared clikit module** — it still vendors `m-cli/clikit/`
   (divergence is doc-comment-only): add the require, rewrite the import path in 6
   files, delete the dir, `go mod tidy`, gates.
3. Add `group:""` tags + curated landing sets in m-cli / v-cli / v-pkg.

**Phase 2 refinements (2026-06-26, clikit v0.4.0):**
- **2-D navigation.** Layout is one row per category: the category NAME is cursor
  **column 0**, its commands are columns 1..N. `←→` move within a row (across name +
  commands), `↑↓` between category rows (col clamped). `paletteState` is now
  `{row,col}` not a flat index; `paletteCats()` replaces `orderedItems()` and carries
  a per-category **description**. `⌫`/Backspace = back, `⏎` = open/run; left/right are
  no longer back/descend.
- **Focusable category names.** Landing on a name (col 0) shows "what the category
  is" in the bottom bar (`catLine`): name + description + `[N commands]`. Descriptions:
  Kong group Description (ExplicitGroups) → built-in `catBlurb` map → count fallback.
- **Colors.** Category names bold white; commands **white**; cursor (on name OR
  command) **bold light blue** (`expSel`, color 117). Detail summary yellow; `[status]`
  green/blue/gray unchanged.
- **Compact landing for a bare invocation.** `m`/`v`/`v-pkg` with no args now print a
  short intro — tagline + one-line-per-category overview + pointers (`help`/`explore`/
  `<cmd> --help`) — via `writeLanding`, and it is **never paged**. (Before: bare `m`
  dumped the full grouped list into `less`, requiring `q`.) Full surface is still
  `<tool> help` / `--help`.
- **Height-aware paging.** `pageThrough` now pages only when content is **taller than
  the terminal** (`tallerThanScreen`); short help/landing print directly and return to
  the prompt — fixes the "stuck in `less`, press q" annoyance (their `$PAGER` lacked
  quit-if-one-screen).

**Renamed `explore` → `menu` (2026-06-26, clikit v0.5.0).** The interactive
palette subcommand is now `menu`: clikit type `ExploreCmd` → **`MenuCmd`** (error
code `EXPLORE_FAILED` → `MENU_FAILED`); the landing pointer now reads `"<tool>
menu" to browse`. Consumers mount it as `Menu clikit.MenuCmd` (field name → Kong
command name). Rolled out the same increment: clikit v0.5.0 tagged, then m-cli /
v-cli / v-pkg / v-rpc each repinned clikit v0.5.0 + renamed the field. The rename
is API-breaking for the type name, but safe across the v suite because the menu
command lives only in each repo's `main.go` (never in the imported `pkgcli`/
`rpccli` packages), so MVS upgrading clikit to v0.5.0 for the umbrella build never
hits a dangling `ExploreCmd` reference. Internal identifiers (`exploreModel`,
`expCat` styles, `explore.go`) were left as-is — invisible. Below this line,
"explore" is the historical name.

---

A once-mooted Phase 3 `browser` (Miller columns) is **rejected as out of scope**:
Miller columns are for browsing **data** (deep open-ended hierarchies), not a small
fixed command surface that `explore` already covers. Discovery-UX ends at Phase 2.

---

**Phase 2 — interactive `explore` palette LANDED in clikit (2026-06-26).** A
Bubbletea TUI over the CLI's own command tree, mounted as the reusable
`clikit.ExploreCmd` (like `SchemaCmd`/`VersionCmd`) so every tool gets `explore`
for free. New deps: `github.com/charmbracelet/bubbletea` (+ bubbles family).

- `palette.go` — **pure, TUI-free navigation state** (so it's fully unit-testable):
  `orderedItems(node)` builds grouped rows from `*kong.Node` (same group-order as
  the help, untagged → "Commands"), each carrying `parent` (has children → descend)
  and `needsArg` (has a required positional → not runnable). `paletteState` holds a
  breadcrumb `stack []*kong.Node` + cursor + filter; methods `move/descend/enter/
  back/setFilter/selected`.
- `explore.go` — the Bubbletea `exploreModel` (value model holding a `*paletteState`
  pointer so cursor survives Bubbletea's value-copy) + `ExploreCmd`. Keys: ↑↓/jk
  move, →/l descend, ←/h/backspace up, ⏎ select-leaf (or descend), `/` filter, q/esc
  quit. Detail strip shows summary + a **runnable / needs-args / group badge**.
  On selecting a leaf it quits and prints that command's help via `emitHelp`.
  A **one-line bottom status bar** (`detailLine`) shows the focused command's full
  path + summary + badge, width-truncated so it never wraps.
- `ExploreCmd.Run(c, k)` uses `interactiveTTY()` (stdin AND stdout TTY); **non-TTY
  falls back to the full styled help** (verified: `m explore | head` → grouped help,
  exit 0).

**Why `*kong.Node` not `SchemaDoc`:** the palette needs the `group` tag +
children + required-positional info; `SchemaDoc` doesn't carry `group`. The live
Kong model is equally drift-free (it IS the registry), so the palette reads it
directly. (A future option: add `Group` to `SchemaCommand` for machine consumers.)

**Verification:** unit tests cover navigation (order/clamp/descend/back/leaf-select/
filter) and the model (down, filter flow, enter-leaf-chooses-and-quits, right-
descends, View shows groups+footer, q quits) — all green; `make check`
(vet/lint/`-race`, 38.3% cov) + govulncheck clean. **GOTCHA:** driving the real
alt-screen TUI through `script` with piped stdin HANGS — don't smoke-test the
interactive path that way; rely on the unit tests + the non-TTY fallback, and let
a human try the real terminal. Rollout to m/v (mount `ExploreCmd` + repin) is the
next step.
