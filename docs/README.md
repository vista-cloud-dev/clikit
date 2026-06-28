# clikit docs

Documentation for **clikit**, the shared CLI convention layer for the
vista-cloud-dev Go toolchain. (The module overview + usage is in the
[repo README](../README.md).)

## Layout

- **`memory/`** — auto-memory (durable lessons only, per the org keep-test).
  - `MEMORY.md` — the per-repo memory index.
  - `clikit-durable-gotchas.md` — durable gotchas + API invariants (styled help,
    pager, the `menu`/`explore` palette).
- **`proposals/`** — repo-local design proposals.
  - `cli-discovery-ux.md` — the command-discovery UX proposal (Phase 1 styled
    help/pager + Phase 2 interactive palette; **complete**, carries its own §6a
    implementation tracker). Pinned by code comments in `help.go`/`explore.go`/
    `palette.go`.

There are currently no `guides/`, `design/`, or `archive/` docs; `modules/` is
not used (clikit is a Go module, not a generated-reference library).
