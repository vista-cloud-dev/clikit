---
name: cli-discovery-ux
description: clikit Phase 1 — styled, curated, grouped help renderer + pager replacing Kong's stock help for the whole m/v CLI suite. Landed in clikit 2026-06-26; m-cli migration + group tags in m/v still pending.
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

Phases 2–3 (Bubbletea `explore`/`menu` palette + `browser` Miller columns) remain
sketches; both reuse this renderer's `SchemaDoc`/style foundation.
