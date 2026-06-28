---
name: clikit-durable-gotchas
description: Durable clikit gotchas + API invariants distilled from the discovery-UX workstream (styled help, pager, the `menu`/`explore` Bubbletea palette). The per-increment rollout narrative and tracker live in the proposal; this file keeps only what code/git don't already say.
metadata:
  type: project
---

# clikit — durable gotchas & invariants

Full design + implementation tracker (Phase 1 styled help/pager, Phase 2
interactive palette, Phase 3 rejected): **`docs/proposals/cli-discovery-ux.md`**
(complete; the code comments in `help.go`/`explore.go`/`palette.go` pin that path).
This file keeps only the **durable** lessons — the per-increment "LANDED"/version
rollout journal that was here failed the keep-test (it duplicated the proposal's
§6a tracker + git history).

## Help / styling / pager

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
- **Pager color**: `pageThrough` bakes color into the buffer **before** paging
  (decided by the real TTY, not the pipe), so it survives. Overrides: `--no-pager`
  (caller), `CLIKIT_NO_PAGER` env.
- **Height-aware paging**: `pageThrough` pages only when content is **taller than
  the terminal** (`tallerThanScreen`); short help/landing print directly and return
  to the prompt — fixes the "stuck in `less`, press q" annoyance (a `$PAGER` that
  lacks quit-if-one-screen). The compact landing for a bare invocation
  (`writeLanding`) is **never paged**.

## Interactive palette (`menu`, internally `explore`)

- **Why `*kong.Node` not `SchemaDoc`:** the palette needs the `group` tag +
  children + required-positional info; `SchemaDoc` doesn't carry `group`. The live
  Kong model is equally drift-free (it IS the registry), so the palette reads it
  directly. (A future option: add `Group` to `SchemaCommand` for machine consumers.)
- Navigation state is kept **TUI-free / pure** in `palette.go` (so it's fully
  unit-testable); the Bubbletea model in `explore.go` is rendering + key handling
  only. The model holds a `*paletteState` **pointer** so the cursor survives
  Bubbletea's value-copy.
- **GOTCHA:** driving the real alt-screen TUI through `script` with piped stdin
  **HANGS** — don't smoke-test the interactive path that way; rely on the unit
  tests + the non-TTY fallback (non-TTY → full styled help, exit 0), and let a
  human try the real terminal.

## API invariant — the `menu` rename (clikit v0.5.0)

The interactive palette subcommand is **`menu`**: exported type **`MenuCmd`**,
error code **`MENU_FAILED`**, landing pointer reads `"<tool> menu" to browse`.
Consumers mount it as `Menu clikit.MenuCmd` (the field name becomes the Kong
command name). Internal identifiers (`exploreModel`, `expCat` styles, the
`explore.go` filename) were deliberately **left as `explore*`** — invisible to
consumers. So in code "explore" is the historical internal name; the public
surface is "menu".
