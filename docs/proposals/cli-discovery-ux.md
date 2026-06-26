---
title: clikit command-discovery UX — styled curated help (and an interactive discovery path)
status: proposed
version: v0.2.0
created: 2026-06-26
last_modified: 2026-06-26
revisions: 2
doc_type: [PROPOSAL]
layer: m
related: [../../README.md]
---

# clikit command-discovery UX — styled curated help

> **Goal.** Give the whole `m`/`v` CLI suite the kind of **colorful, progressively
> discoverable** command surface that `vista-info-hub` (the VistA-Copilot
> *navigator*) has — a landing page, curated category groups, accent-colored
> command names, dimmed summaries — by adding a **styled help renderer to
> `clikit`**. Because clikit is the shared toolkit, one change upgrades `m`, `v`,
> and every future domain at once. **This proposal scopes Phase 1 only**
> (styled curated help + pager); Phases 2–3 (interactive faces) are sketched as
> a deliberate follow-on, not committed.

---

## 1. Motivation

`vista-info-hub` reads as a polished, explorable tool. Its UX comes from three
layers of *progressive disclosure*:

1. A bare landing screen (`vista`) — a one-screen, curated overview.
2. A grouped full listing (`vista help`) — ~10 **editorial** categories, not an
   alphabetical dump; command names in cyan, hints dimmed.
3. Interactive faces (`vista menu` palette, `vista browser` Miller columns) —
   arrow-navigate, badges for runnable-vs-needs-args, detail-on-focus.

The `v` suite (and `m`) looks plain by comparison, **not because it lacks the
styling tools** but because help is still Kong's stock renderer.

### What we already have (the gap is small)

- **`clikit/style.go`** already ships the entire presentation palette `vista-info-hub`
  uses: adaptive semantic colors, Unicode+ASCII glyph fallback, and
  `Title/Subtitle/List/KV/Rule/Panel/Tree/Table/Badge` + status lines — all
  `Color`-gated so they no-op off a TTY / under `NO_COLOR`.
- **`clikit/schema.go`** already reflects the **entire Kong command/flag/enum
  tree** into `SchemaDoc` (`BuildSchema`). That is the exact analog of
  `vista-info-hub`'s single operation registry — one introspection source that a
  help renderer *and* (later) an interactive menu can both read, with **zero
  drift** from the real command surface.

### What's missing

- **`clikit/run.go:27`** wires Kong's stock help:
  `kong.ConfigureHelp(kong.HelpOptions{Compact: true, FlagsLast: true})`.
  That output is **plain text, alphabetical, ungrouped, uncolored**.
- No interactive discovery faces (Phase 2/3; out of scope here).

### Why port, not depend

`vista-info-hub` lives in a **different org** (VistA-Copilot / navigator) and is
built on **Cobra + a runtime registry**. The `v` suite is **Kong + static structs +
reflected `SchemaDoc`**. We therefore lift the *design* (curated categories,
two-tier disclosure, badge/detail discovery) and re-implement it against Kong —
**no cross-org dependency, no framework mismatch**. clikit is `m`-layer
(engine-neutral), so this is a pure toolchain change with no waterline concern.

---

## 2. Scope — Phase 1

Deliver the bulk of the perceived polish with **no new dependencies** (lipgloss is
already a clikit dependency):

1. **Custom Kong help renderer in clikit** — replace the stock printer via
   `kong.Help(...)` (Kong supports a fully custom `HelpPrinter`), rendering
   through the existing `*Context` style primitives so it auto-degrades on
   `NO_COLOR` / non-TTY exactly like the rest of clikit's output.
2. **Curated category grouping** — group commands editorially (e.g. for `m`:
   *Author* `fmt/lint/lsp`, *Verify* `test/coverage/arch`, *Engine* `…`,
   *Introspect* `schema/version`). Expressed with Kong's native **`group:""`
   tags** on the command fields + a small ordering table in clikit — each domain
   declares its own group inline, so there is no central list to drift.
3. **Two-tier disclosure**, mirroring `vista-info-hub`:
   - bare `v` / `m` → a **one-screen landing page** (a handful of key commands +
     "run `v help` for everything");
   - `v help` → the **full grouped surface**.
4. **Pager with force-color** — pipe long help/output through `less -FRX` while
   forcing color (the pager's destination is a TTY even when our stdout pipe is
   not). Honor `$PAGER`, `--no-pager`, and a `*_NO_PAGER` env override.

### Explicitly out of scope (Phase 2/3, see §6)

- Bubbletea-based `explore`/`menu` interactive palette.
- Ranger-style `browser` Miller-column explorer.
- Auto-running a selected command from an interactive face.

---

## 3. Design

### 3.1 Help renderer

- New file **`clikit/help.go`**: a `helpPrinter(options kong.HelpOptions, kctx
  *kong.Context) error` installed in `Run` via `kong.Help(helpPrinter)`.
- It resolves a `*Context` (same `Globals`/TTY/`Color` resolution as command
  output) and renders:
  - **root help**: program description, then category groups via `Title` +
    accent-colored command names + `Faint` summaries (reuse `KV`/`List`);
  - **leaf/group help**: usage line, args, flags (grouped: command flags then
    globals), examples — styled, but structurally the same info Kong shows today.
- **Degradation is free**: every primitive already no-ops when `!Color`, so
  `NO_COLOR`, pipes, and dumb terminals get clean plain text. CI/`--output json`
  unaffected (help is a human surface).

### 3.2 Category metadata (no new registry)

- Commands carry a Kong **`group:"Author"`** (etc.) tag. clikit defines an
  ordered list of known group titles (and renders unknown/untagged commands under
  a trailing "Other"). The grouping for the help view is read from the **same
  reflected model** `BuildSchema` already walks — keeping help and `schema`
  output consistent.

### 3.3 Landing page vs full help

- `Run` distinguishes "no command given" (→ landing page: curated short list) from
  explicit `help` / `--help` (→ full grouped surface). A per-tool curated landing
  set is supplied by the caller (small slice of command paths) or defaults to the
  top-level groups truncated to one screen.

### 3.4 Pager

- New **`clikit/pager.go`**: `pageThrough(c *Context, render func(w io.Writer, forceColor bool))`.
  When stdout is a TTY and paging isn't disabled, spawn `$PAGER` (default `less
  -FRX`) and render with color forced on; otherwise render directly. Wire help
  output (and optionally long command output) through it.

---

## 4. Testing (TDD — hard rule)

Per house rules, write tests first. The renderer is pure-ish (writes to an
`io.Writer`), so it's straightforward to table-test:

- **plain mode** (`Color=false`): assert grouped structure, command names,
  summaries present, and **no ANSI** in output.
- **color mode** (`Color=true`): assert ANSI present and group ordering.
- **landing vs full**: bare invocation emits the short set; `help` emits all
  groups.
- **`NO_COLOR` / non-UTF-8 locale**: glyphs degrade to ASCII, no color.
- **pager**: with paging disabled, output goes straight to the writer (no
  subprocess); selection of `$PAGER` respected.
- Group tags round-trip: every command appears under exactly one group; untagged
  → "Other".

Gates before commit (clikit `Makefile` / house Go gate): `gofmt`, `go vet`,
`golangci-lint`, `go test -race`, `govulncheck`.

---

## 5. Rollout & sequencing (leaf-first)

**Decision (2026-06-26): migrate `m-cli` to the shared module — no stopgap.** The
vendored `m-cli/clikit/` copy was measured against the shared module: 6 of 8 files
are **byte-identical** (including all of `style.go`), and the only two that differ
(`globals.go`, `version.go`) differ **solely in doc-comment text** (package blurb /
one `ldflags` example path) — **zero functional divergence**. Migrating now is the
cheapest it will ever be and is the precondition for "one clikit change upgrades
the whole suite."

1. **clikit (leaf)** — land `help.go` + `pager.go` + group plumbing; `Run` opts in
   to the custom printer. Existing consumers keep working (untagged commands fall
   under "Other"; output identical-or-better). Increment-protocol commit. Tag a
   release (e.g. `clikit v0.2.0`) so consumers can pin it.
2. **`m-cli` migration (companion increment)** — add
   `require github.com/vista-cloud-dev/clikit vX.Y.Z`; rewrite the import path
   `…/m-cli/clikit` → `…/clikit` in the **6 importing files** (`main.go`,
   `vista_cmd.go`, `main_test.go`, `internal/dispatch/dispatch.go` + its test,
   `internal/arch/arch_test.go`); delete `m-cli/clikit/`; `go mod tidy`; run gates.
3. **`v-cli` / `v-pkg` / `m-cli`** — add `group:""` tags + a curated landing set;
   verify the surface. Commit per repo.

Each repo runs the **Increment Protocol** (memory + tracker + commit/push to
`main`) as its slice goes green.

---

## 6. Future phases (sketch only — not committed)

- **Phase 2 — `explore` palette.** Add Bubbletea+Bubbles to clikit; a new
  `clikit/discover/` reads `SchemaDoc` → category→command grid with a detail strip
  and **runnable-vs-needs-args badges** (derived from required args in the
  schema). Mounted as a reusable `clikit.ExploreCmd`, like `SchemaCmd`. This is
  `vista-info-hub`'s `menu`.
- **Phase 3 — `browser`.** Ranger-style Miller-column explorer with breadcrumbs +
  fuzzy filter over the same `SchemaDoc`. High polish, low necessity; gate on
  whether Phase 2's interaction model proves wanted.

Both are additive and reuse Phase 1's introspection + style foundation.

---

## 7. Risks / open questions

- **Kong help API surface — RESOLVED (verified against kong v1.15.0).** The hook
  is sufficient: `kong.Help(printer)` with
  `type HelpPrinter func(options HelpOptions, ctx *Context) error` fully replaces
  the stock renderer. The `*kong.Context` exposes `Selected() *Node` and the root
  model; every `*Node` carries `Help`, `Detail`, `Group`, `Flags`, `Positional`,
  `Children`, `Aliases`, `Type`, `Hidden` — so root and leaf help render straight
  off the live model (no `BuildSchema` re-walk needed). Curated categories are
  **native**: a `group:""` tag populates `Node.Group`, `kong.ExplicitGroups([]Group)`
  attaches title/description, and `Node.ClosestGroup()` handles inheritance. We
  write only the renderer + pager — no framework workaround, no fallback path.
- **`m-cli` vendored clikit — RESOLVED: migrate now** (see §5; divergence is
  doc-comment-only, so the fork costs nothing to delete and everything to keep).
- **Landing-page curation.** Who picks the "key commands"? Default to top group
  heads; let each tool override with a small slice. Avoid it becoming a second
  drift-prone list — derive from groups where possible.
